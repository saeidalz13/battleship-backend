package models

type ReqReadyPlayer struct {
	GameUuid    string  `json:"game_uuid"`
	PlayerUuid  string  `json:"player_uuid"`
	DefenceGrid GridInt `json:"defence_grid"`
}

type ReqJoinGame struct {
	GameUuid string `json:"game_uuid"`
}

type ReqAttack struct {
	GameUuid   string `json:"game_uuid"`
	PlayerUuid string `json:"player_uuid"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
}
