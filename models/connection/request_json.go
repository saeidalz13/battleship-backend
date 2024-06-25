package connection

import (
	b "github.com/saeidalz13/battleship-backend/models/battleship"
)

type ReqCreateGame struct {
	GameDifficulty int `json:"game_difficulty"`
}

type ReqReadyPlayer struct {
	GameUuid    string    `json:"game_uuid"`
	PlayerUuid  string    `json:"player_uuid"`
	DefenceGrid b.GridInt `json:"defence_grid"`
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
