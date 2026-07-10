package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"pic_tool/internal/app/tools"
	"pic_tool/internal/app/usecase"
)

func DefaultWindowSize() fyne.Size { return fyne.NewSize(960, 640) }

func updateStatusText(status *fyne.Container, message string, filePath string) {
	status.RemoveAll()
	label := widget.NewLabel(message)
	label.Wrapping = fyne.TextWrapWord
	status.Add(label)

	if filePath != "" {
		pathButton := widget.NewButton(filePath, func() {
			if err := tools.OpenFileDirectory(filePath); err != nil {
				dir := filepath.Dir(filePath)
				if err := tools.OpenDirectory(dir); err != nil {
					dialog.ShowError(fmt.Errorf("无法打开文件目录: %v", err), nil)
				}
			}
		})
		pathButton.Importance = widget.LowImportance
		status.Add(pathButton)
	}
	status.Refresh()
}

func BuildUI(a fyne.App, w fyne.Window) fyne.CanvasObject {
	selectedFile := binding.NewString()
	_ = selectedFile.Set("")

	headers := binding.NewStringList()
	_ = headers.Set([]string{})

	selectedHeaders := binding.NewStringList()
	_ = selectedHeaders.Set([]string{})

	defaultHeader := "商品图片链接"
	isBusy := false

	progress := widget.NewProgressBar()
	progress.Hide()

	statusText := container.NewVBox()
	dropLabel := widget.NewLabel("拖拽 Excel 到此处，或点击下方按钮选择")
	dropLabel.Alignment = fyne.TextAlignCenter
	dropObj := widget.NewCard("上传文件", "仅支持 .xlsx/.xlsm", container.NewCenter(dropLabel))

	sheetSelect := widget.NewSelect([]string{}, nil)
	sheetSelect.PlaceHolder = "选择工作表"
	sheetSelect.Disable()

	headerSearch := widget.NewEntry()
	headerSearch.SetPlaceHolder("搜索表头")
	headerSearch.Disable()

	headerHolder := container.NewMax()
	var runBtn *widget.Button
	var chooseBtn *widget.Button

	setBusy := func(busy bool) {
		isBusy = busy
		if busy {
			runBtn.Disable()
			chooseBtn.Disable()
			sheetSelect.Disable()
			headerSearch.Disable()
			return
		}
		runBtn.Enable()
		chooseBtn.Enable()
		if len(sheetSelect.Options) > 0 {
			sheetSelect.Enable()
		}
		items, _ := headers.Get()
		if len(items) > 0 {
			headerSearch.Enable()
		}
	}

	createHeaderList := func() *container.Scroll {
		items, _ := headers.Get()
		filtered := filterHeaders(items, headerSearch.Text)
		checkboxes := make([]fyne.CanvasObject, 0, len(filtered))
		selected, _ := selectedHeaders.Get()

		if len(items) > 0 && len(filtered) == 0 {
			return container.NewScroll(container.NewVBox(widget.NewLabel("没有匹配的表头")))
		}

		for _, header := range filtered {
			header := header
			chk := widget.NewCheck(header, nil)
			chk.SetChecked(contains(selected, header))
			chk.OnChanged = func(checked bool) {
				cur, _ := selectedHeaders.Get()
				if checked {
					if !contains(cur, header) {
						_ = selectedHeaders.Append(header)
					}
					return
				}
				removeString(&cur, header)
				_ = selectedHeaders.Set(cur)
			}
			checkboxes = append(checkboxes, chk)
		}
		return container.NewScroll(container.NewVBox(checkboxes...))
	}

	refreshHeaderList := func() {
		headerHolder.RemoveAll()
		headerHolder.Add(createHeaderList())
		headerHolder.Refresh()
	}
	refreshHeaderList()

	headerSearch.OnChanged = func(string) {
		refreshHeaderList()
	}

	loadSheetHeaders := func(path, sheet string) {
		headerSearch.SetText("")
		headerSearch.Disable()
		updateStatusText(statusText, fmt.Sprintf("正在读取工作表: %s", sheet), "")
		loadHeaders(w, path, sheet, headers, selectedHeaders, defaultHeader, func() {
			refreshHeaderList()
			items, _ := headers.Get()
			if len(items) > 0 {
				headerSearch.Enable()
			}
			updateStatusText(statusText, fmt.Sprintf("已加载工作表: %s", sheet), "")
		})
	}

	loadWorkbook := func(path string) {
		if isBusy {
			return
		}
		_ = selectedFile.Set(path)
		dropLabel.SetText(filepath.Base(path))
		updateStatusText(statusText, "正在读取工作簿...", "")
		sheetSelect.Disable()
		headerSearch.SetText("")
		headerSearch.Disable()

		go func() {
			sheets, err := usecase.ListSheets(path)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					updateStatusText(statusText, fmt.Sprintf("读取失败: %v", err), "")
					return
				}
				if len(sheets) == 0 {
					updateStatusText(statusText, "读取失败: Excel 无工作表", "")
					return
				}
				sheetSelect.Options = sheets
				sheetSelect.Refresh()
				sheetSelect.Enable()
				sheetSelect.SetSelected(sheets[0])
			})
		}()
	}

	sheetSelect.OnChanged = func(sheet string) {
		if sheet == "" || isBusy {
			return
		}
		file, _ := selectedFile.Get()
		if file == "" {
			return
		}
		loadSheetHeaders(file, sheet)
	}

	runBtn = widget.NewButton("开始处理并生成", func() {
		file, _ := selectedFile.Get()
		if file == "" {
			dialog.ShowInformation("提示", "请先选择 Excel 文件", w)
			return
		}
		if sheetSelect.Selected == "" {
			dialog.ShowInformation("提示", "请先选择工作表", w)
			return
		}
		sel, _ := selectedHeaders.Get()
		if len(sel) == 0 {
			dialog.ShowInformation("提示", "请至少选择一个表头", w)
			return
		}
		sheet := sheetSelect.Selected

		progress.SetValue(0)
		progress.Show()
		updateStatusText(statusText, "处理中...", "")
		setBusy(true)

		go func() {
			result, err := usecase.ProcessExcel(usecase.ProcessOptions{
				Path:            file,
				SheetName:       sheet,
				SelectedHeaders: sel,
				Concurrency:     8,
				Timeout:         120 * time.Second,
			}, func(p usecase.ProcessProgress) {
				fyne.Do(func() {
					if p.Total > 0 {
						progress.SetValue(float64(p.Done) / float64(p.Total))
					}
					updateStatusText(statusText, progressMessage(p), "")
				})
			})

			fyne.Do(func() {
				defer setBusy(false)
				defer progress.Hide()
				if err != nil {
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "失败", Content: err.Error()})
					updateStatusText(statusText, fmt.Sprintf("失败: %v", err), "")
					return
				}
				title := "完成"
				if len(result.Failed) > 0 {
					title = "部分完成"
				}
				fyne.CurrentApp().SendNotification(&fyne.Notification{Title: title, Content: filepath.Base(result.OutputPath)})
				updateStatusText(statusText, resultSummary(result), result.OutputPath)
			})
		}()
	})

	chooseBtn = widget.NewButton("选择 Excel 文件", func() {
		if isBusy {
			return
		}
		fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			uri := rc.URI()
			if uri == nil {
				return
			}
			_ = rc.Close()
			if !isExcel(uri.Name()) {
				dialog.ShowInformation("提示", "仅支持 .xlsx/.xlsm", w)
				return
			}
			loadWorkbook(uri.Path())
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx", ".xlsm"}))
		fd.Resize(fyne.NewSize(800, 600))
		fd.Show()
	})

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if isBusy || len(uris) == 0 {
			return
		}
		u := uris[0]
		if !isExcel(u.Name()) {
			dialog.ShowInformation("提示", "仅支持 .xlsx/.xlsm", w)
			return
		}
		loadWorkbook(u.Path())
	})

	left := container.NewBorder(nil, container.NewVBox(runBtn, progress, statusText), nil, nil,
		container.NewVBox(dropObj, chooseBtn),
	)
	right := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("选择工作表"),
			sheetSelect,
			container.NewBorder(nil, nil, widget.NewLabel("选择需要处理的表头"), nil, headerSearch),
		),
		nil,
		nil,
		nil,
		headerHolder,
	)
	return container.NewHSplit(left, right)
}

func progressMessage(p usecase.ProcessProgress) string {
	switch p.Stage {
	case usecase.StageDownload:
		return fmt.Sprintf("下载中: %d/%d", p.Done, p.Total)
	case usecase.StageWrite:
		return fmt.Sprintf("写入中: %d/%d", p.Done, p.Total)
	default:
		return "处理中..."
	}
}

func resultSummary(result usecase.ProcessResult) string {
	lines := []string{
		fmt.Sprintf("处理完成: 共 %d 个单元格，成功 %d 个，失败 %d 个。", result.Total, result.Success, len(result.Failed)),
		"输出:",
	}
	if len(result.Failed) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "失败摘要:")
	limit := len(result.Failed)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		failure := result.Failed[i]
		lines = append(lines, fmt.Sprintf("- %s %s: %v", failure.Cell, failure.Stage, failure.Err))
	}
	if len(result.Failed) > limit {
		lines = append(lines, fmt.Sprintf("还有 %d 个失败未显示。", len(result.Failed)-limit))
	}
	return strings.Join(lines, "\n")
}

func filterHeaders(headers []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return headers
	}
	filtered := make([]string, 0, len(headers))
	for _, header := range headers {
		if strings.Contains(strings.ToLower(header), query) {
			filtered = append(filtered, header)
		}
	}
	return filtered
}

func isExcel(name string) bool {
	low := strings.ToLower(name)
	return strings.HasSuffix(low, ".xlsx") || strings.HasSuffix(low, ".xlsm")
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(arr *[]string, s string) {
	a := *arr
	out := make([]string, 0, len(a))
	for _, v := range a {
		if v != s {
			out = append(out, v)
		}
	}
	*arr = out
}
