package models

type ReqReadyPlayer struct {
	DefenceGrid GridInt `json:"defence_grid"`
	GameUuid    string  `json:"game_uuid"`
	PlayerUuid  string  `json:"player_uuid"`
}

type ReqJoinGame struct {
	GameUuid string `json:"game_uuid"`
}

type ReqAttack struct {
	GameUuid      string `json:"game_uuid"`
	PlayerUuid    string `json:"player_uuid"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	PositionState int    `json:"position_state"`
}
