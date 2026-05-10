package main

import (
	"encoding/json"
	"os"
	"slices"
)

type Position struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}
type CellDisplay struct {
	Value     int  `json:"value"`
	Displayed bool `json:"displayed"`
}

type CellState struct {
	Solved bool `json:"solved"`
	Fogged bool `json:"fogged"`
	Defogs bool `json:"defogs"`
}

type Cell struct {
	Position Position    `json:"position"` // Global grid position
	Value    int         `json:"value"`    // True value (fixed solution)
	Function bool        `json:"function"` // True value is provided as a lambda function on runtime (Value ignored)
	Display  CellDisplay `json:"display"`  // Display value (either hardcoded, set dinamically, or player input)
	Editable bool        `json:"editable"` // Editable by player
	Foggers  []Position  `json:"foggers"`  // Positions of cells whose display value must be set for this cell to disipate fog. An empty array means no fog.
	State    CellState   `json:"state"`    // Cached cell state, updated on player input
	Graphics Graphics    `json:"graphics"`
}
type Thermo struct {
	Cells    []Position `json:"cells"` // Positions of cells that comprise the thermo (first cell is thermo bulb)
	Graphics Graphics   `json:"graphics"`
}
type Group struct {
	Cells    []Position `json:"cells"` // Positions of cells that comprise the group
	Graphics Graphics   `json:"graphics"`
}
type KillerZone struct {
	Cells    []Position `json:"cells"` // Positions of cells that comprise the killer zone
	Graphics Graphics   `json:"graphics"`
}

type BoardBounds struct {
	From Position `json:"from"` // Upperleftmost cell absolute position
	To   Position `json:"to"`   // Lowerrightmost cell absolute position
}
type Board struct {
	Bounds      *BoardBounds `json:"bounds"`
	Cells       [][]*Cell    `json:"cells"`       // All cells, in a grid
	Groups      []Group      `json:"groups"`      // All cell groups
	Thermos     []Thermo     `json:"thermos"`     // All thermos
	KillerZones []KillerZone `json:"killerzones"` // All killer zones
}

func NewCellDisplay() CellDisplay {
	return CellDisplay{
		Value:     0,
		Displayed: false,
	}
}

func NewCellState() CellState {
	return CellState{
		Solved: false,
		Fogged: false,
		Defogs: false,
	}
}

func NewCell(position Position, value int, editable bool, graphics *Graphics) *Cell {
	if graphics == nil {
		graphics = NewDefaultCellGraphics()
	}
	return &Cell{
		Position: position,
		Value:    value,
		Display:  NewCellDisplay(),
		Editable: editable,
		Foggers:  []Position{},
		State:    NewCellState(),
		Graphics: *graphics,
	}
}

func (c *Cell) UpdateSolved() {
	c.State.Solved = c.Value == c.Display.Value && c.Display.Displayed
}

func (c *Cell) UpdateFogged(foggers []*Cell) {
	c.State.Fogged = true
	for _, f := range foggers {
		c.State.Fogged = c.State.Fogged && f.Display.Displayed
	}
}

func (c *Cell) UpdateDefogs(foggers []*Cell) {
	c.State.Defogs = !c.State.Fogged
	for _, f := range foggers {
		c.State.Defogs = c.State.Defogs && f.State.Solved
	}
}

func (c *Cell) IsFogger(cellPosition Position) bool {
	return slices.Contains(c.Foggers, cellPosition)
}

func (c *Cell) AddFogger(cellPosition Position) {
	c.Foggers = append(c.Foggers, cellPosition)
}

func (c *Cell) DeleteFogger(cellPosition Position) {
	newFoggers := []Position{}
	for _, fp := range c.Foggers {
		if fp != cellPosition {
			newFoggers = append(newFoggers, fp)
		}
	}
	c.Foggers = newFoggers
}

func (c *Cell) ToggleFogger(cellPosition Position) {
	if !c.IsFogger(cellPosition) {
		c.AddFogger(cellPosition)
	} else {
		c.DeleteFogger(cellPosition)
	}
}

func NewGroup(cellPositions []Position, graphics *Graphics) *Group {
	if graphics == nil {
		graphics = NewDefaultGroupGraphics()
	}
	return &Group{
		Cells:    cellPositions,
		Graphics: *graphics,
	}
}

func NewThermo(cellPositions []Position, graphics *Graphics) *Thermo {
	if graphics == nil {
		graphics = NewDefaultThermoGraphics()
	}
	return &Thermo{
		Cells:    cellPositions,
		Graphics: *graphics,
	}
}

func (t *Thermo) IsThermoCell(row, column int) bool {
	return slices.Contains(t.Cells, Position{row, column})
}

func NewKillerZone(cellPositions []Position, graphics *Graphics) *KillerZone {
	if graphics == nil {
		graphics = NewDefaultKillerZoneGraphics()
	}
	return &KillerZone{
		Cells:    cellPositions,
		Graphics: *graphics,
	}
}

func NewBoard() *Board {
	return &Board{
		Bounds:      nil,
		Cells:       [][]*Cell{},
		Groups:      []Group{},
		Thermos:     []Thermo{},
		KillerZones: []KillerZone{},
	}
}

func (b *Board) IsValidIndex(row, column int) bool {
	if b.Bounds == nil {
		return false
	}
	rows, columns := b.Size()
	return row >= b.Bounds.From.Row && row < b.Bounds.From.Row+rows && column >= b.Bounds.From.Column && column < b.Bounds.From.Column+columns
}

func (b *Board) Index(row, column int) (int, int) {
	if b.Bounds == nil {
		return 0, 0
	}
	return row - b.Bounds.From.Row, column - b.Bounds.From.Column
}

func (b *Board) GetCell(row, column int) *Cell {
	if b.IsValidIndex(row, column) {
		rowIndex, columnIndex := b.Index(row, column)
		return b.Cells[rowIndex][columnIndex]
	}
	return nil
}

func (b *Board) GetCellFoggers(cell *Cell) []*Cell {
	var foggers []*Cell
	for _, fp := range cell.Foggers {
		rowIndex, columnIndex := b.Index(fp.Row, fp.Column)
		foggers = append(foggers, b.Cells[rowIndex][columnIndex])
	}
	return foggers
}

func (b *Board) Size() (int, int) {
	if b.Bounds == nil {
		return 0, 0
	}
	return len(b.Cells), len(b.Cells[0])
}

func (b *Board) UpdateSolved() {
	for _, row := range b.Cells {
		for _, cell := range row {
			cell.UpdateSolved()
		}
	}
}

func (b *Board) UpdateFogged() {
	for _, row := range b.Cells {
		for _, cell := range row {
			cell.UpdateFogged(b.GetCellFoggers(cell))
		}
	}
}

func (b *Board) UpdateDefogs() {
	for _, row := range b.Cells {
		for _, cell := range row {
			cell.UpdateDefogs(b.GetCellFoggers(cell))
		}
	}
}

func (b *Board) AddCell(value, row, column int, editable bool) *Cell {
	cell := NewCell(Position{Row: row, Column: column}, value, editable, nil)

	if b.Bounds == nil {
		b.Bounds = &BoardBounds{From: Position{Row: row, Column: column}, To: Position{Row: row, Column: column}}
		b.Cells = [][]*Cell{[]*Cell{cell}}
	} else {
		rows, columns := b.Size()
		rowIndex, columnIndex := b.Index(row, column)
		rowsToPrepend, columnsToPrepend := -rowIndex, -columnIndex
		if rowsToPrepend > 0 {
			for range rowsToPrepend {
				b.Cells = append([][]*Cell{make([]*Cell, columns)}, b.Cells...)
			}
		}
		if columnsToPrepend > 0 {
			for i, _ := range b.Cells {
				b.Cells[i] = append(make([]*Cell, columnsToPrepend), b.Cells[i]...)
			}
		}

		rows, columns = b.Size()
		rowIndex, columnIndex = b.Index(row, column)
		rowsToAppend, columnsToAppend := rowIndex-rows+1, columnIndex-columns+1
		if rowsToAppend > 0 {
			for range rowsToAppend {
				b.Cells = append(b.Cells, make([]*Cell, columns))
			}
		}
		if columnsToAppend > 0 {
			for i, _ := range b.Cells {
				b.Cells[i] = append(b.Cells[i], make([]*Cell, columnsToAppend)...)
			}
		}

		if row < b.Bounds.From.Row {
			b.Bounds.From.Row = row
		}
		if column < b.Bounds.From.Column {
			b.Bounds.From.Column = column
		}
		if row > b.Bounds.To.Row {
			b.Bounds.To.Row = row
		}
		if column > b.Bounds.To.Column {
			b.Bounds.To.Column = column
		}

		rowIndex, columnIndex = b.Index(row, column)
		b.Cells[rowIndex][columnIndex] = cell
	}
	return cell
}

func (b *Board) DeleteCell(row, column int) {
	if b.Bounds == nil {
		return
	}
	rows, columns := b.Size()
	rowIndex, columnIndex := b.Index(row, column)
	if rowIndex >= 0 && rowIndex < rows && columnIndex >= 0 && columnIndex < columns {
		b.Cells[rowIndex][columnIndex] = nil
	}
}

func (b *Board) AddThermo(cellPositions []Position) *Thermo {
	b.Thermos = append(b.Thermos, *NewThermo(cellPositions, nil))
	return &b.Thermos[len(b.Thermos)-1]
}

func (b *Board) DeleteThermo(thermo *Thermo) {
	thermo.Cells = []Position{}
	b.DeleteEmptyThermos()
}

func (b *Board) GetThermo(row, column int) (*Thermo, int) {
	var smallest *Thermo = nil
	index := -1
	for i, t := range b.Thermos {
		if t.IsThermoCell(row, column) {
			if smallest == nil || len(t.Cells) < len(smallest.Cells) {
				smallest = &t
				index = i
			}
		}
	}
	return smallest, index
}

func (b *Board) DeleteEmptyThermos() {
	newThermos := []Thermo{}
	for _, t := range b.Thermos {
		if len(t.Cells) > 0 {
			newThermos = append(newThermos, t)
		}
	}
	b.Thermos = newThermos
}

func (p Position) Neighbors() []Position {
	neighbors := []Position{}
	for _, d := range [][]int{
		[]int{-1, -1},
		[]int{-1, 0},
		[]int{-1, 1},
		[]int{0, 1},
		[]int{1, 1},
		[]int{1, 0},
		[]int{1, -1},
		[]int{0, -1},
	} {
		dy, dx := d[0], d[1]
		neighbors = append(neighbors, Position{p.Row + dy, p.Column + dx})
	}
	return neighbors
}

func (b *Board) ToJSON(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(b); err != nil {
		return err
	}

	return nil
}

func NewBoardFromJSON(path string) (*Board, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var b Board
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (b *Board) DeepCopy() (*Board, error) {
	bytes, err := json.Marshal(&b)
	if err != nil {
		return nil, err
	}
	var copy *Board
	err = json.Unmarshal(bytes, &copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}
