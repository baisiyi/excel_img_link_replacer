package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"pic_tool/internal/app/tools"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ProcessExcel
func ProcessExcel(path string, selectedHeaders []string, progressCb func(int, int)) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("excel 无工作表")
	}
	sheet := sheets[0]

	// 读首行表头
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("首行为空")
	}
	header := rows[0]

	// 选中列索引
	headerIndex := make(map[int]string)
	for colIdx, colName := range header {
		if contains(selectedHeaders, strings.TrimSpace(colName)) {
			headerIndex[colIdx] = colName
		}
	}
	if len(headerIndex) == 0 {
		return "", fmt.Errorf("未找到选中表头")
	}

	// 收集所有 URL
	type cellPos struct{ ColIdx, RowIdx int }
	urlByPos := make(map[cellPos]string)
	urls := make([]string, 0)
	for r := 1; r < len(rows); r++ { // 从第二行
		row := rows[r]
		for c := range headerIndex {
			if c < len(row) {
				u := strings.TrimSpace(row[c])
				if u != "" && strings.HasPrefix(u, "http") {
					urlByPos[cellPos{ColIdx: c, RowIdx: r}] = u
					urls = append(urls, u)
				}
			}
		}
	}
	if len(urls) == 0 {
		return "", fmt.Errorf("未找到可用链接")
	}

	if progressCb == nil {
		progressCb = func(int, int) {}
	}
	progressCb(0, len(urls))

	// 批量下载
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	bodies, err := tools.BatchGetCDNImageBytes(ctx, urls, 8)
	if err != nil {
		return "", err
	}

	// 写入图片
	done := 0
	for pos, u := range urlByPos {
		img, ok := bodies[u]
		if !ok || len(img) == 0 {
			done++
			progressCb(done, len(urls))
			continue
		}
		colName, _ := excelize.ColumnNumberToName(pos.ColIdx + 1)
		cell, _ := excelize.CoordinatesToCellName(pos.ColIdx+1, pos.RowIdx+1)
		_ = f.SetCellValue(sheet, cell, "")
		tools.SetCellPicture(f, sheet, cell, colName, pos.RowIdx, img)
		done++
		progressCb(done, len(urls))
	}

	// 输出副本
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	out := filepath.Join(filepath.Dir(path), fmt.Sprintf("%s_output%s", base, ext))
	if err := f.SaveAs(out); err != nil {
		return "", err
	}
	return out, nil
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
