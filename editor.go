package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"

	"github.com/ebitengine/debugui"
	"golang.org/x/image/font/gofont/gomono"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	minZoom        = 0.1
	maxZoom        = 5.0
	cellSize       = 50
	numberFontSize = 35
	textFontSize   = 15
)

type pathOptions struct {
	fillColor   *color.RGBA
	strokeColor *color.RGBA
	strokeWidth float32
}

type pan struct {
	x float32
	y float32
}

type zoom struct {
	current float32
	min     float32
	max     float32
	speed   float32
}

type camera struct {
	pan  pan
	zoom zoom
}

type cursorPosition struct {
	x, y float32
}
type cursor struct {
	position cursorPosition
}

type boardSelection struct {
	cell  *Position
	group []*Position
}

type fonts struct {
	numbers *text.GoTextFaceSource
	text    *text.GoTextFaceSource
}

type GameEditor struct {
	debugui   debugui.DebugUI
	savePath  string
	camera    camera
	cursor    cursor
	board     *Board
	selection boardSelection
	input     string
	fonts     fonts
}

func (g *GameEditor) getFont(font string, fontSize float32, scale float32) text.Face {
	fontSrc := g.fonts.numbers
	if font == "text" {
		fontSrc = g.fonts.text
	}
	size := float64(scale * fontSize * g.camera.zoom.current)
	return &text.GoTextFace{Source: fontSrc, Size: size}
}

func NewGameEditor(readPath, savePath string) (*GameEditor, error) {
	textFontSrc, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return nil, err
	}

	numberFontBytes, err := os.ReadFile("SimplyMono-Bold.ttf")
	if err != nil {
		return nil, err
	}
	numberFontSrc, err := text.NewGoTextFaceSource(bytes.NewReader(numberFontBytes))
	if err != nil {
		return nil, err
	}

	board := NewBoard()
	if readPath != "" {
		board, err = NewBoardFromJSON(readPath)
		if err != nil {
			return nil, err
		}
		if savePath == "" {
			savePath = readPath
		}
	}

	return &GameEditor{
		savePath: savePath,
		camera: camera{
			pan:  pan{x: 0, y: 0},
			zoom: zoom{current: 1.0, min: minZoom, max: maxZoom, speed: 0.1},
		},
		board: board,
		fonts: fonts{numbers: numberFontSrc, text: textFontSrc},
	}, nil
}

func screenSize(screen *ebiten.Image) (float32, float32) {
	s := screen.Bounds().Size()
	return float32(s.X), float32(s.Y)
}

func screenCursorPosition() (float32, float32) {
	mx, my := ebiten.CursorPosition()
	return float32(mx), float32(my)
}

func (c *camera) apply(geoM *ebiten.GeoM) {
	geoM.Translate(float64(-c.pan.x), float64(-c.pan.y))
	geoM.Scale(float64(c.zoom.current), float64(c.zoom.current))
}

func (c *camera) worldPosition(screenX, screenY float32) (float32, float32) {
	// screen = (world - pan)*zoom
	// world  = screen/zoom + pan
	worldX := screenX/c.zoom.current + c.pan.x
	worldY := screenY/c.zoom.current + c.pan.y
	return worldX, worldY
}

func (c *camera) screenPosition(worldX, worldY float32) (float32, float32) {
	screenX := (worldX - c.pan.x) * c.zoom.current
	screenY := (worldY - c.pan.y) * c.zoom.current
	return screenX, screenY
}

func (c *camera) worldCursorPosition() (float32, float32) {
	mx, my := screenCursorPosition()
	return c.worldPosition(mx, my)
}

func (c *camera) updateZoom() {
	_, dy := ebiten.Wheel()
	if dy != 0 {
		mx, my := screenCursorPosition()
		worldMx, worldMy := c.worldCursorPosition()

		factor := float32(math.Pow(float64(1+c.zoom.speed), dy))
		c.zoom.current *= factor
		if c.zoom.current < c.zoom.min {
			c.zoom.current = c.zoom.min
		}
		if c.zoom.current > c.zoom.max {
			c.zoom.current = c.zoom.max
		}

		c.pan.x = worldMx - mx/c.zoom.current
		c.pan.y = worldMy - my/c.zoom.current
	}
}

func (g *GameEditor) updatePan() {
	mx, my := g.camera.worldCursorPosition()
	dx, dy := mx-g.cursor.position.x, my-g.cursor.position.y
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) && (dx != 0 || dy != 0) {
		g.camera.pan.x = g.camera.pan.x - dx
		g.camera.pan.y = g.camera.pan.y - dy
	}
}

func (g *GameEditor) updateWorldCursorPosition() {
	x, y := g.camera.worldCursorPosition()
	g.cursor.position.x = x
	g.cursor.position.y = y
}

func (g *GameEditor) updateDebugui() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Debug", image.Rect(10, 10, 170, 320), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
			ctx.Text(fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
			ctx.Text(fmt.Sprintf("Zoom: %0.2f", g.camera.zoom.current))
			ctx.Text(fmt.Sprintf("Pan X: %0.2f", g.camera.pan.x))
			ctx.Text(fmt.Sprintf("Pan Y: %0.2f", g.camera.pan.y))
			ctx.Text(fmt.Sprintf("Hover Row: %d", g.hoveredCellPosition().Row))
			ctx.Text(fmt.Sprintf("Hover Column: %d", g.hoveredCellPosition().Column))
			if g.selection.cell != nil {
				ctx.Text(fmt.Sprintf("Highlight Row: %d", g.selection.cell.Row))
				ctx.Text(fmt.Sprintf("Highlight Column: %d", g.selection.cell.Column))
			}
			if g.board.Bounds != nil {
				ctx.Text(fmt.Sprintf("Grid start Row: %d", g.board.Bounds.From.Row))
				ctx.Text(fmt.Sprintf("Grid start Column: %d", g.board.Bounds.From.Column))
			}
			ctx.Text(fmt.Sprintf("Input: %s", g.input))
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (g *GameEditor) updateSelection() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p := g.hoveredCellPosition()
		if g.selection.cell != nil && p.Row == g.selection.cell.Row && p.Column == g.selection.cell.Column {
			g.selection.cell = nil
		} else {
			g.selection.cell = &p
		}
		g.input = ""
	}
}

func (g *GameEditor) updateCellValue() error {
	if g.selection.cell != nil && g.input != "" && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		value, err := strconv.Atoi(g.input)
		if err != nil {
			return err
		}
		row, column := g.selection.cell.Row, g.selection.cell.Column
		if g.board.IsValidIndex(row, column) && g.board.GetCell(row, column) != nil {
			g.board.GetCell(row, column).Value = value
		} else {
			g.board.AddCell(value, row, column, false)
		}
		g.input = ""
	}
	return nil
}

func (g *GameEditor) updateDigitLogger() {
	for i, key := range []ebiten.Key{
		ebiten.Key0,
		ebiten.Key1,
		ebiten.Key2,
		ebiten.Key3,
		ebiten.Key4,
		ebiten.Key5,
		ebiten.Key6,
		ebiten.Key7,
		ebiten.Key8,
		ebiten.Key9,
	} {
		if inpututil.IsKeyJustPressed(key) {
			g.input += strconv.Itoa(i)
		}
	}
}

func (g *GameEditor) updateKeyboardInput() {
	g.updateDigitLogger()
	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.board.toJSON(g.savePath)
	}
}

func (g *GameEditor) Update() error {
	err := g.updateDebugui()
	if err != nil {
		return err
	}

	g.updateKeyboardInput()
	g.camera.updateZoom()
	g.updatePan()
	g.updateWorldCursorPosition()
	g.updateSelection()
	err = g.updateCellValue()
	if err != nil {
		return err
	}
	return nil
}

func (g *GameEditor) drawWorldPath(dst *ebiten.Image, path vector.Path, op pathOptions) {
	var screenPath vector.Path
	addOp := &vector.AddPathOptions{}
	g.camera.apply(&addOp.GeoM)
	screenPath.AddPath(&path, addOp)

	if op.fillColor != nil {
		drawOp := &vector.DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(*op.fillColor)
		vector.FillPath(dst, &screenPath, &vector.FillOptions{}, drawOp)
	}

	if op.strokeColor != nil {
		drawOp := &vector.DrawPathOptions{}
		drawOp.AntiAlias = true
		drawOp.ColorScale.ScaleWithColor(*op.strokeColor)
		strokeOp := &vector.StrokeOptions{
			Width:    op.strokeWidth,
			LineCap:  vector.LineCapRound,
			LineJoin: vector.LineJoinRound,
		}
		vector.StrokePath(dst, &screenPath, strokeOp, drawOp)
	}
}

func (g *GameEditor) drawCenteredText(dst *ebiten.Image, str string, x, y float64, face text.Face, color color.RGBA) {
	sx, sy := g.camera.screenPosition(float32(x), float32(y))
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(sx), float64(sy))
	op.ColorScale.ScaleWithColor(color)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(dst, str, face, op)
}

func (g *GameEditor) drawGrid(dst *ebiten.Image) {
	sw, sh := screenSize(dst)

	worldMinX := g.camera.pan.x
	worldMinY := g.camera.pan.y
	worldMaxX, worldMaxY := g.camera.worldPosition(sw, sh)

	startX := float32(math.Floor(float64(worldMinX)/cellSize) * cellSize)
	startY := float32(math.Floor(float64(worldMinY)/cellSize) * cellSize)

	var worldPath vector.Path
	for x := startX; x <= worldMaxX; x += cellSize {
		worldPath.MoveTo(x, worldMinY)
		worldPath.LineTo(x, worldMaxY)
	}
	for y := startY; y <= worldMaxY; y += cellSize {
		worldPath.MoveTo(worldMinX, y)
		worldPath.LineTo(worldMaxX, y)
	}

	g.drawWorldPath(dst, worldPath, pathOptions{strokeColor: &color.RGBA{0, 0, 0, 30}, strokeWidth: 1.0})
}

func (g *GameEditor) hoveredCellPosition() Position {
	x, y := g.cursor.position.x, g.cursor.position.y
	return Position{
		Row:    int(math.Floor(float64(y) / cellSize)),
		Column: int(math.Floor(float64(x) / cellSize)),
	}
}

func (g *GameEditor) drawHighlightHover(dst *ebiten.Image) {
	p := g.hoveredCellPosition()
	x, y := float32(p.Column*cellSize), float32(p.Row*cellSize)
	var worldPath vector.Path
	worldPath.MoveTo(x, y)
	worldPath.LineTo(x, y+cellSize)
	worldPath.LineTo(x+cellSize, y+cellSize)
	worldPath.LineTo(x+cellSize, y)
	worldPath.LineTo(x, y)

	g.drawWorldPath(dst, worldPath, pathOptions{
		fillColor:   &color.RGBA{20, 20, 100, 50},
		strokeColor: &color.RGBA{50, 50, 255, 255},
		strokeWidth: 2.0,
	})
}

func (g *GameEditor) drawHighlightSelection(dst *ebiten.Image) {
	p := g.selection.cell
	if p != nil {
		x, y := float32(p.Column*cellSize), float32(p.Row*cellSize)
		var worldPath vector.Path
		worldPath.MoveTo(x, y)
		worldPath.LineTo(x, y+cellSize)
		worldPath.LineTo(x+cellSize, y+cellSize)
		worldPath.LineTo(x+cellSize, y)
		worldPath.LineTo(x, y)

		g.drawWorldPath(dst, worldPath, pathOptions{
			fillColor:   &color.RGBA{20, 20, 20, 50},
			strokeColor: &color.RGBA{100, 100, 100, 255},
			strokeWidth: 2.0,
		})
	}
}

func (g *GameEditor) drawBoardBounds(dst *ebiten.Image) {
	if g.board.Bounds != nil {
		from := g.board.Bounds.From
		to := g.board.Bounds.To
		xFrom, yFrom := float32(from.Column*cellSize), float32(from.Row*cellSize)
		xTo, yTo := float32(to.Column*cellSize+cellSize), float32(to.Row*cellSize+cellSize)
		var worldPath vector.Path
		worldPath.MoveTo(xFrom, yFrom)
		worldPath.LineTo(xFrom, yTo)
		worldPath.LineTo(xTo, yTo)
		worldPath.LineTo(xTo, yFrom)
		worldPath.LineTo(xFrom, yFrom)

		g.drawWorldPath(dst, worldPath, pathOptions{
			strokeColor: &color.RGBA{150, 150, 150, 255},
			strokeWidth: 2.0,
		})
	}
}

func (g *GameEditor) drawCellValues(dst *ebiten.Image) {
	for _, row := range g.board.Cells {
		for _, cell := range row {
			if cell != nil {
				str := strconv.Itoa(cell.Value)
				x, y := (float32(cell.Position.Column)+0.5)*cellSize, (float32(cell.Position.Row)+0.5)*cellSize
				g.drawCenteredText(dst, str, float64(x), float64(y), g.getFont("numbers", numberFontSize, 1), color.RGBA{0, 0, 0, 255})
			}
		}
	}
}
func (g *GameEditor) Draw(screen *ebiten.Image) {
	dst := screen
	dst.Fill(color.RGBA{224, 224, 224, 255})
	g.drawGrid(dst)
	g.drawBoardBounds(dst)
	g.drawHighlightSelection(dst)
	g.drawHighlightHover(dst)
	g.drawCellValues(dst)
	g.debugui.Draw(screen)
}

func (g *GameEditor) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
