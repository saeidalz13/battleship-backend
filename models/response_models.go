package models

type RespGeneralMessage struct {
	Message string `json:"message"`
}

type RespReadyPlayer struct {
	Success bool `json:"success"`
}

type RespJoinGame struct {
	PlayerUuid string `json:"player_uuid"`
}

type RespCreateGame struct {
	GameUuid string `json:"game_uuid"`
	HostUuid string `json:"host_uuid"`
}

type RespAttack struct {
	IsTurn bool `json:"is_turn"`
	// Potentially other fields
}

type RespFail struct {
	ErrorDetails string `json:"error_details"`
	Message      string `json:"message"`
}

func NewRespFail(errorDetails, message string) *RespFail {
	return &RespFail{
		ErrorDetails: errorDetails,
		Message:      message,
	}
}
