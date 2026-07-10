package ui

import (
	"errors"
	"strings"
	"testing"

	"pic_tool/internal/app/usecase"
)

func TestProgressMessage(t *testing.T) {
	if got := progressMessage(usecase.ProcessProgress{Stage: usecase.StageDownload, Done: 1, Total: 3}); got != "下载中: 1/3" {
		t.Fatalf("download progress = %q", got)
	}
	if got := progressMessage(usecase.ProcessProgress{Stage: usecase.StageWrite, Done: 2, Total: 4}); got != "写入中: 2/4" {
		t.Fatalf("write progress = %q", got)
	}
}

func TestResultSummaryIncludesFailureDetails(t *testing.T) {
	summary := resultSummary(usecase.ProcessResult{
		Total:   2,
		Success: 1,
		Failed: []usecase.CellFailure{{
			Cell:  "A2",
			Stage: usecase.StageDownload,
			Err:   errors.New("bad status: 500"),
		}},
	})
	for _, want := range []string{"成功 1 个", "失败 1 个", "A2 download: bad status: 500"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary %q missing %q", summary, want)
		}
	}
}

func TestFilterHeaders(t *testing.T) {
	headers := []string{"订单ID", "商品图片链接", "buyerStatus", "平台发货状态"}

	if got := filterHeaders(headers, ""); len(got) != len(headers) {
		t.Fatalf("empty query returned %d headers, want %d", len(got), len(headers))
	}
	if got := filterHeaders(headers, "图片"); len(got) != 1 || got[0] != "商品图片链接" {
		t.Fatalf("chinese query result = %#v", got)
	}
	if got := filterHeaders(headers, "STATUS"); len(got) != 1 || got[0] != "buyerStatus" {
		t.Fatalf("case-insensitive query result = %#v", got)
	}
	if got := filterHeaders(headers, "不存在"); len(got) != 0 {
		t.Fatalf("missing query result = %#v, want empty", got)
	}
}

func TestSelectionHelpers(t *testing.T) {
	values := []string{"A", "B"}
	if !contains(values, "A") {
		t.Fatal("contains should find existing value")
	}
	removeString(&values, "A")
	if contains(values, "A") || len(values) != 1 {
		t.Fatalf("removeString failed: %#v", values)
	}
	if !isExcel("book.XLSX") || isExcel("book.csv") {
		t.Fatal("isExcel extension check failed")
	}
}
