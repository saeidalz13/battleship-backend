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

// func ExtractFindGamePlayer(server *Server, initMap map[string]interface{}) (*md.Game, *md.Player, error) {
// 	desiredStrings, err := TypeAssertStringPayload(initMap, md.KeyGameUuid, md.KeyPlayerUuid)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	gameUuid, playerUuid := desiredStrings[0], desiredStrings[1]

// 	game := server.FindGame(gameUuid)
// 	if game == nil {
// 		return nil, nil, cerr.ErrGameNotExists(gameUuid)
// 	}
// 	player := server.FindPlayer(playerUuid)
// 	if player == nil {
// 		return nil, nil, cerr.ErrPlayerNotExist(playerUuid)
// 	}
// 	return game, player, nil
// }
