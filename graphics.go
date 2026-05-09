package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Stroke int

const (
	Solid Stroke = iota
	Dashed
	Dotted
)

type Graphics struct {
	ValueColor      color.RGBA           `json:"valuecolor"`
	BackgroundColor color.RGBA           `json:"backgroundcolor"`
	Stroke          Stroke               `json:"stroke"`
	StrokeOptions   vector.StrokeOptions `json:"strokeoptions"`
}

func NewDefaultCellGraphics() *Graphics {
	return &Graphics{
		ValueColor:      color.RGBA{0, 0, 0, 255},
		BackgroundColor: color.RGBA{255, 255, 255, 255},
		Stroke:          Solid,
		StrokeOptions: vector.StrokeOptions{
			Width: 1,
		},
	}
}
func NewDefaultGroupGraphics() *Graphics {
	return &Graphics{
		ValueColor:      color.RGBA{0, 0, 0, 255},
		BackgroundColor: color.RGBA{255, 255, 255, 0},
		Stroke:          Solid,
		StrokeOptions: vector.StrokeOptions{
			Width: 2,
		},
	}
}
func NewDefaultKillerZoneGraphics() *Graphics {
	return &Graphics{
		ValueColor:      color.RGBA{200, 100, 100, 255},
		BackgroundColor: color.RGBA{255, 255, 255, 0},
		Stroke:          Dashed,
		StrokeOptions: vector.StrokeOptions{
			Width: 2,
		},
	}
}
