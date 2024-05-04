package models

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/saeidalz13/battleship-backend/internal/log"
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

func NewGame() *Game {
	return &Game{
		Uuid:       uuid.NewString()[:6],
	}
}

func (g *Game) GetPlayers() []*Player {
	return []*Player{g.HostPlayer, g.JoinPlayer}
}


func (g *Game) AddJoinPlayer(ws *websocket.Conn)  {
	joinPlayer := NewPlayer(ws, false, false)
	g.JoinPlayer = joinPlayer
	log.LogCustom("join player created and added to game", joinPlayer.Uuid)
}

func (g *Game) AddHostPlayer(ws *websocket.Conn) {
	hostPlayer := NewPlayer(ws, true, true)
	g.HostPlayer = hostPlayer
	log.LogCustom("host player created and added to game", hostPlayer.Uuid)
}