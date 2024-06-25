package battleship

type GridInt [][]int

type Coords struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func NewCoords(x, y int) Coords {
	return Coords{X: x, Y: y}
}

const (
	GameGridSize = 5
)

// Creates a new default grid
// All indexes are zero/PositionStatusNeutral
func NewGrid() GridInt {
	grid := make(GridInt, GameGridSize)
	for i := 0; i < GameGridSize; i++ {
		grid[i] = make([]int, GameGridSize)
	}
	return grid
}
