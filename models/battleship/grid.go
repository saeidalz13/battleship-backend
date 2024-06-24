package battleship

type GridInt [][]int

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
