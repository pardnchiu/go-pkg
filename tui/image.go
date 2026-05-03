package tui

import (
	"image"

	"github.com/rivo/tview"
)

type Image struct {
	Image           image.Image
	Rows            int
	Columns         int
	Colors          int
	Dithering       int
	AspectRatio     float64
	AlignVertical   int
	AlignHorizontal int
}

func NewImage(config *Image, border *Border) *tview.Image {
	view := tview.NewImage().
		SetImage(config.Image).
		SetSize(config.Rows, config.Columns).
		SetColors(config.Colors).
		SetDithering(config.Dithering).
		SetAlign(config.AlignVertical, config.AlignHorizontal)
	if config.AspectRatio > 0 {
		view.SetAspectRatio(config.AspectRatio)
	}
	if border != nil {
		view.SetBorder(true).
			SetTitle(" " + border.Title + " ").
			SetTitleAlign(border.TitleAlign)
	}
	return view
}
