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

// Convenient helper func to fetch both the game and player
func (gm *GameManager) FindGameAndPlayer(gameUuid, playerUuid string) (*md.Game, *md.Player, error) {
	game, err := gm.FindGame(gameUuid)
	if err != nil {
		return nil, nil, err
	}

	player, err := game.FindPlayer(playerUuid)
	if err != nil {
		return nil, nil, err
	}

	return game, player, nil
}

