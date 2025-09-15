package usecase

import (
	"github.com/xuri/excelize/v2"
)

// ListHeaders 返回首行表头
func ListHeaders(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return []string{}, nil
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) == 0 {
		return []string{}, err
	}
	return rows[0], nil
}
