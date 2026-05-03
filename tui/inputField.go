package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InputField struct {
	Label                string
	LabelWidth           int
	LabelColor           tcell.Color
	FieldWidth           int
	FieldBgColor         tcell.Color
	FieldTextColor       tcell.Color
	PlaceholderText      string
	PlaceholderTextColor tcell.Color
	MaskCharacter        rune
	DoneFunc             func(key tcell.Key)
	InputCapture         func(event *tcell.EventKey) *tcell.EventKey
	Border               *Border
}

func NewInputField(config *InputField) *tview.InputField {
	view := tview.NewInputField().
		SetLabel(config.Label).
		SetLabelWidth(config.LabelWidth).
		SetLabelColor(config.LabelColor).
		SetFieldWidth(config.FieldWidth).
		SetFieldBackgroundColor(config.FieldBgColor).
		SetFieldTextColor(config.FieldTextColor).
		SetPlaceholder(config.PlaceholderText).
		SetPlaceholderTextColor(config.PlaceholderTextColor).
		SetMaskCharacter(config.MaskCharacter)
	if config.DoneFunc != nil {
		view.SetDoneFunc(config.DoneFunc)
	}
	if config.InputCapture != nil {
		view.SetInputCapture(config.InputCapture)
	}
	applyBorder(view, config.Border)
	return view
}
