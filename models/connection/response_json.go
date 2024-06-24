package connection


import (
	b "github.com/saeidalz13/battleship-backend/models/battleship"
)

type RespJoinGame struct {
	GameUuid   string `json:"game_uuid"`
	PlayerUuid string `json:"player_uuid"`
}

type RespCreateGame struct {
	GameUuid string `json:"game_uuid"`
	HostUuid string `json:"host_uuid"`
}

type RespAttack struct {
	X               int  `json:"x"`
	Y               int  `json:"y"`
	PositionState   int  `json:"position_state"`
	IsTurn          bool `json:"is_turn"`
	SunkenShipsHost int  `json:"sunken_ships_host"`
	SunkenShipsJoin int  `json:"sunken_ships_join"`
}

type RespSessionId struct {
	SessionID string `json:"session_id"`
}

type RespEndGame struct {
	PlayerMatchStatus int `json:"player_match_status"`
}

type RespReconnect struct {
	IsTurn            bool    `json:"is_turn"`
	PlayerMatchStatus int     `json:"player_match_status"`
	SunkenShipsHost   int     `json:"sunken_ships_host"`
	SunkenShipsJoin   int     `json:"sunken_ships_join"`
	SessionID         string  `json:"session_id"`
	GameUuid          string  `json:"game_uuid"`
	PlayerUuid        string  `json:"player_uuid"`
	DefenceGrid       b.GridInt `json:"defence_grid"`
	AttackGrid        b.GridInt `json:"attack_grid"`
}

func NewRespReconnect(player *b.Player, game *b.Game) RespReconnect {
	return RespReconnect{
		IsTurn:            player.IsTurn,
		PlayerMatchStatus: player.MatchStatus,
		// SunkenShipsHost: ,
		SessionID:   player.SessionID,
		GameUuid:    player.CurrentGame.Uuid,
		PlayerUuid:  player.Uuid,
		DefenceGrid: player.DefenceGrid,
		AttackGrid:  player.AttackGrid,
	}
}

type RespErr struct {
	ErrorDetails string `json:"error_details"`
	Message      string `json:"message"`
}

func NewRespErr(errorDetails, message string) *RespErr {
	return &RespErr{
		ErrorDetails: errorDetails,
		Message:      message,
	}
}
