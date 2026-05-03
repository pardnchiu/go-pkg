package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Border struct {
	Title      string
	TitleAlign int
	Color      tcell.Color
	FocusColor tcell.Color
}

func applyBorder(view tview.Primitive, b *Border) {
	if b == nil {
		return
	}
	box, ok := view.(interface {
		SetBorder(bool) *tview.Box
		SetTitle(string) *tview.Box
		SetTitleAlign(int) *tview.Box
		SetBorderColor(tcell.Color) *tview.Box
		SetFocusFunc(func()) *tview.Box
		SetBlurFunc(func()) *tview.Box
	})
	if !ok {
		return
	}
	box.SetBorder(true)
	box.SetTitle(" " + b.Title + " ")
	box.SetTitleAlign(b.TitleAlign)
	if b.FocusColor == 0 && b.Color == 0 {
		return
	}
	focus := b.FocusColor
	if focus == 0 {
		focus = tcell.ColorYellow
	}
	blur := b.Color
	if blur == 0 {
		blur = tcell.ColorWhite
	}
	box.SetBorderColor(blur)
	box.SetFocusFunc(func() { box.SetBorderColor(focus) })
	box.SetBlurFunc(func() { box.SetBorderColor(blur) })
}
