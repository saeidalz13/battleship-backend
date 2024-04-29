package models

func NewGrid() [][]int {
	grid := make([][]int, 0)
	col := []int{0, 1, 2, 3, 4}

	rowColSize := 5
	for i := 0; i <= rowColSize; i++ {
		grid = append(grid, col)
	}
	return grid
}

type Player struct {
	RemoteAddr string
	IsReady    bool
	Grid       [][]int
}

type Game struct {
	Id   string
	Host Player
	Join Player
}
