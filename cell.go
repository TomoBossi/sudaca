package main

import (
	"encoding/json"
	"os"
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
	Value    int         `json:"value"`    // True value (solution)
	Display  CellDisplay `json:"display"`  // Display value (either hardcoded, set dinamically, or player input)
	Editable bool        `json:"editable"` // Editable by player
	Foggers  []*Cell     `json:"foggers"`  // Cells whose display value must be set for this cell to disipate fog. An empty array means no fog.
	State    CellState   `json:"state"`    // Cached cell state, updated on player input
	Graphics Graphics    `json:"graphics"`
}
type Group struct {
	Cells    []*Cell  `json:"cells"` // Cells that comprise the group
	Graphics Graphics `json:"graphics"`
}
type KillerZone struct {
	Cells    []*Cell  `json:"cells"` // Cells that comprise the killer zone
	Graphics Graphics `json:"graphics"`
}

type BoardBounds struct {
	From Position `json:"from"` // Upperleftmost cell absolute position
	To   Position `json:"to"`   // Lowerrightmost cell absolute position
}
type Board struct {
	Bounds      *BoardBounds  `json:"bounds"`
	Cells       [][]*Cell     `json:"cells"`       // All cells, in a grid
	Groups      []*Group      `json:"groups"`      // All cell groups
	KillerZones []*KillerZone `json:"killerzones"` // All killer zones
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
	c := Cell{
		Position: position,
		Value:    value,
		Display:  NewCellDisplay(),
		Editable: editable,
		Foggers:  []*Cell{},
		State:    NewCellState(),
		Graphics: *graphics,
	}
	c.UpdateFogged()
	c.UpdateDefogs()
	return &c
}

func (c Cell) UpdateSolved() {
	c.State.Solved = c.Value == c.Display.Value && c.Display.Displayed
}

func (c Cell) UpdateFogged() {
	c.State.Fogged = true
	for _, f := range c.Foggers {
		c.State.Fogged = c.State.Fogged && (*f).Display.Displayed
	}
}

func (c Cell) UpdateDefogs() {
	c.State.Defogs = !c.State.Fogged
	for _, f := range c.Foggers {
		c.State.Defogs = c.State.Defogs && (*f).State.Solved
	}
}

func NewGroup(cells []*Cell, graphics *Graphics) *Group {
	if graphics == nil {
		graphics = NewDefaultGroupGraphics()
	}
	return &Group{
		Cells:    cells,
		Graphics: *graphics,
	}
}

func NewKillerZone(cells []*Cell, graphics *Graphics) *KillerZone {
	if graphics == nil {
		graphics = NewDefaultKillerZoneGraphics()
	}
	return &KillerZone{
		Cells:    cells,
		Graphics: *graphics,
	}
}

func NewBoard() *Board {
	return &Board{
		Bounds:      nil,
		Cells:       [][]*Cell{},
		Groups:      []*Group{},
		KillerZones: []*KillerZone{},
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
	if b.Bounds == nil {
		return nil
	}
	rowIndex, columnIndex := b.Index(row, column)
	return b.Cells[rowIndex][columnIndex]
}

func (b *Board) Size() (int, int) {
	if b.Bounds == nil {
		return 0, 0
	}
	return len(b.Cells), len(b.Cells[0])
}

func (b *Board) AddCell(value, row, column int, editable bool) {
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

}

func (b *Board) toJSON(path string) error {
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
