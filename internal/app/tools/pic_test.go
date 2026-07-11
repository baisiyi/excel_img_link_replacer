package tools

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestProcessImageForExcelNormalizesPNGAndJPEG(t *testing.T) {
	for name, data := range map[string][]byte{
		"png":  testPicturePNG(t),
		"jpeg": testPictureJPEG(t),
	} {
		t.Run(name, func(t *testing.T) {
			out, err := processImageForExcel(data)
			if err != nil {
				t.Fatalf("processImageForExcel returned error: %v", err)
			}
			if !IsPNGMagicNumber(out) {
				t.Fatal("processed image should be png")
			}
			img, _, err := image.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatalf("decode processed image: %v", err)
			}
			if got := img.Bounds().Dx(); got != 360 {
				t.Fatalf("processed image width = %d, want 360", got)
			}
		})
	}
}

func TestProcessImageForExcelRejectsBrokenImage(t *testing.T) {
	if _, err := processImageForExcel([]byte("not an image")); err == nil {
		t.Fatal("expected broken image error")
	}
}

func TestSetCellPictureReturnsExcelErrors(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	if err := SetCellPicture(f, "Sheet1", "A2", "A", 1, testPicturePNG(t)); err != nil {
		t.Fatalf("SetCellPicture valid sheet returned error: %v", err)
	}
	if err := SetCellPicture(f, "Missing", "A2", "A", 1, testPicturePNG(t)); err == nil {
		t.Fatal("expected missing sheet error")
	}
}

func testPicturePNG(t *testing.T) []byte {
	t.Helper()
	img := testImage()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func testPictureJPEG(t *testing.T) []byte {
	t.Helper()
	img := testImage()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	return img
}
