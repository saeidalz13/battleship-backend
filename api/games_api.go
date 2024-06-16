package api

import (
	"sync"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

var GlobalGameManager = NewGameManager()

type GameManager struct {
	Games         map[string]*md.Game
	EndGameSignal chan string
	mu            sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		Games:         make(map[string]*md.Game),
		EndGameSignal: make(chan string),
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
	for {
		gameUuid := <-gm.EndGameSignal

		gm.mu.Lock()
		delete(gm.Games, gameUuid)
		gm.mu.Unlock()
	}
}

// 	for {
// 		// waiting for an end game signal
// 		egs := <-gm.EndGameSignal

// 		switch egs.Code {
// 		case md.ManageGameCodePlayerDisconnect:
// 			// Not checking the error since I have just pinged them and checked that
// 			// the other connection is open
// 			// In our case now, even if this write gives an error, it doesn't matter
// 			// since the game is ending anyways
// 			if egs.DisconnectedRemoteAddr == egs.Game.HostPlayer.WsConn.RemoteAddr().String(){
// 				_ = egs.Game.JoinPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
// 			} else if egs.DisconnectedRemoteAddr == egs.Game.JoinPlayer.WsConn.RemoteAddr().String() {
// 				_ = egs.Game.HostPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
// 			}

// 		case md.ManageGameCodeSuccess, md.ManageGameCodeMaxTimeReached:
// 			// pass
// 		}
// 		gm.mu.Lock()
// 		delete(gm.Games, egs.Game.Uuid)
// 		gm.mu.Unlock()
// 		log.Printf("deleted game %s and its associated players\n", egs.Game.Uuid)
// 	}
// }

// // Checks the availability of both players throughout the game. This is
// // done through pinging the connections every minute to see if they're open.
// //
// // This funcion also ensures the memory is cleaned up after
// // when 30 mins passed each game creation.
// func (gm *GameManager) CheckGameHealth(game *md.Game) {
// 	timer := time.NewTimer(maxTimeGame)
// 	ticker := time.NewTicker(connHealthCheckInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-timer.C:
// 			gm.EndGameSignal <- md.NewEndGameSignal(md.ManageGameCodeMaxTimeReached, game, "")
// 			return

// 		case <-ticker.C:
// 			errJoinConn := game.JoinPlayer.WsConn.WriteMessage(websocket.PingMessage, nil)
// 			errHostConn := game.HostPlayer.WsConn.WriteMessage(websocket.PingMessage, nil)

// 			// This shows the game has ended. So this goroutine
// 			// needs to stop the check
// 			if errHostConn != nil && errJoinConn != nil {
// 				return
// 			}

// 			// This means that host is disconnected
// 			if errHostConn != nil {
// 				gm.EndGameSignal <- md.NewEndGameSignal(md.ManageGameCodePlayerDisconnect, game, game.HostPlayer.WsConn.RemoteAddr().String())
// 				return
// 			}

// 			// This means that join is disconnected
// 			if errJoinConn != nil {
// 				gm.EndGameSignal <- md.NewEndGameSignal(md.ManageGameCodePlayerDisconnect, game, game.JoinPlayer.WsConn.RemoteAddr().String())
// 				return
// 			}
// 		}
// 	}
// }
