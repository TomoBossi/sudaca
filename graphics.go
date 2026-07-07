package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	minZoom          = 0.3
	maxZoom          = 5.0
	cellSize         = 50
	numberFontSize   = 35
	textFontSize     = 15
	dashedLineLength = 5
	thermoBulbRadius = 20
	thermoStemWidth  = 15
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
			Width: 3,
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

func NewDefaultThermoGraphics() *Graphics {
	return &Graphics{
		ValueColor: color.RGBA{250, 200, 150, 255},
	}
}
