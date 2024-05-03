package models

import "github.com/gorilla/websocket"

const (
	// Create game
	CodeReqCreateGame = iota
	CodeRespCreateGame

	// Start game
	CodeRespSuccessStartGame
	CodeEndGame

	// Join game
	CodeReqJoinGame
	CodeRespSuccessJoinGame
	CodeRespFailJoinGame

	// Select grid
	CodeReqSelectGrid
	CodeRespSuccessSelectGrid
	CodeRespFailSelectGrid

	// Attack
	CodeReqAttack
	CodeRespSuccessAttack
	CodeRespFailAttack

	// Ready
	CodeReqReady
	CodeRespSuccessReady
	CodeRespFailReady

	// Misc
	CodeRespInvalidSignal
)

type Signal struct {
	Code int `json:"code"`
}

type GridInt [][]int

func NewGrid() GridInt {
	grid := make(GridInt, 0)
	col := []int{0, 0, 0, 0, 0}

	rowColSize := 5
	for i := 0; i <= rowColSize; i++ {
		grid = append(grid, col)
	}
	return grid
}

type Player struct {
	IsReady     bool
	IsTurn      bool
	IsHost      bool
	Uuid        string
	AttackGrid  GridInt
	DefenceGrid GridInt
	WsConn      *websocket.Conn
}

func NewPlayer(uuid string, ws *websocket.Conn, isHost, isTurn bool) *Player {
	return &Player{
		IsReady:     false,
		IsTurn:      isTurn,
		IsHost:      isHost,
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

func (g *Game) GetPlayers() []*Player {
	return []*Player{g.Host, g.Join}
}
