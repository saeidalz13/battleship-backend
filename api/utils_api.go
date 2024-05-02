package api

import (
	"github.com/google/uuid"
	"github.com/saeidalz13/battleship-backend/models"
)

func StartGame(remoteAddr string) (string, *models.Game, *models.Player) {
	// TODO: change this to a 6-char string
	newGameUuid := uuid.NewString()
	newPlayerUuid := uuid.NewString()

	newPlayer := models.NewPlayer(newPlayerUuid, remoteAddr)
	newGame := models.NewGame(newGameUuid, newPlayer)

	return newGameUuid, newGame, newPlayer
}
