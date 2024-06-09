package api

import (
	"log"

	md "github.com/saeidalz13/battleship-backend/models"
)

func SendMsgToBothPlayers(game *md.Game, hostMsg, joinMsg interface{}) error {
	playerOfGames := game.GetPlayers()
	for _, player := range playerOfGames {
		if player.IsHost {
			if err := player.WsConn.WriteJSON(hostMsg); err != nil {
				return err
			}
		} else {
			if err := player.WsConn.WriteJSON(joinMsg); err != nil {
				return err
			}
		}
		log.Printf("message sent to player: %+v\n", player.Uuid)
	}
	return nil
}

func FindGameAndPlayer(w *Request, gameUuid, playerUuid string) (*md.Game, *md.Player, error) {
	game, err := w.Server.FindGame(gameUuid)
	if err != nil {
		return nil, nil, err
	}
	player, err := w.Server.FindPlayer(playerUuid)
	if err != nil {
		return nil, nil, err
	}

	return game, player, nil
}
