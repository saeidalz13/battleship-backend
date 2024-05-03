package models

type RespGeneralMessage struct {
	Message string `json:"message"`
}

type RespReadyPlayer struct {
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
