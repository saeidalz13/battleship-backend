package models

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

type RespEndGame struct {
	PlayerMatchStatus int `json:"player_match_status"`
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
