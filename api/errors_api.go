package api

import "fmt"

func ErrorGameNotExist(gameUuid string) error {
	return fmt.Errorf("game with this uuid does not exist, uuid: %s", gameUuid)
}

func ErrorPlayerNotExist(playerUuid string) error {
	return fmt.Errorf("player with this uuid does not exist, uuid: %s", playerUuid)
}
