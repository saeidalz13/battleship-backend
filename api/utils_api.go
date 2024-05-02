package api

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/models"
)

func StartGame(ws *websocket.Conn) (string, *models.Game, string, *models.Player) {
	newGameUuid := uuid.NewString()[:6]
	newPlayerUuid := uuid.NewString()

	newPlayer := models.NewPlayer(newPlayerUuid, ws)
	newGame := models.NewGame(newGameUuid, newPlayer)

	return newGameUuid, newGame, newPlayerUuid, newPlayer
}
