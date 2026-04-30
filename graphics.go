package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	CellSize int     = 20
	FontSize int     = 20
	ZoomMin  float64 = 0.25
	ZoomMax  float64 = 3.0
	ZoomStep float64 = 0.05
)

type Stroke int

const (
	Solid Stroke = iota
	Dashed
	Dotted
)

type Graphics struct {
	ValueColor      color.RGBA
	BackgroundColor color.RGBA
	Stroke          Stroke
	StrokeOptions   vector.StrokeOptions
}

var DefaultCellGraphics = Graphics{
	ValueColor:      color.RGBA{0, 0, 0, 255},
	BackgroundColor: color.RGBA{255, 255, 255, 255},
	Stroke:          Solid,
	StrokeOptions: vector.StrokeOptions{
		Width: 1,
	},
}

var DefaultGroupGraphics = Graphics{
	ValueColor:      color.RGBA{0, 0, 0, 255},
	BackgroundColor: color.RGBA{255, 255, 255, 0},
	Stroke:          Solid,
	StrokeOptions: vector.StrokeOptions{
		Width: 2,
	},
}

var DefaultKillerZoneGraphics = Graphics{
	ValueColor:      color.RGBA{200, 100, 100, 255},
	BackgroundColor: color.RGBA{255, 255, 255, 0},
	Stroke:          Dashed,
	StrokeOptions: vector.StrokeOptions{
		Width: 2,
	},
}
