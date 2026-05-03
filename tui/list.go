package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type List struct {
	MainTextColor      tcell.Color
	ShowSecondaryText  bool
	SecondaryTextColor tcell.Color
}

func NewList(config *List, border *Border) *tview.List {
	view := tview.NewList().
		ShowSecondaryText(config.ShowSecondaryText).
		SetMainTextColor(config.MainTextColor).
		SetSecondaryTextColor(config.SecondaryTextColor)
	if border != nil {
		view.SetBorder(true).
			SetTitle(" " + border.Title + " ").
			SetTitleAlign(border.TitleAlign)
	}
	return view
}
