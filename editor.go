package main

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var hexMap = map[int]string{
	10: "A",
	11: "B",
	12: "C",
	13: "D",
	14: "E",
	15: "F",
}

type pathOptions struct {
	fillColor   *color.RGBA
	strokeColor *color.RGBA
	strokeWidth float32
}

type pan struct {
	x float32
	y float32
}

type area struct {
	w float32
	h float32
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

type fonts struct {
	numbers *text.GoTextFaceSource
	text    *text.GoTextFaceSource
}

type boardState struct {
	boards []*Board
	index  int
}

type editState struct {
	fogging bool

	celling       bool
	cellSelection *Position

	thermoing       bool
	thermoSelection *Thermo
	thermoIndex     int

	grouping            bool
	groupSelection      *Group
	groupCellsSelection []Position
	groupIndex          int
}
type GameEditor struct {
	screenSize area
	savePath   string
	camera     camera
	cursor     cursor
	boardState boardState
	input      string
	fonts      fonts
	showInfo   bool
	editState  editState
}

func (g *GameEditor) resetEditState() {
	g.editState = editState{}
	g.resetInput()
}

func (g *GameEditor) board() *Board {
	return g.boardState.boards[g.boardState.index]
}

func (g *GameEditor) getFont(font string, fontSize float64) text.Face {
	fontSrc := g.fonts.numbers
	if font == "text" {
		fontSrc = g.fonts.text
	}
	return &text.GoTextFace{Source: fontSrc, Size: fontSize}
}

func NewGameEditor(readPath, savePath string) (*GameEditor, error) {
	textFontBytes, err := os.ReadFile("./fonts/Instruction.otf")
	if err != nil {
		return nil, err
	}
	textFontSrc, err := text.NewGoTextFaceSource(bytes.NewReader(textFontBytes))
	if err != nil {
		return nil, err
	}

	numberFontBytes, err := os.ReadFile("./fonts/SimplyMono-Bold.ttf")
	if err != nil {
		return nil, err
	}
	numberFontSrc, err := text.NewGoTextFaceSource(bytes.NewReader(numberFontBytes))
	if err != nil {
		return nil, err
	}

	var board *Board
	if readPath != "" {
		board, err = NewBoardFromJSON(readPath)
		if err != nil {
			return nil, err
		}
	} else {
		board = NewBoard()
	}

	return &GameEditor{
		savePath: savePath,
		camera: camera{
			pan:  pan{x: 0, y: 0},
			zoom: zoom{current: 1.0, min: minZoom, max: maxZoom, speed: 0.1},
		},
		boardState: boardState{boards: []*Board{board}, index: 0},
		editState:  editState{celling: true},
		fonts:      fonts{numbers: numberFontSrc, text: textFontSrc},
		showInfo:   true,
	}, nil
}

func ctrl() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControl)
}

func shift() bool {
	return ebiten.IsKeyPressed(ebiten.KeyShift)
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

func (c *camera) canZoomIn() bool {
	return c.zoom.current < c.zoom.max
}

func (c *camera) canZoomOut() bool {
	return c.zoom.current > c.zoom.min
}

func (c *camera) updateZoom(change float64, x, y float32) {
	if change > 0 && c.canZoomIn() || change < 0 && c.canZoomOut() {
		worldMx, worldMy := c.worldPosition(x, y)

		factor := float32(math.Pow(float64(1+c.zoom.speed), change))
		c.zoom.current *= factor
		if c.zoom.current < c.zoom.min {
			c.zoom.current = c.zoom.min
		}
		if c.zoom.current > c.zoom.max {
			c.zoom.current = c.zoom.max
		}

		c.pan.x = worldMx - x/c.zoom.current
		c.pan.y = worldMy - y/c.zoom.current
	}
}

func (g *GameEditor) updateMousePan() {
	mx, my := g.camera.worldCursorPosition()
	dx, dy := mx-g.cursor.position.x, my-g.cursor.position.y
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		ebiten.SetCursorShape(ebiten.CursorShapeMove)
		if dx != 0 || dy != 0 {
			g.camera.pan.x -= dx
			g.camera.pan.y -= dy
		}
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}
}

func (g *GameEditor) updateMouseZoom() {
	_, dy := ebiten.Wheel()
	x, y := screenCursorPosition()
	g.camera.updateZoom(dy, x, y)
}

func (g *GameEditor) updateWorldCursorPosition() {
	x, y := g.camera.worldCursorPosition()
	g.cursor.position.x = x
	g.cursor.position.y = y
}

func (g *GameEditor) updateCellSelection() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.editState.celling {
		p := g.hoveredCellPosition()
		if g.editState.cellSelection != nil && p.Row == g.editState.cellSelection.Row && p.Column == g.editState.cellSelection.Column {
			g.editState.cellSelection = nil
		} else {
			g.editState.cellSelection = &p
		}
		g.resetInput()
	}
}

func (g *GameEditor) updateState() error {
	if g.canRedo() {
		g.boardState.boards = g.boardState.boards[:g.boardState.index+1]
	}
	copy, err := g.boardState.boards[g.boardState.index].DeepCopy()
	if err != nil {
		return err
	}
	g.boardState.boards = append(g.boardState.boards, copy)
	g.boardState.index++
	g.resetInput()
	return nil
}

func (g *GameEditor) updateDeselection() {
	if !shift() && !ctrl() && g.editState.celling && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.editState.cellSelection = nil
		g.editState.groupSelection = nil
		g.resetInput()
	}
}

func (g *GameEditor) updateSelection() {
	g.updateCellSelection()
	g.updateDeselection()
}

func (g *GameEditor) updateCellValue() error {
	if g.editState.cellSelection != nil && g.input != "" && !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		value, err := strconv.Atoi(g.input)
		if err != nil {
			return err
		}
		err = g.updateState()
		if err != nil {
			return err
		}
		row, column := g.editState.cellSelection.Row, g.editState.cellSelection.Column
		if g.board().IsValidIndex(row, column) && g.board().GetCell(row, column) != nil {
			g.board().GetCell(row, column).Value = value
		} else {
			g.board().AddCell(value, row, column, false)
		}
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
		if !ctrl() && !shift() && g.editState.celling && inpututil.IsKeyJustPressed(key) {
			g.input += strconv.Itoa(i)
		}
	}
}

func (g *GameEditor) resetInput() {
	g.input = ""
}

func (g *GameEditor) canSave() bool {
	return g.savePath != ""
}

func (g *GameEditor) save() {
	if g.canSave() {
		g.board().ToJSON(g.savePath)
	}
}

func (g *GameEditor) updateSave() {
	if ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.save()
		g.resetInput()
	}
}

func (g *GameEditor) updateQuit() error {
	if !ctrl() && shift() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	return nil
}

func (g *GameEditor) canUndo() bool {
	return g.boardState.index > 0
}

func (g *GameEditor) canRedo() bool {
	return g.boardState.index < len(g.boardState.boards)-1
}

func (g *GameEditor) undo() {
	if g.canUndo() {
		g.boardState.index--
	}
}

func (g *GameEditor) redo() {
	if g.canRedo() {
		g.boardState.index++
	}
}

func (g *GameEditor) updateUndo() {
	if ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.undo()
		g.resetInput()
	}
}

func (g *GameEditor) updateRedo() {
	if ctrl() && shift() && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.redo()
		g.resetInput()
	}
}

func toggleFullscreen() {
	if ebiten.IsFullscreen() {
		ebiten.SetFullscreen(false)
	} else {
		ebiten.SetFullscreen(true)
	}
}

func (g *GameEditor) updateFullScreen() {
	if ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyF) {
		toggleFullscreen()
		g.resetInput()
	}
}

func (g *GameEditor) toggleShowInfo() {
	g.showInfo = !g.showInfo
}

func (g *GameEditor) updateShowInfo() {
	if !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyI) {
		g.toggleShowInfo()
		g.resetInput()
	}
}

func (c *camera) recenter() {
	c.zoom.current = 1
	c.pan.x = 0
	c.pan.y = 0
}

func (c *camera) canRecenter() bool {
	return c.zoom.current != 1 || c.pan.x != 0 || c.pan.y != 0
}

func (g *GameEditor) updateRecenter() {
	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.camera.recenter()
		g.resetInput()
	}
}

func (g *GameEditor) updateKeyboardZoom() {
	if ctrl() && !shift() {
		if inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
			g.camera.updateZoom(5, g.screenSize.w/2, g.screenSize.h/2)
			g.resetInput()
		} else if inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
			g.camera.updateZoom(-5, g.screenSize.w/2, g.screenSize.h/2)
			g.resetInput()
		}
	}
}

func (g *GameEditor) updateKeyboardPan() {
	if !ctrl() && !shift() {
		change := 10 / g.camera.zoom.current
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			g.camera.pan.x -= change
			g.resetInput()
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			g.camera.pan.x += change
			g.resetInput()
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			g.camera.pan.y += change
			g.resetInput()
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			g.camera.pan.y -= change
			g.resetInput()
		}
	}
}

func (g *GameEditor) updateCellEditable() error {
	if g.editState.cellSelection != nil && !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyE) {
		row, column := g.editState.cellSelection.Row, g.editState.cellSelection.Column
		if g.board().IsValidIndex(row, column) {
			cell := g.board().GetCell(row, column)
			if cell != nil {
				if err := g.updateState(); err != nil {
					return err
				}
				g.board().GetCell(row, column).Editable = !cell.Editable
			}
		}
	}
	return nil
}

func (g *GameEditor) updateCellFunction() error {
	if g.editState.cellSelection != nil && !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyX) {
		if err := g.updateState(); err != nil {
			return err
		}
		row, column := g.editState.cellSelection.Row, g.editState.cellSelection.Column
		cell := g.board().GetCell(row, column)
		if cell == nil {
			cell = g.board().AddCell(0, row, column, false)
		}
		cell.Function = !cell.Function
	}
	return nil
}

func (g *GameEditor) updateDeleteCell() error {
	if g.editState.cellSelection != nil && !ctrl() && !shift() && (inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyDelete)) {
		if err := g.updateState(); err != nil {
			return err
		}
		row, column := g.editState.cellSelection.Row, g.editState.cellSelection.Column
		g.board().DeleteCell(row, column)
	}
	return nil
}

func (g *GameEditor) updateFogging() {
	if g.editState.cellSelection != nil {
		if !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyF) {
			if g.editState.fogging {
				g.resetEditState()
				g.editState.celling = true
			} else {
				g.resetEditState()
				g.editState.fogging = true
			}
		}
	}
}

func (g *GameEditor) updateCellFoggers() error {
	selected := g.editState.cellSelection
	if selected != nil {
		hovered := g.hoveredCellPosition()
		if g.editState.fogging && *selected != hovered {
			selectedCell := g.board().GetCell(selected.Row, selected.Column)
			if selectedCell != nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				if err := g.updateState(); err != nil {
					return err
				}
				selectedCell = g.board().GetCell(selected.Row, selected.Column)
				selectedCell.ToggleFogger(hovered)
			}
		}
	}
	return nil
}

func (g *GameEditor) updateDeleteFoggers() error {
	selected := g.editState.cellSelection
	if selected != nil && g.editState.fogging && !ctrl() && !shift() && (inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyDelete)) {
		cell := g.board().GetCell(selected.Row, selected.Column)
		if len(cell.Foggers) > 0 {
			if err := g.updateState(); err != nil {
				return err
			}
			cell = g.board().GetCell(selected.Row, selected.Column)
			cell.Foggers = []Position{}
		}
		g.resetInput()
	}
	return nil
}

func (g *GameEditor) updateCell() error {
	if err := g.updateCellValue(); err != nil {
		return err
	}
	if err := g.updateCellEditable(); err != nil {
		return err
	}
	if err := g.updateCellFunction(); err != nil {
		return err
	}
	if err := g.updateDeleteCell(); err != nil {
		return err
	}
	g.updateFogging()
	if err := g.updateCellFoggers(); err != nil {
		return err
	}
	if err := g.updateDeleteFoggers(); err != nil {
		return err
	}
	return nil
}

func (g *GameEditor) updateThermoing() {
	if g.editState.cellSelection == nil {
		if !ctrl() && !shift() && inpututil.IsKeyJustPressed(ebiten.KeyT) {
			if g.editState.thermoing {
				g.resetEditState()
				g.editState.celling = true
			} else {
				g.resetEditState()
				g.editState.thermoing = true
			}
		}
	}
}

func (g *GameEditor) updatePlaceThermo() error {
	if !shift() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.editState.thermoing && g.editState.thermoSelection == nil {
		p := g.hoveredCellPosition()
		if err := g.updateState(); err != nil {
			return err
		}
		g.editState.thermoSelection = g.board().AddThermo([]Position{p})
		g.editState.thermoIndex = len(g.board().Thermos) - 1
	}
	return nil
}

func (g *GameEditor) updateDeleteThermo() error {
	if (inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyDelete)) && g.editState.thermoing && g.editState.thermoSelection != nil {
		if err := g.updateState(); err != nil {
			return err
		}
		g.board().Thermos[g.editState.thermoIndex].Cells = []Position{}
		g.board().DeleteEmptyThermos()
		g.editState.thermoSelection = nil
	}
	return nil
}

func (g *GameEditor) updateExtendThermo() (bool, error) {
	t := g.editState.thermoSelection
	if g.editState.thermoing && t != nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p := g.hoveredCellPosition()
		if len(t.Cells) > 0 {
			for _, neighbor := range t.Cells[len(t.Cells)-1].Neighbors() {
				if p == neighbor {
					if err := g.updateState(); err != nil {
						return false, err
					}
					g.editState.thermoSelection = &g.board().Thermos[g.editState.thermoIndex]
					g.editState.thermoSelection.Cells = append(g.editState.thermoSelection.Cells, p)
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (g *GameEditor) updateShortenThermo() (bool, error) {
	t := g.editState.thermoSelection
	if g.editState.thermoing && t != nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p := g.hoveredCellPosition()
		newCells := []Position{}
		for _, cp := range t.Cells {
			if p == cp {
				if err := g.updateState(); err != nil {
					return false, err
				}
				g.editState.thermoSelection = &g.board().Thermos[g.editState.thermoIndex]
				g.editState.thermoSelection.Cells = newCells
				if len(newCells) == 0 {
					g.board().DeleteEmptyThermos()
					g.editState.thermoSelection = nil
				}
				return true, nil
			}
			newCells = append(newCells, cp)
		}
	}
	return false, nil
}

func (g *GameEditor) updateSelectThermo() {
	if g.editState.thermoing && g.editState.cellSelection == nil {
		if !ctrl() && shift() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			p := g.hoveredCellPosition()
			g.editState.thermoSelection, g.editState.thermoIndex = g.board().GetThermo(p.Row, p.Column)
		}
	}
}

func (g *GameEditor) updateThermos() error {
	g.updateThermoing()
	if err := g.updateDeleteThermo(); err != nil {
		return err
	}
	shortened, err := g.updateShortenThermo()
	if err != nil {
		return err
	}
	if !shortened {
		extended, err := g.updateExtendThermo()
		if err != nil {
			return err
		}
		if !extended && !shortened {
			g.updateSelectThermo()
			if err := g.updatePlaceThermo(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *GameEditor) Update() error {
	if err := g.updateQuit(); err != nil {
		return err
	}
	g.updateFullScreen()
	g.updateDigitLogger()
	g.updateUndo()
	g.updateRedo()
	g.updateSave()
	g.updateRecenter()
	g.updateKeyboardZoom()
	g.updateKeyboardPan()
	g.updateShowInfo()
	if err := g.updateCell(); err != nil {
		return err
	}
	if err := g.updateThermos(); err != nil {
		return err
	}
	g.updateSelection()
	g.updateMouseZoom()
	g.updateMousePan()
	g.updateWorldCursorPosition()
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
	worldMinX := g.camera.pan.x
	worldMinY := g.camera.pan.y
	worldMaxX, worldMaxY := g.camera.worldPosition(g.screenSize.w, g.screenSize.h)

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

func (g *GameEditor) drawCellHighlightHover(dst *ebiten.Image) {
	p := g.hoveredCellPosition()
	if g.editState.fogging {
		if p != *g.editState.cellSelection {
			g.drawCellHighlight(dst, p, &color.RGBA{20, 50, 100, 100}, &color.RGBA{50, 50, 180, 255})
		}
	} else if g.editState.thermoing {
		t := g.editState.thermoSelection
		if t == nil {
			if shift() {
				ht, _ := g.board().GetThermo(p.Row, p.Column)
				if ht != nil {
					for _, cp := range ht.Cells {
						g.drawCellHighlight(dst, cp, &color.RGBA{}, &color.RGBA{80, 20, 20, 100})
					}
				}
			} else {
				g.drawCellHighlight(dst, p, &color.RGBA{100, 50, 20, 100}, &color.RGBA{180, 50, 50, 255})
			}
		} else {
			if t.IsThermoCell(p.Row, p.Column) {
				g.drawCellHighlight(dst, p, &color.RGBA{80, 20, 20, 100}, &color.RGBA{100, 30, 30, 255})
				if len(t.Cells) > 0 && p == t.Cells[len(t.Cells)-1] {
					for _, neighbor := range p.Neighbors() {
						if !t.IsThermoCell(neighbor.Row, neighbor.Column) {
							g.drawCellHighlight(dst, neighbor, &color.RGBA{100, 50, 20, 30}, &color.RGBA{180, 50, 50, 80})
						}
					}
				}
			} else {
				neighboring := false
				for _, neighbor := range p.Neighbors() {
					if t.IsThermoCell(neighbor.Row, neighbor.Column) && len(t.Cells) > 0 && neighbor == t.Cells[len(t.Cells)-1] {
						neighboring = true
					}
				}
				if neighboring {
					g.drawCellHighlight(dst, p, &color.RGBA{100, 50, 20, 100}, &color.RGBA{180, 50, 50, 255})
				}
			}
		}
	} else {
		g.drawCellHighlight(dst, p, &color.RGBA{20, 70, 50, 50}, &color.RGBA{50, 150, 100, 255})
		cell := g.board().GetCell(p.Row, p.Column)
		if cell != nil {
			for _, fp := range cell.Foggers {
				g.drawCellHighlight(dst, fp, &color.RGBA{20, 20, 70, 50}, &color.RGBA{50, 50, 180, 255})
			}
		}
	}
}

func (g *GameEditor) drawCellHighlight(dst *ebiten.Image, cellPosition Position, fillColor, strokeColor *color.RGBA) {
	x, y := float32(cellPosition.Column*cellSize), float32(cellPosition.Row*cellSize)
	var worldPath vector.Path
	worldPath.MoveTo(x, y)
	worldPath.LineTo(x, y+cellSize)
	worldPath.LineTo(x+cellSize, y+cellSize)
	worldPath.LineTo(x+cellSize, y)
	worldPath.LineTo(x, y)

	g.drawWorldPath(dst, worldPath, pathOptions{
		fillColor:   fillColor,
		strokeColor: strokeColor,
		strokeWidth: 2.0,
	})
}

func (g *GameEditor) drawHighlightSelection(dst *ebiten.Image) {
	p := g.editState.cellSelection
	if p != nil {
		g.drawCellHighlight(dst, *p, &color.RGBA{20, 20, 20, 50}, &color.RGBA{100, 100, 100, 255})
		if g.editState.fogging {
			cell := g.board().GetCell(p.Row, p.Column)
			if cell != nil {
				for _, fp := range cell.Foggers {
					g.drawCellHighlight(dst, fp, &color.RGBA{20, 20, 70, 50}, &color.RGBA{50, 50, 180, 255})
				}
			}
		}
	}
	t := g.editState.thermoSelection
	if t != nil {
		p := g.hoveredCellPosition()
		for _, cp := range t.Cells {
			if p == cp {
				break
			}
			g.drawCellHighlight(dst, cp, &color.RGBA{}, &color.RGBA{80, 20, 20, 100})
		}
	}
}

func dashedLineHorizontal(vp *vector.Path, xFrom, xTo, y float64) {
	signedLength := xTo - xFrom
	length := math.Abs(signedLength)
	sign := signedLength / length
	steps := math.Floor(length / dashedLineLength)
	padding := (length - dashedLineLength*(steps-1)) / 2
	x := float32(xFrom + sign*padding)
	vp.MoveTo(x, float32(y))
	for range int(steps/2 + 0.001) {
		x += float32(sign * dashedLineLength)
		vp.LineTo(x, float32(y))
		x += float32(sign * dashedLineLength)
		vp.MoveTo(x, float32(y))
	}
}

func dashedLineVertical(vp *vector.Path, yFrom, yTo, x float64) {
	signedLength := yTo - yFrom
	length := math.Abs(signedLength)
	sign := signedLength / length
	steps := math.Floor(length / dashedLineLength)
	padding := (length - dashedLineLength*(steps-1)) / 2
	y := float32(yFrom + sign*padding)
	vp.MoveTo(float32(x), y)
	for range int(steps/2 + 0.001) {
		y += float32(sign * dashedLineLength)
		vp.LineTo(float32(x), y)
		y += float32(sign * dashedLineLength)
		vp.MoveTo(float32(x), y)
	}
}

func (g *GameEditor) drawBoardBounds(dst *ebiten.Image) {
	if g.board().Bounds != nil {
		from := g.board().Bounds.From
		to := g.board().Bounds.To
		xFrom, yFrom := float32(from.Column*cellSize), float32(from.Row*cellSize)
		xTo, yTo := float32(to.Column*cellSize+cellSize), float32(to.Row*cellSize+cellSize)
		var worldPath vector.Path
		dashedLineHorizontal(&worldPath, float64(xFrom), float64(xTo), float64(yFrom))
		dashedLineVertical(&worldPath, float64(yFrom), float64(yTo), float64(xTo))
		dashedLineHorizontal(&worldPath, float64(xTo), float64(xFrom), float64(yTo))
		dashedLineVertical(&worldPath, float64(yTo), float64(yFrom), float64(xFrom))

		g.drawWorldPath(dst, worldPath, pathOptions{
			strokeColor: &color.RGBA{0, 0, 0, 50},
			strokeWidth: 2.0,
		})
	}
}

func (g *GameEditor) drawCellValue(dst *ebiten.Image, cell *Cell) {
	if cell != nil {
		x, y := (float32(cell.Position.Column)+0.5)*cellSize, (float32(cell.Position.Row)+0.5)*cellSize
		str := strconv.Itoa(cell.Value)
		hex, ok := hexMap[cell.Value]
		if ok && g.hoveredCellPosition() != cell.Position {
			str = hex
		}
		if cell.Function {
			str = "X"
		}
		var alpha uint8 = 255
		if cell.Editable {
			alpha = 100
		}
		var blue uint8 = 0
		if len(cell.Foggers) > 0 {
			blue = 100
		}
		g.drawCenteredText(dst, str, float64(x), float64(y), g.getFont("numbers", float64(numberFontSize*g.camera.zoom.current)), color.RGBA{0, 0, blue, alpha})

	}
}

func (g *GameEditor) drawCellsValues(dst *ebiten.Image) {
	for _, row := range g.board().Cells {
		for _, cell := range row {
			g.drawCellValue(dst, cell)
		}
	}
}

func (g *GameEditor) drawInfoText(dst *ebiten.Image, str string, x, y, fontSize float64) {
	op := &text.DrawOptions{}
	op.LineSpacing = fontSize * 1.5
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255})
	text.Draw(dst, str, g.getFont("text", fontSize), op)
}

func (g *GameEditor) drawInfo(dst *ebiten.Image) {
	fontSize := textFontSize * 0.8
	var str string

	p := g.hoveredCellPosition()
	if g.showInfo {
		if g.canSave() {
			str += fmt.Sprintf("Ctrl + S - Save Board (%s)\n", g.savePath)
		} else {
			str += fmt.Sprintf("Saving disabled, run again with --read-path or --save-path flags\n")
		}
		str += "Shift + ESC - Quit\n"
		str += "Ctrl + F - Toggle Fullscreen\n"
		if g.canUndo() {
			str += "Ctrl + Z - Undo\n"
		}
		if g.canRedo() {
			str += "Ctrl + Shift + Z - Redo\n"
		}
		if g.camera.canRecenter() {
			str += "Ctrl + R - Recenter\n"
		}
		if g.camera.canZoomIn() {
			str += "Ctrl + (+) / MMB Up - Zoom In\n"
		}
		if g.camera.canZoomOut() {
			str += "Ctrl + (-) / MMB Down - Zoom Out\n"
		}
		str += "Arrow Keys / RMB Drag - Pan\n"
		str += "I - Toggle Info Display\n"
		str += "\n"

		if g.editState.celling {
			if g.editState.cellSelection != nil {
				if *g.editState.cellSelection == p {
					str += "ESC/LMB Click - Deselect cell\n"
				} else {
					str += "LMB Click - Select cell (Deselect Current)\n"
					str += "ESC - Deselect cell\n"
				}
				if g.input != "" {
					str += fmt.Sprintf("[0-9] - Update number input (%s)\n", g.input)
					str += "Enter - Set cell value\n"
				} else {
					str += "[0-9] - Update number input\n"
				}
				row, column := g.editState.cellSelection.Row, g.editState.cellSelection.Column
				cell := g.board().GetCell(row, column)
				if cell != nil {
					str += fmt.Sprintf("E - Toggle User Editable (%t)\n", cell.Editable)
					str += fmt.Sprintf("X - Toggle Function (%t)\n", cell.Function)
					str += fmt.Sprintf("F - Update Foggers (%d)\n", len(cell.Foggers))
					str += "Backspace/Del - Delete Cell\n"
				} else {
					str += "X - Set as function cell\n"
				}
			} else {
				str += "LMB Click - Select cell\n"
			}
		}

		if g.editState.fogging {
			str += "LMB Click - Toggle Fogger\n"
			str += "F - Finish Fogger Update\n"
			str += "Backspace/Del - Delete all Foggers\n"
		}

		t := g.editState.thermoSelection
		if g.editState.celling && g.editState.cellSelection == nil {
			if t == nil {
				str += "T - Edit Thermos\n"
			}
		}
		if g.editState.thermoing {
			if t == nil {
				str += "LMB Click - Place New Thermo\n"
				if len(g.board().Thermos) > 0 {
					str += "Shift + LMB Click - Select Thermo\n"
				}
			} else {
				for _, cp := range t.Cells {
					if cp == p {
						str += "LMB Click - Shorten Thermo\n"
						break
					}
				}
				if len(t.Cells) > 0 {
					for _, neighbor := range t.Cells[len(t.Cells)-1].Neighbors() {
						if neighbor == p {
							str += "LMB Click - Extend Thermo\n"
							break
						}
					}
				}
			}
			str += "T - Finish Editing Thermos\n"
			if t != nil {
				str += "Backspace/Del - Delete Thermo\n"
			}
		}

		g.drawInfoText(dst, str, 10, 10, fontSize)

		str = fmt.Sprintf("TPS: %0.2f\n", ebiten.ActualTPS())
		str += fmt.Sprintf("FPS: %0.2f\n", ebiten.ActualFPS())
		g.drawInfoText(dst, str, 10, float64(g.screenSize.h-40), fontSize)
	}
}

func (g *GameEditor) updateScreenSize(dst *ebiten.Image) {
	w, h := screenSize(dst)
	g.screenSize = area{w: w, h: h}
}

func (g *GameEditor) drawThermos(dst *ebiten.Image) {
	numThermos := len(g.board().Thermos)
	for i, t := range g.board().Thermos {
		if len(t.Cells) > 0 {
			color := &t.Graphics.ValueColor
			color.R = uint8(float64(color.R) * (1 - float64(i)/float64(numThermos)/3))
			color.B = uint8(float64(color.B) * (1 + float64(i)/float64(numThermos)/5))
			bulbPosition := t.Cells[0]
			x, y := float32(bulbPosition.Column*cellSize)+cellSize/2, float32(bulbPosition.Row*cellSize)+cellSize/2
			var bulbWorldPath vector.Path
			bulbWorldPath.MoveTo(x, y)
			bulbWorldPath.Arc(x, y, thermoBulbRadius, 0, 2*math.Pi, vector.Clockwise)
			g.drawWorldPath(dst, bulbWorldPath, pathOptions{
				fillColor: color,
			})
			var stemWorldPath vector.Path
			stemWorldPath.MoveTo(x, y)
			for _, tp := range t.Cells[1:] {
				x, y = float32(tp.Column*cellSize)+cellSize/2, float32(tp.Row*cellSize)+cellSize/2
				stemWorldPath.LineTo(x, y)
				bulbWorldPath.Arc(x, y, thermoStemWidth*g.camera.zoom.current/2, 0, 2*math.Pi, vector.Clockwise)
				stemWorldPath.MoveTo(x, y)
			}
			g.drawWorldPath(dst, stemWorldPath, pathOptions{
				strokeColor: color,
				strokeWidth: thermoStemWidth * g.camera.zoom.current,
			})
		}
	}
}

// func (g *GameEditor) drawGroups(dst *ebiten.Image) {
// 	directions := map[]

// 	for _, gr := range g.board().Groups {
// 		if len(gr.Cells) > 0 {
// 			var path vector.Path
// 			leftmost := gr.LeftMostCellPosition()
// 			start := Position{leftmost.Row + 1, leftmost.Column}
// 			xFrom, yFrom := float32(start.Column*cellSize), float32(start.Row*cellSize)
// 			path.MoveTo(xFrom, yFrom)
// 			xTo, yTo := xFrom - cellSize, yTo
// 			g.drawWorldPath(dst, path, pathOptions{
// 				fillColor: &gr.Graphics.ValueColor,
// 			})
// 		}
// 	}
// }

func (g *GameEditor) Draw(screen *ebiten.Image) {
	dst := screen
	g.updateScreenSize(dst)
	dst.Fill(color.RGBA{224, 224, 224, 255})
	g.drawThermos(dst)
	g.drawGrid(dst)
	g.drawBoardBounds(dst)
	g.drawHighlightSelection(dst)
	g.drawCellsValues(dst)
	// g.drawGroups(dst)
	g.drawCellHighlightHover(dst)
	g.drawInfo(dst)
}

func (g *GameEditor) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
