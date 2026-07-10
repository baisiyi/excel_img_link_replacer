package usecase

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func ListSheets(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// ListHeaders 返回指定工作表首行表头。
func ListHeaders(path, sheet string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return []string{}, nil
	}
	if sheet == "" {
		sheet = sheets[0]
	}
	if !contains(sheets, sheet) {
		return nil, fmt.Errorf("工作表不存在: %s", sheet)
	}
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return []string{}, err
	}
	return rows[0], nil
}
