package models

const (
	CodeCreateGame = iota
	CodeStartGame
	CodeEndGame
	CodeJoinGame
	CodeSelectGrid
	CodeAttack
	CodeAttackResult
	CodeReady
)

type SignalStruct struct {
	Code int `json:"code"`
}

type CreateGameResp struct {
	GameUuid string `json:"game_uuid"`
	HostUuid string `json:"host_uuid"`
}
