package models

type ReqReadyPlayer struct {
	Code        int     `json:"code"`
	DefenceGrid GridInt `json:"defence_grid"`
	GameUuid    string  `json:"game_uuid"`
	PlayerUuid  string  `json:"player_uuid"`
}

type ReqJoinGame struct {
	Code     int    `json:"code"`
	GameUuid string `json:"gamae_uuid"`
}
