package models

type RespGeneralMessage struct {
	Message string `json:"message"`
}

type RespReadyPlayer struct {
	Code    int  `json:"code"`
	Success bool `json:"success"`
}

type RespJoinGame struct {
	Code       int    `json:"code"`
	PlayerUuid string `json:"player_uuid"`
}

type RespCreateGame struct {
	Code     int    `json:"code"`
	GameUuid string `json:"game_uuid"`
	HostUuid string `json:"host_uuid"`
}

type RespSuccessAttack struct {
	Code   int  `json:"code"`
	IsTurn bool `json:"is_turn"`
	// Potentially other fields
}

type RespFail struct {
	Code         int    `json:"code"`
	ErrorDetails string `json:"error_details"`
	Message      string `json:"message"`
}

func NewRespFail(code int, errorDetails, message string) *RespFail {
	return &RespFail{
		Code:         code,
		ErrorDetails: errorDetails,
		Message:      message,
	}
}
