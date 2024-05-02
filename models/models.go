package models

import "github.com/gorilla/websocket"

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
	IsReady     bool
	Uuid        string
	AttackGrid  [][]int
	DefenceGrid [][]int
	WsConn      *websocket.Conn
}

func NewPlayer(uuid string, ws *websocket.Conn) *Player {
	return &Player{
		IsReady:     false,
		Uuid:        uuid,
		AttackGrid:  NewGrid(),
		DefenceGrid: NewGrid(),
		WsConn:      ws,
	}
}

type Game struct {
	Uuid string
	Host *Player
	Join *Player
}

func NewGame(uuid string, host *Player) *Game {
	return &Game{
		Uuid: uuid,
		Host: host,
	}
}
