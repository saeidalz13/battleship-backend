package api

import (
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	BreakLoop int = iota
	ContinueLoop
	PassThrough
	RetryWriteConn
)

func SendMsgToBothPlayers(game *md.Game, hostMsg, joinMsg interface{}) int {
	playerOfGames := game.GetPlayers()

	for _, player := range playerOfGames {
		if player.IsHost {
			switch WriteJsonWithRetry(player.WsConn, hostMsg) {
			case BreakLoop:
				return BreakLoop
			default:
				continue
			}

		} else {
			switch WriteJsonWithRetry(player.WsConn, joinMsg) {
			case BreakLoop:
				return BreakLoop
			default:
				continue
			}
		}
	}

	return PassThrough
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
		log.Println("close error:", err)
		return BreakLoop
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
	return ContinueLoop
}

func WriteJsonWithRetry(ws *websocket.Conn, resp interface{}) int {
	var retries int

writeJsonLoop:
	for {
		if err := ws.WriteJSON(resp); err != nil {
			switch IdentifyWsErrorAction(err) {
			case RetryWriteConn:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("writing json failed to ws [%s]; retrying... (retry no. %d)\n", ws.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue writeJsonLoop

				} else {
					log.Printf("max retries reached for writing to ws [%s]:%s", ws.RemoteAddr().String(), err)
					return BreakLoop
				}

			case BreakLoop:
				log.Println("breaking writeJsonLoop due to:", err)
				return BreakLoop

			case ContinueLoop:
				log.Println("continue but this error happened:", err)
				return ContinueLoop
			}

			// Successful write and continue the main ws loop
		} else {
			return PassThrough
		}
	}
}
