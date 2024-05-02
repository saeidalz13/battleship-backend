package models

const (
	CodeStartGame = iota
	CodeEndGame
	CodeAttack
	CodeReady
)

type SignalStruct struct {
	Code int `json:"code"`
}

type StartGameResp struct {
	GameUuid string  `json:"game_uuid"`
	HostUuid string  `json:"host_uuid"`
}
