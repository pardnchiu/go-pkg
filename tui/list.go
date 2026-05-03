package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type List struct {
	MainTextColor      tcell.Color
	ShowSecondaryText  bool
	SecondaryTextColor tcell.Color
	Border             *Border
}

func NewList(config *List) *tview.List {
	view := tview.NewList().
		ShowSecondaryText(config.ShowSecondaryText).
		SetMainTextColor(config.MainTextColor).
		SetSecondaryTextColor(config.SecondaryTextColor)
	applyBorder(view, config.Border)
	return view
}
