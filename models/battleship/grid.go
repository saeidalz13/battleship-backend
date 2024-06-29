package battleship

type Grid [][]int

type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func NewCoordinates(x, y int) Coordinates {
	return Coordinates{X: x, Y: y}
}

const (
	GameGridSize = 5
)

// Creates a new default grid
// All indexes are zero/PositionStatusNeutral
func NewGrid(gridSize int) Grid {
	grid := make(Grid, gridSize)
	for i := 0; i < gridSize; i++ {
		grid[i] = make([]int, gridSize)
	}
	return grid
}
