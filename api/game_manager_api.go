package api

import (
	"log"
	"sync"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	b "github.com/saeidalz13/battleship-backend/models/battleship"
)

type GameManager struct {
	Games       map[string]*b.Game
	EndGameChan chan string
	mu          sync.RWMutex
}

func NewGameManager() *GameManager {
	return &GameManager{
		Games:       make(map[string]*b.Game),
		EndGameChan: make(chan string),
	}
}

func (gm *GameManager) AddGame() *b.Game {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	newGame := b.NewGame()
	gm.Games[newGame.Uuid] = newGame
	return newGame
}

func (gm *GameManager) FindGame(gameUuid string) (*b.Game, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, prs := gm.Games[gameUuid]
	if !prs {
		return nil, cerr.ErrGameNotExists(gameUuid)
	}

	if game == nil {
		return nil, cerr.ErrGameIsNil(gameUuid)
	}

	return game, nil
}

func (gm *GameManager) DeletePlayerFromGame(gameUuid, playerUuid string) {
	game, err := gm.FindGame(gameUuid)
	if err != nil {
		return
	}

	gm.mu.Lock()
	delete(game.Players, playerUuid)
	log.Printf("player deleted: %s", playerUuid)

	// Check if that was the last player
	// If yes, remove the game
	if len(game.Players) == 0 {
		delete(gm.Games, game.Uuid)
		log.Printf("game deleted: %s", game.Uuid)
	}
	gm.mu.Unlock()
}

// Convenient helper func to fetch both the game and player
func (gm *GameManager) FindGameAndPlayer(gameUuid, playerUuid string) (*b.Game, *b.Player, error) {
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
