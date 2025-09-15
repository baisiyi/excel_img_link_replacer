package pic

import (
	"pic_tool/internal/app/tools"

	"github.com/xuri/excelize/v2"
)

// SetCellPicture 导出封装，供外部调用以在指定单元格写入图片
func SetCellPicture(f *excelize.File, sheet, cell, col string, row int, imageData []byte) bool {
	return tools.SetCellPicture(f, sheet, cell, col, row, imageData)
}
