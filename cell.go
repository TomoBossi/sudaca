package main

type Position struct {
	Row    int
	Column int
}

type Cell struct {
	Position Position // Global grid position
	Value    int      // True value (solution)
	Display  int      // Display value (either hardcoded, set dinamically, or player input)
	Editable bool     // Editable by player
	Defog    bool     // Keeps state about whether this cell counts for the foggers array of other cells, so it doesn't have to be recomputed on every input. It is true only if all foggers have defog set to true.
	Foggers  []*Cell  // Cells whose display value must be set for this cell to disipate fog. An empty array means no fog.
	Graphics Graphics
}

func newCell(position Position, value, display int, editable bool, graphics *Graphics) *Cell {
	if graphics == nil {
		graphics = &DefaultCellGraphics
	}
	return &Cell{
		Position: position,
		Value:    value,
		Display:  display,
		Editable: editable,
		Defog:    false,
		Foggers:  []*Cell{},
		Graphics: *graphics,
	}
}

type Group struct {
	Cells    []*Cell // Cells that comprise the group
	Killer   bool    // Killer zone
	Graphics Graphics
}

type Board struct {
	Cells  [][]*Cell // All cells
	Groups []*Group  // All cell groups
}
