package error

import "fmt"

func ErrGameNotExists(gameUuid string) error {
	return fmt.Errorf("game with this uuid does not exist, uuid: %s", gameUuid)
}

func ErrPlayerNotExist(playerUuid string) error {
	return fmt.Errorf("player with this uuid does not exist, uuid: %s", playerUuid)
}

func ErrNilPayload() error {
	return fmt.Errorf("the payload is nil and is not of type map")
}

func ErrKeyNotExists(key string) error {
	return fmt.Errorf("the key does not exist:\t%s", key)
}

func ErrValueNotString(value interface{}) error {
	return fmt.Errorf("the value is not of type string:\t%t", value)
}

func ErrValueNotInt(value interface{}) error {
	return fmt.Errorf("the value is not of type int:\t%t", value)
}

func ErrValueNotGridInt() error {
	return fmt.Errorf("the value is not of type GridInt")
}

func ErrXorYOutOfGridBound(x, y int) error {
	return fmt.Errorf("incoming x or y is out of game grid bound\tx: %d\ty: %d", x, y)
}

func ErrAttackPositionAlreadyFilled(x, y int) error {
	return fmt.Errorf("current position in grid already taken\tx: %d\ty: %d", x, y)
}
