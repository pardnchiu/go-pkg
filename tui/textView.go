package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TextView struct {
	TextAlign     int
	TextColor     tcell.Color
	Scrollable    bool
	Wrap          bool
	WordWrap      bool
	DynamicColors bool
	Regions       bool
	MaxLines      int
	Border        *Border
}

func NewTextView(config *TextView) *tview.TextView {
	view := tview.NewTextView().
		SetScrollable(config.Scrollable).
		SetWrap(config.Wrap).
		SetWordWrap(config.WordWrap).
		SetDynamicColors(config.DynamicColors).
		SetRegions(config.Regions).
		SetMaxLines(config.MaxLines).
		SetTextAlign(config.TextAlign).
		SetTextColor(config.TextColor)
	applyBorder(view, config.Border)
	return view
}
