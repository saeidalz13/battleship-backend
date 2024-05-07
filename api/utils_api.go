package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type WsRequestHandler interface {
	CreateGame() (*md.Message, error)
	ManageReadyPlayer() (*md.Message, *md.Game, error)
	JoinPlayerToGame() (*md.Message, *md.Game, error)
	Attack() (*md.Game, error)
}

type WsRequest struct {
	Server  *Server
	Ws      *websocket.Conn
	Payload []byte
}

// This tells the compiler that WsRequest struct must be of type of WsRequestHandler
var _ WsRequestHandler = (*WsRequest)(nil)

func NewWsRequest(server *Server, ws *websocket.Conn, payload ...[]byte) *WsRequest {
	if len(payload) > 1 {
		log.Println("cannot accept more than one payload")
		return nil
	}

	wsReq := WsRequest{
		Server: server,
		Ws:     ws,
	}
	if len(payload) != 0 {
		wsReq.Payload = payload[0]
	}
	return &wsReq
}

func (w *WsRequest) CreateGame() (*md.Message, error) {
	newGame := md.NewGame()
	w.Server.Games[newGame.Uuid] = newGame

	newGame.AddHostPlayer(w.Ws)
	w.Server.Players[newGame.HostPlayer.Uuid] = newGame.HostPlayer

	resp := md.NewMessage(md.CodeSuccessCreateGame,
		md.WithPayload(
			md.RespCreateGame{
				GameUuid: newGame.Uuid,
				HostUuid: newGame.HostPlayer.Uuid,
			},
		))
	return &resp, nil
}

func (w *WsRequest) ManageReadyPlayer() (*md.Message, *md.Game, error) {
	var readyPlayerReq md.Message
	if err := json.Unmarshal(w.Payload, &readyPlayerReq); err != nil {
		return nil, nil, err
	}
	log.Printf("unmarshaled ready player payload: %+v\n", readyPlayerReq)

	// Check if payload empty
	initMap, err := TypeAssertPayloadToMap(readyPlayerReq.Payload)
	if err != nil {
		return nil, nil, err
	}

	game, player, err := ExtractFindGamePlayer(w.Server, initMap)
	if err != nil {
		return nil, nil, err
	}
	defenceGrid, err := TypeAssertGridIntPayload(initMap, md.KeyDefenceGrid)
	if err != nil {
		return nil, nil, err
	}

	player.SetReady(defenceGrid)

	// prepare the response
	resp := md.NewMessage(md.CodeRespSuccessReady)
	return &resp, game, nil
}

func (w *WsRequest) JoinPlayerToGame() (*md.Message, *md.Game, error) {
	var joinGameReq md.Message
	if err := json.Unmarshal(w.Payload, &joinGameReq); err != nil {
		return &md.Message{}, nil, err
	}
	log.Printf("unmarshaled join game payload: %+v\n", joinGameReq)

	initMap, err := TypeAssertPayloadToMap(joinGameReq.Payload)
	if err != nil {
		return nil, nil, err
	}
	desiredStrings, err := TypeAssertStringPayload(initMap, md.KeyGameUuid)
	if err != nil {
		return nil, nil, err
	}
	gameUuid := desiredStrings[0]

	game := w.Server.FindGame(gameUuid)
	if game == nil {
		return &md.Message{}, nil, cerr.ErrGameNotExists(gameUuid)
	}
	log.Printf("found game in database: %+v\n", game)

	// Join the player to game and prepare the ws response
	game.AddJoinPlayer(w.Ws)
	resp := md.NewMessage(md.CodeRespSuccessJoinGame, md.WithPayload(md.RespJoinGame{PlayerUuid: game.JoinPlayer.Uuid}))

	return &resp, game, nil
}

func (w *WsRequest) Attack() (*md.Game, error) {
	var reqAttack md.Message
	if err := json.Unmarshal(w.Payload, &reqAttack); err != nil {
		return nil, err
	}

	initMap, err := TypeAssertPayloadToMap(reqAttack.Payload)
	if err != nil {
		return nil, err
	}

	// extract attack info
	attackInfo, err := TypeAssertIntPayload(initMap, md.KeyX, md.KeyY, md.KeyPositionState)
	if err != nil {
		return nil, err
	}
	x, y, positionState := attackInfo[0], attackInfo[1], attackInfo[2]

	game, player, err := ExtractFindGamePlayer(w.Server, initMap)
	if err != nil {
		return nil, err
	}

	// TODO: Assumption is that the frontend decides if the attack is hit or miss
	if player.IsHost {
		game.HostPlayer.AttackGrid[x][y] = positionState
		game.HostPlayer.IsTurn = false
		game.JoinPlayer.IsTurn = true
	} else {
		game.JoinPlayer.AttackGrid[x][y] = positionState
		game.JoinPlayer.IsTurn = false
		game.HostPlayer.IsTurn = true
	}

	return game, nil
}

func EndGame(s *Server, ws *websocket.Conn, payload []byte) error {
	return nil
}

func SendMsgToBothPlayers(game *md.Game, hostMsg, joinMsg *md.Message) error {
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

func ExtractFindGamePlayer(server *Server, initMap map[string]interface{}) (*md.Game, *md.Player, error) {
	// Extract info from payload map
	desiredStrings, err := TypeAssertStringPayload(initMap, md.KeyGameUuid, md.KeyPlayerUuid)
	if err != nil {
		return nil, nil, err
	}
	gameUuid, playerUuid := desiredStrings[0], desiredStrings[1]

	game := server.FindGame(gameUuid)
	if game == nil {
		return nil, nil, cerr.ErrGameNotExists(gameUuid)
	}
	player := server.FindPlayer(playerUuid)
	if player == nil {
		return nil, nil, cerr.ErrPlayerNotExist(playerUuid)
	}
	return game, player, nil
}
