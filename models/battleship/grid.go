package battleship

type Coordinates struct {
	X uint8 `json:"x"`
	Y uint8 `json:"y"`
}

func NewCoordinates(x, y uint8) Coordinates {
	return Coordinates{X: x, Y: y}
}

type Grid [][]uint8

// Creates a new default grid
// All indexes are zero/PositionStatusNeutral
func NewGrid(gridSize uint8) Grid {
	grid := make(Grid, gridSize)

	for i := uint8(0); i < gridSize; i++ {
		grid[i] = make([]uint8, gridSize)
	}
	return grid
}
