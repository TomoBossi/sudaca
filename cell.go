package main

type Position struct {
	Row    int
	Column int
}
type CellDisplay struct {
	Value     int
	Displayed bool
}

func NewEmptyCellDisplay() CellDisplay {
	return CellDisplay{
		Value:     0,
		Displayed: false,
	}
}

type CellState struct {
	Solved bool
	Fogged bool
	Defogs bool
}

func NewCellState() CellState {
	return CellState{
		Solved: false,
		Fogged: false,
		Defogs: false,
	}
}

type Cell struct {
	Position Position    // Global grid position
	Value    int         // True value (solution)
	Display  CellDisplay // Display value (either hardcoded, set dinamically, or player input)
	Editable bool        // Editable by player
	Foggers  []*Cell     // Cells whose display value must be set for this cell to disipate fog. An empty array means no fog.
	State    CellState   // Cached cell state, updated on player input
	Graphics Graphics
}

func NewCell(position Position, value int, editable bool, graphics *Graphics) *Cell {
	if graphics == nil {
		graphics = NewDefaultCellGraphics()
	}
	c := Cell{
		Position: position,
		Value:    value,
		Display:  NewEmptyCellDisplay(),
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

type Group struct {
	Cells    []*Cell // Cells that comprise the group
	Graphics Graphics
}

func NewGroup(cells []*Cell, graphics *Graphics) Group {
	if graphics == nil {
		graphics = NewDefaultGroupGraphics()
	}
	return Group{
		Cells:    cells,
		Graphics: *graphics,
	}
}

type KillerZone struct {
	Cells    []*Cell // Cells that comprise the killer zone
	Graphics Graphics
}

func NewKillerZone(cells []*Cell, graphics *Graphics) Group {
	if graphics == nil {
		graphics = NewDefaultKillerZoneGraphics()
	}
	return Group{
		Cells:    cells,
		Graphics: *graphics,
	}
}

type Board struct {
	Cells       [][]*Cell     // All cells, in a grid
	Groups      []*Group      // All cell groups
	KillerZones []*KillerZone // All killer zones
}
