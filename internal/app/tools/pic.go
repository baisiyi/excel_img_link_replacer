package tools

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"

	"github.com/xuri/excelize/v2"
	"golang.org/x/image/draw"
)

func SetCellPicture(f *excelize.File, sheet, cell, col string, row int, imageData []byte) bool {
	if len(imageData) == 0 {
		return false
	}

	var (
		cellWidthChars   = 8.0  // Excel列宽（字符）
		cellHeightPoints = 60.0 // Excel行高（磅）
	)

	processedImage, err := processImageForExcel(imageData)
	if err != nil {
		return false
	}
	pic := &excelize.Picture{
		Extension: ".png",
		File:      processedImage,
		Format: &excelize.GraphicOptions{
			Positioning:         "oneCell", // 单单元格定位
			LockAspectRatio:     true,      // 锁定宽高比，避免变形
			AutoFit:             true,      // 关闭自动适应，使用手动缩放
			AutoFitIgnoreAspect: false,
		},
	}
	err = f.SetColWidth(sheet, col, col, cellWidthChars)
	err = f.SetRowHeight(sheet, row+1, cellHeightPoints)
	if err = f.AddPictureFromBytes(sheet, cell, pic); err != nil {
		return false
	}
	return true
}

func processImageForExcel(imageData []byte) ([]byte, error) {

	var (
		targetWidthInches = 1.0 // 目标宽度
		DPI               = 300 // 标准显示DPI
	)

	if len(imageData) == 0 {
		return nil, errors.New("empty image data")
	}

	// 解码图片
	src, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// 获取原始尺寸
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// 计算目标宽度
	targetWidthPixels := int(targetWidthInches * float64(DPI)) // 63像素
	// 计算目标高度（保持宽高比）
	targetHeight := int(float64(srcHeight) * float64(targetWidthPixels) / float64(srcWidth))

	// 创建目标图片
	dst := image.NewRGBA(image.Rect(0, 0, targetWidthPixels, targetHeight))

	// 使用高质量缩放
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	// 编码为png
	var buf bytes.Buffer
	err = png.Encode(&buf, dst)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}

	return buf.Bytes(), nil
}
