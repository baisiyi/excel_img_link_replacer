package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"pic_tool/internal/app/tools"
	"pic_tool/internal/app/usecase"
)

func DefaultWindowSize() fyne.Size { return fyne.NewSize(960, 640) }

// updateStatusText 更新状态文本，支持可点击的文件路径
func updateStatusText(container *fyne.Container, message string, filePath string) {
	container.RemoveAll()

	if filePath != "" {
		// 如果有文件路径，创建可点击的链接
		statusLabel := canvas.NewText(message, nil)
		statusLabel.TextStyle = fyne.TextStyle{}

		// 创建文件路径按钮
		pathButton := widget.NewButton(filePath, func() {
			if err := tools.OpenFileDirectory(filePath); err != nil {
				// 如果打开失败，尝试只打开目录
				dir := filepath.Dir(filePath)
				if err := tools.OpenDirectory(dir); err != nil {
					// 如果还是失败，显示错误信息
					dialog.ShowError(fmt.Errorf("无法打开文件目录: %v", err), nil)
				}
			}
		})
		pathButton.Importance = widget.LowImportance

		container.Add(statusLabel)
		container.Add(pathButton)
	} else {
		// 普通文本消息
		statusLabel := canvas.NewText(message, nil)
		statusLabel.TextStyle = fyne.TextStyle{}
		container.Add(statusLabel)
	}

	container.Refresh()
}

// 全局变量，用于在回调中访问
var (
	globalRunBtn     *widget.Button
	globalDropObj    *widget.Card
	globalChooseBtn  *widget.Button
	globalProgress   *widget.ProgressBar
	globalStatusText *fyne.Container
)

func BuildUI(a fyne.App, w fyne.Window) fyne.CanvasObject {
	// 绑定状态
	selectedFile := binding.NewString()
	selectedFile.Set("")

	headers := binding.NewStringList()
	headers.Set([]string{})

	selectedHeaders := binding.NewStringList()
	selectedHeaders.Set([]string{})

	progress := widget.NewProgressBar()
	progress.Hide()
	globalProgress = progress

	// 创建状态容器，初始为空
	statusText := container.NewHBox()
	globalStatusText = statusText

	// 默认勾选项
	defaultHeader := "商品图片链接"

	// 多选列表 - 使用容器+复选框的方式，避免滚动时状态重置
	var headerList *container.Scroll
	var headerCheckboxes []*widget.Check

	// 创建表头列表的函数
	createHeaderList := func() *container.Scroll {
		items, _ := headers.Get()
		headerCheckboxes = make([]*widget.Check, len(items))

		checkboxes := make([]fyne.CanvasObject, len(items))
		for i, header := range items {
			chk := widget.NewCheck(header, func(checked bool) {})
			headerCheckboxes[i] = chk

			// 设置初始选中状态
			selected, _ := selectedHeaders.Get()
			chk.SetChecked(contains(selected, header))

			// 设置点击事件
			chk.OnChanged = func(checked bool) {
				cur, _ := selectedHeaders.Get()
				if checked {
					if !contains(cur, header) {
						selectedHeaders.Append(header)
					}
				} else {
					removeString(&cur, header)
					selectedHeaders.Set(cur)
				}
			}

			checkboxes[i] = chk
		}

		return container.NewScroll(container.NewVBox(checkboxes...))
	}

	headerList = createHeaderList()

	// 处理按钮
	runBtn := widget.NewButton("开始处理并生成", func() {
		file, _ := selectedFile.Get()
		if file == "" {
			dialog.ShowInformation("提示", "请先选择 Excel 文件", w)
			return
		}
		sel, _ := selectedHeaders.Get()
		if len(sel) == 0 {
			dialog.ShowInformation("提示", "请至少选择一个表头", w)
			return
		}
		progress.SetValue(0)
		progress.Show()
		updateStatusText(statusText, "处理中...", "")

		// 异步执行
		go func() {
			outPath, err := usecase.ProcessExcel(file, sel, func(done, total int) {
				// UI 更新必须在主线程
				fyne.Do(func() {
					progress.SetValue(float64(done) / float64(total))
				})
			})

			// 所有 UI 操作都包装在 fyne.Do 中
			fyne.Do(func() {
				if err != nil {
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "失败", Content: err.Error()})
					updateStatusText(statusText, fmt.Sprintf("失败: %v", err), "")
				} else {
					fyne.CurrentApp().SendNotification(&fyne.Notification{Title: "完成", Content: filepath.Base(outPath)})
					updateStatusText(statusText, "完成，输出: ", outPath)
				}
				progress.Hide()
			})
		}()
	})
	globalRunBtn = runBtn

	// 文件选择按钮
	var chooseBtn *widget.Button
	chooseBtn = widget.NewButton("选择 Excel 文件", func() {
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
			_ = selectedFile.Set(uri.Path())
			loadHeaders(w, uri.Path(), headers, selectedHeaders, defaultHeader, func() {
				// 重新创建列表
				headerList = createHeaderList()
				// 更新右侧面板
				right := container.NewBorder(widget.NewLabel("选择需要处理的表头"), nil, nil, nil, headerList)
				left := container.NewBorder(nil, container.NewVBox(globalRunBtn, globalProgress, globalStatusText), nil, nil,
					container.NewVBox(globalDropObj, globalChooseBtn),
				)
				w.SetContent(container.NewHSplit(left, right))
			})
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx", ".xlsm"}))
		// 设置弹窗为可调整大小
		fd.Resize(fyne.NewSize(800, 600))
		fd.Show()
	})
	globalChooseBtn = chooseBtn

	// 拖拽区
	var dropLabel *canvas.Text
	var drop *fyne.Container
	var dropObj *widget.Card
	dropLabel = canvas.NewText("拖拽 Excel 到此处，或点击下方按钮选择", nil)
	drop = container.NewMax(container.NewCenter(dropLabel))
	dropObj = widget.NewCard("上传文件", "仅支持 .xlsx/.xlsm", drop)
	globalDropObj = dropObj
	// 窗口级拖拽处理
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		u := uris[0]
		if !isExcel(u.Name()) {
			dialog.ShowInformation("提示", "仅支持 .xlsx/.xlsm", w)
			return
		}
		_ = selectedFile.Set(u.Path())
		loadHeaders(w, u.Path(), headers, selectedHeaders, defaultHeader, func() {
			// 重新创建列表
			headerList = createHeaderList()
			// 更新右侧面板
			right := container.NewBorder(widget.NewLabel("选择需要处理的表头"), nil, nil, nil, headerList)
			left := container.NewBorder(nil, container.NewVBox(globalRunBtn, globalProgress, globalStatusText), nil, nil,
				container.NewVBox(globalDropObj, globalChooseBtn),
			)
			w.SetContent(container.NewHSplit(left, right))
		})
	})

	// 布局
	left := container.NewBorder(nil, container.NewVBox(runBtn, progress, statusText), nil, nil,
		container.NewVBox(dropObj, chooseBtn),
	)
	right := container.NewBorder(widget.NewLabel("选择需要处理的表头"), nil, nil, nil, headerList)
	return container.NewHSplit(left, right)
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
