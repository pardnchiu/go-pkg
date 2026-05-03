package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Border struct {
	Title      string
	TitleAlign int
}

func BindFocusBorder(views ...tview.Primitive) {
	for _, v := range views {
		box, ok := v.(interface {
			SetBorderColor(tcell.Color) *tview.Box
			SetFocusFunc(func()) *tview.Box
			SetBlurFunc(func()) *tview.Box
		})
		if !ok {
			continue
		}
		box.SetFocusFunc(func() { box.SetBorderColor(tcell.ColorYellow) })
		box.SetBlurFunc(func() { box.SetBorderColor(tcell.ColorWhite) })
	}
}
