package internal

import (
	"github.com/google/uuid"
	"github.com/saeidalz13/battleship-backend/models"
)

func StartGame(remoteAddr string) (string, *models.Game, *models.Player) {
	newId := uuid.New()
	newPlayer := &models.Player{
		IsReady:    false,
		RemoteAddr: remoteAddr,
		Grid:       models.NewGrid(),
	}
	newGame := models.NewGame(newId.String(), newPlayer)

	return newId.String(), newGame, newPlayer
}
