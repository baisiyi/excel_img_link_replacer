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

	for _, cell := range []string{"A2", "A3"} {
		value, err := output.GetCellValue("Sheet1", cell)
		if err != nil {
			t.Fatalf("get %s: %v", cell, err)
		}
		if value != "" {
			t.Fatalf("%s value = %q, want empty", cell, value)
		}
		pictures, err := output.GetPictures("Sheet1", cell)
		if err != nil {
			t.Fatalf("get pictures %s: %v", cell, err)
		}
		if len(pictures) == 0 {
			t.Fatalf("expected picture in %s", cell)
		}
	}

	secondValue, err := output.GetCellValue("Second", "A2")
	if err != nil {
		t.Fatalf("get second sheet value: %v", err)
	}
	if secondValue == "" {
		t.Fatal("second sheet should not be processed")
	}
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

func hasProgressStage(progress []ProcessProgress, stage string) bool {
	for _, p := range progress {
		if p.Stage == stage {
			return true
		}
	}
	return false
}
