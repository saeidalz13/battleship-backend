package models

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Create game
	CodeReqCreateGame = iota
	CodeSuccessCreateGame
	CodeFailCreateGame

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

func NewPlayer(ws *websocket.Conn, isHost, isTurn bool) *Player {
	return &Player{
		IsReady:     false,
		IsTurn:      isTurn,
		IsHost:      isHost,
		Uuid:        uuid.NewString()[:10],
		AttackGrid:  NewGrid(),
		DefenceGrid: NewGrid(),
		WsConn:      ws,
	}
}

type Game struct {
	Uuid       string
	HostPlayer *Player
	JoinPlayer *Player
}

func NewGame(host *Player) *Game {
	return &Game{
		Uuid:       uuid.NewString()[:6],
		HostPlayer: host,
	}
}

func (g *Game) GetPlayers() []*Player {
	return []*Player{g.HostPlayer, g.JoinPlayer}
}


func (g *Game) AddJoinPlayer(ws *websocket.Conn) {
	NewPlayer(ws, false, false)
}

func (g *Game) AddHostPlayer(ws *websocket.Conn) {
	NewPlayer(ws, true, true)
}