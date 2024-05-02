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

type ReadyPlayerReq struct {
	DefenceGrid [][]int `json:"defence_grid"`
	PlayerUuid  string  `json:"player_uuid"`
	GameUuid    string  `json:"game_uuid"`
}

type ReadyPlayerResp struct {
	Success bool `json:"success"`
}


type CreateGameResp struct {
	GameUuid string `json:"game_uuid"`
	HostUuid string `json:"host_uuid"`
}
