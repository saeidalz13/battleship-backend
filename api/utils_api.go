package api

import (
	"log"
	"net"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	BreakWsLoop int = iota
	ContinueWsLoop
	RetryWriteConn
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

func IdentifyWsErrorAction(err error) int {
	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
		log.Println("close error:",err)
		return BreakWsLoop
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		log.Println("timeout error:", err)
		return RetryWriteConn
	}

	/*
		Cases to continue:

		CloseProtocolError           = 1002
		CloseUnsupportedData         = 1003
		CloseNoStatusReceived        = 1005
		CloseInvalidFramePayloadData = 1007
		ClosePolicyViolation         = 1008
		CloseMessageTooBig           = 1009
		CloseMandatoryExtension      = 1010
		CloseInternalServerErr       = 1011
		CloseServiceRestart          = 1012
		CloseTryAgainLater           = 1013
		CloseTLSHandshake            = 1015
	*/
	log.Println("continuing due to:", err)
	return ContinueWsLoop
}
