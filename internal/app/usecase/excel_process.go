package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"pic_tool/internal/app/tools"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	defaultConcurrency = 8
	defaultTimeout     = 120 * time.Second

	StageValidate = "validate"
	StageDownload = "download"
	StageWrite    = "write"
)

type ProcessOptions struct {
	Path            string
	SheetName       string
	SelectedHeaders []string
	Concurrency     int
	Timeout         time.Duration
}

type ProcessResult struct {
	OutputPath string
	Total      int
	Success    int
	Failed     []CellFailure
}

type CellFailure struct {
	Sheet string
	Cell  string
	URL   string
	Stage string
	Err   error
}

type ProcessProgress struct {
	Stage string
	Done  int
	Total int
}

// ProcessExcel replaces image links in the selected sheet and returns a
// structured report. Individual cell failures do not abort the whole run.
func ProcessExcel(opts ProcessOptions, progressCb func(ProcessProgress)) (ProcessResult, error) {
	var result ProcessResult
	if opts.Path == "" {
		return result, errors.New("excel 路径为空")
	}
	if len(opts.SelectedHeaders) == 0 {
		return result, errors.New("未选择表头")
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if progressCb == nil {
		progressCb = func(ProcessProgress) {}
	}

	f, err := excelize.OpenFile(opts.Path)
	if err != nil {
		return result, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return result, fmt.Errorf("excel 无工作表")
	}
	sheet := opts.SheetName
	if sheet == "" {
		sheet = sheets[0]
	}
	if !contains(sheets, sheet) {
		return result, fmt.Errorf("工作表不存在: %s", sheet)
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return result, err
	}
	if len(rows) == 0 {
		return result, fmt.Errorf("首行为空")
	}

	headerIndex := selectedHeaderIndexes(rows[0], opts.SelectedHeaders)
	if len(headerIndex) == 0 {
		return result, fmt.Errorf("未找到选中表头")
	}

	imageColumns := imageColumnIndexes(headerIndex)
	jobs, failures := collectImageJobs(sheet, rows, headerIndex, imageColumns)
	result.Failed = append(result.Failed, failures...)
	result.Total = len(jobs)
	if result.Total == 0 {
		return result, fmt.Errorf("未找到可处理链接")
	}

	if err := insertImageColumns(f, sheet, headerIndex, imageColumns); err != nil {
		return result, err
	}

	validURLs := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if job.valid {
			validURLs = append(validURLs, job.url)
		}
	}

	progressCb(ProcessProgress{Stage: StageDownload, Done: 0, Total: len(tools.Unique(validURLs))})
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	downloaded, err := tools.BatchGetCDNImageBytes(ctx, validURLs, opts.Concurrency)
	if err != nil {
		return result, err
	}
	progressCb(ProcessProgress{Stage: StageDownload, Done: len(downloaded.Images) + len(downloaded.Failures), Total: len(tools.Unique(validURLs))})

	writeDone := 0
	progressCb(ProcessProgress{Stage: StageWrite, Done: writeDone, Total: result.Total})
	for _, job := range jobs {
		if !job.valid {
			writeDone++
			progressCb(ProcessProgress{Stage: StageWrite, Done: writeDone, Total: result.Total})
			continue
		}

		img, ok := downloaded.Images[job.url]
		if !ok || len(img) == 0 {
			err := downloaded.Failures[job.url]
			if err == nil {
				err = errors.New("download failed")
			}
			result.Failed = append(result.Failed, CellFailure{
				Sheet: sheet,
				Cell:  job.cell,
				URL:   job.url,
				Stage: StageDownload,
				Err:   err,
			})
			writeDone++
			progressCb(ProcessProgress{Stage: StageWrite, Done: writeDone, Total: result.Total})
			continue
		}

		if err := tools.SetCellPicture(f, sheet, job.pictureCell, job.pictureColName, job.rowIdx, img); err != nil {
			result.Failed = append(result.Failed, CellFailure{
				Sheet: sheet,
				Cell:  job.cell,
				URL:   job.url,
				Stage: StageWrite,
				Err:   err,
			})
			writeDone++
			progressCb(ProcessProgress{Stage: StageWrite, Done: writeDone, Total: result.Total})
			continue
		}
		result.Success++
		writeDone++
		progressCb(ProcessProgress{Stage: StageWrite, Done: writeDone, Total: result.Total})
	}

	result.OutputPath = uniqueOutputPath(opts.Path)
	if err := f.SaveAs(result.OutputPath); err != nil {
		return result, err
	}
	return result, nil
}

type imageJob struct {
	cell           string
	pictureCell    string
	pictureColName string
	rowIdx         int
	url            string
	valid          bool
}

func selectedHeaderIndexes(header []string, selectedHeaders []string) map[int]string {
	selected := make(map[string]bool, len(selectedHeaders))
	for _, h := range selectedHeaders {
		selected[strings.TrimSpace(h)] = true
	}

	headerIndex := make(map[int]string)
	for colIdx, colName := range header {
		trimmed := strings.TrimSpace(colName)
		if selected[trimmed] {
			headerIndex[colIdx] = trimmed
		}
	}
	return headerIndex
}

func collectImageJobs(sheet string, rows [][]string, headerIndex map[int]string, imageColumns map[int]int) ([]imageJob, []CellFailure) {
	jobs := make([]imageJob, 0)
	failures := make([]CellFailure, 0)
	for r := 1; r < len(rows); r++ {
		row := rows[r]
		for _, c := range sortedHeaderColumns(headerIndex) {
			if c >= len(row) {
				continue
			}
			rawURL := strings.TrimSpace(row[c])
			if rawURL == "" {
				continue
			}
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				continue
			}
			pictureColIdx, ok := imageColumns[c]
			if !ok {
				continue
			}
			pictureColName, err := excelize.ColumnNumberToName(pictureColIdx + 1)
			if err != nil {
				continue
			}
			pictureCell, err := excelize.CoordinatesToCellName(pictureColIdx+1, r+1)
			if err != nil {
				continue
			}
			job := imageJob{
				cell:           cell,
				pictureCell:    pictureCell,
				pictureColName: pictureColName,
				rowIdx:         r,
				url:            rawURL,
				valid:          tools.IsHTTPImageURL(rawURL),
			}
			jobs = append(jobs, job)
			if !job.valid {
				failures = append(failures, CellFailure{
					Sheet: sheet,
					Cell:  cell,
					URL:   rawURL,
					Stage: StageValidate,
					Err:   errors.New("invalid image url"),
				})
			}
		}
	}
	return jobs, failures
}

func imageColumnIndexes(headerIndex map[int]string) map[int]int {
	selectedCols := sortedHeaderColumns(headerIndex)
	imageColumns := make(map[int]int, len(selectedCols))
	for _, colIdx := range selectedCols {
		insertionsBeforeOrAt := 0
		for _, selectedCol := range selectedCols {
			if selectedCol <= colIdx {
				insertionsBeforeOrAt++
			}
		}
		imageColumns[colIdx] = colIdx + insertionsBeforeOrAt
	}
	return imageColumns
}

func insertImageColumns(f *excelize.File, sheet string, headerIndex map[int]string, imageColumns map[int]int) error {
	selectedCols := sortedHeaderColumns(headerIndex)
	for i := len(selectedCols) - 1; i >= 0; i-- {
		colIdx := selectedCols[i]
		insertBeforeCol, err := excelize.ColumnNumberToName(colIdx + 2)
		if err != nil {
			return err
		}
		if err := f.InsertCols(sheet, insertBeforeCol, 1); err != nil {
			return fmt.Errorf("insert image column: %w", err)
		}
	}

	for originalColIdx, imageColIdx := range imageColumns {
		headerCell, err := excelize.CoordinatesToCellName(imageColIdx+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, headerCell, headerIndex[originalColIdx]+"_图片"); err != nil {
			return fmt.Errorf("set image column header: %w", err)
		}
	}
	return nil
}

func sortedHeaderColumns(headerIndex map[int]string) []int {
	cols := make([]int, 0, len(headerIndex))
	for colIdx := range headerIndex {
		cols = append(cols, colIdx)
	}
	sort.Ints(cols)
	return cols
}

func uniqueOutputPath(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	dir := filepath.Dir(path)
	out := filepath.Join(dir, fmt.Sprintf("%s_output%s", base, ext))
	if !fileExists(out) {
		return out
	}
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_output_%d%s", base, i, ext))
		if !fileExists(candidate) {
			return candidate
		}
	}
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
