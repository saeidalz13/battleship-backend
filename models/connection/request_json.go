package connection

import (
	b "github.com/saeidalz13/battleship-backend/models/battleship"
)

type ReqCreateGame struct {
	GameDifficulty uint8 `json:"game_difficulty"`
}

type ReqReadyPlayer struct {
	GameUuid    string `json:"game_uuid"`
	PlayerUuid  string `json:"player_uuid"`
	DefenceGrid b.Grid `json:"defence_grid"`
}

type ReqJoinGame struct {
	GameUuid string `json:"game_uuid"`
}

type ReqAttack struct {
	GameUuid   string `json:"game_uuid"`
	PlayerUuid string `json:"player_uuid"`
	X          uint8    `json:"x"`
	Y          uint8    `json:"y"`
}
