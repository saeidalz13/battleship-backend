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
	Uuid       string
	RemoteAddr string
}

func NewPlayer(uuid, remoteAddr string) *Player {
	return &Player{
		IsReady:    false,
		Uuid:       uuid,
		RemoteAddr: remoteAddr,
	}
}

type Game struct {
	Uuid string
	Host *Player
	Join *Player
	Grid [][]int
}

func NewGame(uuid string, host *Player) *Game {
	return &Game{
		Uuid: uuid,
		Host: host,
		Grid: NewGrid(),
	}
}
