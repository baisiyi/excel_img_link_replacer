package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"

	"pic_tool/internal/app/usecase"
)

func loadHeaders(w fyne.Window, path, sheet string, headers binding.StringList, selected binding.StringList, defaultHeader string, recreateListFunc func()) {
	_ = headers.Set([]string{})
	_ = selected.Set([]string{})
	go func() {
		hs, err := usecase.ListHeaders(path, sheet)
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(err, w)
			})
			return
		}

		fyne.Do(func() {
			_ = headers.Set(hs)
			for _, h := range hs {
				if h == defaultHeader {
					_ = selected.Append(h)
					break
				}
			}
			recreateListFunc()
		})
	}()
}
