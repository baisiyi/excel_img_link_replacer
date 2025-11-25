package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"pic_tool/internal/app/ui"
)

func main() {
	a := app.NewWithID("pic_tool.desktop")
	a.Settings().SetTheme(theme.LightTheme())
	w := a.NewWindow("Excel图片链接替换工具")
	w.Resize(ui.DefaultWindowSize())

	content := ui.BuildUI(a, w)
	w.SetContent(content)

	w.SetCloseIntercept(func() {
		// 可在此做清理
		w.Close()
	})

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %v\n", r)
		}
	}()

	w.ShowAndRun()
}
