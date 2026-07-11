package usecase

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestProcessExcelReportsPartialFailuresAndWritesOutput(t *testing.T) {
	imageBytes := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imageBytes)
		case "/fail.png":
			http.Error(w, "failed", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	inputPath := createWorkbookFixture(t, server.URL)
	var progress []ProcessProgress
	result, err := ProcessExcel(ProcessOptions{
		Path:            inputPath,
		SheetName:       "Sheet1",
		SelectedHeaders: []string{"商品图片链接"},
		Concurrency:     2,
		Timeout:         5 * time.Second,
	}, func(p ProcessProgress) {
		progress = append(progress, p)
	})
	if err != nil {
		t.Fatalf("ProcessExcel returned error: %v", err)
	}

	if result.Total != 4 {
		t.Fatalf("total = %d, want 4", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("success = %d, want 2", result.Success)
	}
	if len(result.Failed) != 2 {
		t.Fatalf("failed = %d, want 2: %#v", len(result.Failed), result.Failed)
	}
	if result.OutputPath == "" {
		t.Fatal("expected output path")
	}
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	if !hasProgressStage(progress, StageDownload) || !hasProgressStage(progress, StageWrite) {
		t.Fatalf("expected download and write progress, got %#v", progress)
	}

	output, err := excelize.OpenFile(result.OutputPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer output.Close()

	assertCellValue(t, output, "Sheet1", "A1", "商品图片链接")
	assertCellValue(t, output, "Sheet1", "B1", "商品图片链接_图片")
	assertCellValue(t, output, "Sheet1", "C1", "Other")
	assertCellValue(t, output, "Sheet1", "A2", server.URL+"/ok.png")
	assertCellValue(t, output, "Sheet1", "A3", server.URL+"/ok.png")
	assertCellValue(t, output, "Sheet1", "A4", "httpx://invalid")
	assertCellValue(t, output, "Sheet1", "A5", server.URL+"/fail.png")
	assertCellValue(t, output, "Sheet1", "C5", "ignored")

	for _, cell := range []string{"B2", "B3"} {
		assertPictureCount(t, output, "Sheet1", cell, true)
	}
	for _, cell := range []string{"A2", "A3", "B4", "B5"} {
		assertPictureCount(t, output, "Sheet1", cell, false)
	}

	secondValue, err := output.GetCellValue("Second", "A2")
	if err != nil {
		t.Fatalf("get second sheet value: %v", err)
	}
	if secondValue == "" {
		t.Fatal("second sheet should not be processed")
	}
}

func TestProcessExcelAddsImageColumnsForMultipleSelectedHeaders(t *testing.T) {
	imageBytes := fixturePNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer server.Close()

	inputPath := createMultiHeaderWorkbookFixture(t, server.URL)
	result, err := ProcessExcel(ProcessOptions{
		Path:            inputPath,
		SheetName:       "Sheet1",
		SelectedHeaders: []string{"主图链接", "详情图链接"},
		Concurrency:     2,
		Timeout:         5 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("ProcessExcel returned error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("success = %d, want 2", result.Success)
	}

	output, err := excelize.OpenFile(result.OutputPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer output.Close()

	assertCellValue(t, output, "Sheet1", "A1", "主图链接")
	assertCellValue(t, output, "Sheet1", "B1", "主图链接_图片")
	assertCellValue(t, output, "Sheet1", "C1", "SKU")
	assertCellValue(t, output, "Sheet1", "D1", "详情图链接")
	assertCellValue(t, output, "Sheet1", "E1", "详情图链接_图片")
	assertCellValue(t, output, "Sheet1", "F1", "备注")
	assertCellValue(t, output, "Sheet1", "A2", server.URL+"/main.png")
	assertCellValue(t, output, "Sheet1", "C2", "sku-1")
	assertCellValue(t, output, "Sheet1", "D2", server.URL+"/detail.png")
	assertCellValue(t, output, "Sheet1", "F2", "keep")
	assertPictureCount(t, output, "Sheet1", "B2", true)
	assertPictureCount(t, output, "Sheet1", "E2", true)
	assertPictureCount(t, output, "Sheet1", "A2", false)
	assertPictureCount(t, output, "Sheet1", "D2", false)
}

func TestUniqueOutputPathAvoidsOverwrite(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "book.xlsx")
	existing := filepath.Join(dir, "book_output.xlsx")
	if err := os.WriteFile(existing, []byte("exists"), 0644); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	got := uniqueOutputPath(input)
	want := filepath.Join(dir, "book_output_1.xlsx")
	if got != want {
		t.Fatalf("uniqueOutputPath = %q, want %q", got, want)
	}
}

func createWorkbookFixture(t *testing.T, serverURL string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "book.xlsx")

	f := excelize.NewFile()
	defer f.Close()
	if _, err := f.NewSheet("Second"); err != nil {
		t.Fatalf("new sheet: %v", err)
	}

	values := map[string]string{
		"A1": "商品图片链接",
		"B1": "Other",
		"A2": serverURL + "/ok.png",
		"A3": serverURL + "/ok.png",
		"A4": "httpx://invalid",
		"A5": serverURL + "/fail.png",
		"B5": "ignored",
	}
	for cell, value := range values {
		if err := f.SetCellValue("Sheet1", cell, value); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}
	if err := f.SetCellValue("Second", "A1", "商品图片链接"); err != nil {
		t.Fatalf("set second header: %v", err)
	}
	if err := f.SetCellValue("Second", "A2", serverURL+"/ok.png"); err != nil {
		t.Fatalf("set second value: %v", err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save fixture: %v", err)
	}
	return path
}

func createMultiHeaderWorkbookFixture(t *testing.T, serverURL string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	values := map[string]string{
		"A1": "主图链接",
		"B1": "SKU",
		"C1": "详情图链接",
		"D1": "备注",
		"A2": serverURL + "/main.png",
		"B2": "sku-1",
		"C2": serverURL + "/detail.png",
		"D2": "keep",
	}
	for cell, value := range values {
		if err := f.SetCellValue("Sheet1", cell, value); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save fixture: %v", err)
	}
	return path
}

func fixturePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func assertCellValue(t *testing.T, f *excelize.File, sheet, cell, want string) {
	t.Helper()
	got, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("get %s: %v", cell, err)
	}
	if got != want {
		t.Fatalf("%s value = %q, want %q", cell, got, want)
	}
}

func assertPictureCount(t *testing.T, f *excelize.File, sheet, cell string, wantPicture bool) {
	t.Helper()
	pictures, err := f.GetPictures(sheet, cell)
	if err != nil {
		t.Fatalf("get pictures %s: %v", cell, err)
	}
	if wantPicture && len(pictures) == 0 {
		t.Fatalf("expected picture in %s", cell)
	}
	if !wantPicture && len(pictures) != 0 {
		t.Fatalf("expected no picture in %s, got %d", cell, len(pictures))
	}
}

func hasProgressStage(progress []ProcessProgress, stage string) bool {
	for _, p := range progress {
		if p.Stage == stage {
			return true
		}
	}
	return false
}
