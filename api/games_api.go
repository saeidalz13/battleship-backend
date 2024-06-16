package api

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

var GlobalGameManager = NewGameManager()

type GameManager struct {
	Games         map[string]*md.Game
	endGameSignal chan md.EndGameSignal
	mu            sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		Games:         make(map[string]*md.Game),
		endGameSignal: make(chan md.EndGameSignal),
	}
}

func (gm *GameManager) AddGame() *md.Game {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	newGame := md.NewGame()
	gm.Games[newGame.Uuid] = newGame
	return newGame
}

func (gm *GameManager) FindGame(gameUuid string) (*md.Game, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, prs := gm.Games[gameUuid]
	if !prs {
		return nil, cerr.ErrGameNotExists(gameUuid)
	}
	return game, nil
}

func (gm *GameManager) ManageGameTermination() {
mangeGamesLoop:
	for {
		// waiting for an end game signal
		egs := <-gm.endGameSignal

		switch egs.Code {
		case md.ManageGameCodePlayerDisconnect:
			game, err := gm.FindGame(egs.GameUuid)
			if err != nil {
				continue mangeGamesLoop
			}

			// egs RemoteAddr is the closed connection. We want to notify the
			// other player that the opponent connection is closed and game is done

			// Not checking the error since I have just pinged them and checked that
			// the other connection is open
			// In our case now, even if this write gives an error, it doesn't matter
			// since the game is ending anyways
			if egs.RemoteAddr == game.HostPlayer.WsConn.RemoteAddr().String() {
				_ = game.JoinPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
			} else if egs.RemoteAddr == game.JoinPlayer.WsConn.RemoteAddr().Network() {
				_ = game.HostPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
			}

		case md.ManageGameCodeSuccess, md.ManageGameCodeMaxTimeReached:
			// pass
		}
		gm.mu.Lock()
		delete(gm.Games, egs.GameUuid)
		gm.mu.Unlock()
		log.Printf("deleted game %s and its associated players\n", egs.GameUuid)
	}
}

// Checks the availability of both players throughout the game. This is
// done through pinging the connections every minute to see if they're open.
//
// This funcion also ensures the memory is cleaned up after
// when 30 mins passed each game creation.
func (gm *GameManager) CheckGameHealth(game *md.Game) {
	timer := time.NewTimer(maxTimeGame)
	ticker := time.NewTicker(connHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			gm.endGameSignal <- md.NewEndGameSignal(md.ManageGameCodeMaxTimeReached, game.Uuid, "")
			return

		case <-ticker.C:
			errJoinConn := game.JoinPlayer.WsConn.WriteMessage(websocket.PingMessage, nil)
			errHostConn := game.HostPlayer.WsConn.WriteMessage(websocket.PingMessage, nil)

			// This shows the game has ended. So this goroutine
			// needs to stop the check
			if errHostConn != nil && errJoinConn != nil {
				return
			}

			// This means that host is disconnected
			if errHostConn != nil {
				gm.endGameSignal <- md.NewEndGameSignal(md.ManageGameCodePlayerDisconnect, game.Uuid, game.HostPlayer.WsConn.RemoteAddr().String())
				return
			}

			// This means that join is disconnected
			if errJoinConn != nil {
				gm.endGameSignal <- md.NewEndGameSignal(md.ManageGameCodePlayerDisconnect, game.Uuid, game.JoinPlayer.WsConn.RemoteAddr().String())
				return
			}
		}
	}
}
