package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"

	"pic_tool/internal/app/usecase"
)

func loadHeaders(w fyne.Window, path string, headers binding.StringList, selected binding.StringList, defaultHeader string, recreateListFunc func()) {
	headers.Set([]string{})
	selected.Set([]string{})
	go func() {
		hs, err := usecase.ListHeaders(path)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, w)
			})
			return
		}

		// 设置表头数据
		headers.Set(hs)

		// 默认选择 - 只在导入文件时执行一次
		for _, h := range hs {
			if h == defaultHeader {
				selected.Append(h)
				break
			}
		}

		// 重新创建列表，确保选择状态正确
		fyne.Do(func() {
			recreateListFunc()
		})
	}()
}
