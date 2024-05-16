package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type WsRequestHandler interface {
	HandleCreateGame() (*md.Message, error)
	HandleReadyPlayer() (*md.Message, *md.Game, error)
	HandleJoinPlayer() (*md.Message, *md.Game, error)
	HandleAttack() (*md.Game, error)
}

// Every incoming valid request will have this structure
// The request then is handled in line with WsRequestHandler interface
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

func (w *WsRequest) HandleCreateGame() (*md.Message, error) {
	game := w.Server.AddGame()
	hostPlayer := w.Server.AddHostPlayer(game, w.Ws)

	resp := md.NewMessage(md.CodeSuccessCreateGame,
		md.WithPayload(
			md.RespCreateGame{
				GameUuid: game.Uuid,
				HostUuid: hostPlayer.Uuid,
			},
		))
	return &resp, nil
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (w *WsRequest) HandleReadyPlayer() (*md.Message, *md.Game, error) {
	var readyPlayerReq md.Message
	if err := json.Unmarshal(w.Payload, &readyPlayerReq); err != nil {
		return nil, nil, err
	}
	log.Printf("unmarshaled ready player payload: %+v\n", readyPlayerReq)

	initMap, err := TypeAssertPayloadToMap(readyPlayerReq.Payload)
	if err != nil {
		return nil, nil, err
	}
	defenceGrid, err := TypeAssertGridIntPayload(initMap, md.KeyDefenceGrid)
	if err != nil {
		return nil, nil, err
	}
	game, player, err := ExtractFindGamePlayer(w.Server, initMap)
	if err != nil {
		return nil, nil, err
	}
	player.SetReady(defenceGrid)

	resp := md.NewMessage(md.CodeRespSuccessReady)
	return &resp, game, nil
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (w *WsRequest) HandleJoinPlayer() (*md.Message, *md.Game, error) {
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

	game, err := w.Server.AddJoinPlayer(gameUuid, w.Ws)
	if err != nil {
		return nil, nil, err
	}

	resp := md.NewMessage(md.CodeRespSuccessJoinGame, md.WithPayload(md.RespJoinGame{PlayerUuid: game.JoinPlayer.Uuid}))
	return &resp, game, nil
}

func (w *WsRequest) HandleAttack() (*md.Game, error) {
	var reqAttack md.Message
	if err := json.Unmarshal(w.Payload, &reqAttack); err != nil {
		return nil, err
	}

	initMap, err := TypeAssertPayloadToMap(reqAttack.Payload)
	if err != nil {
		return nil, err
	}

	attackInfo, err := TypeAssertIntPayload(initMap, md.KeyX, md.KeyY, md.KeyPositionState)
	if err != nil {
		return nil, err
	}
	x, y, positionState := attackInfo[0], attackInfo[1], attackInfo[2]

	if x >= md.GameGridSize || y >= md.GameGridSize {
		return nil, cerr.ErrXorYOutOfGridBound(x, y)
	}

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