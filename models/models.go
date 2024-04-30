package models

func NewGrid() [][]int {
	grid := make([][]int, 0)
	col := []int{0, 0, 0, 0, 0}

	rowColSize := 5
	for i := 0; i <= rowColSize; i++ {
		grid = append(grid, col)
	}
	return grid
}

type Player struct {
	IsReady    bool
	RemoteAddr string
	Grid       [][]int
}

type Game struct {
	Id   string
	Host *Player
	Join *Player
}

func NewGame(id string, host *Player) *Game {
	return &Game{
		Id:   id,
		Host: host,
	}
}
