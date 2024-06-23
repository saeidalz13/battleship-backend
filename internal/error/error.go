package error

import "fmt"

const (
	ConstErrAttack         = "attack operation failed"
	ConstErrReady          = "ready player operation failed"
	ConstErrJoin           = "join player operation failed"
	ConstErrInvalidPayload = "invalid request payload"
)

func ErrGameNotExists(gameUuid string) error {
	return fmt.Errorf("game with this uuid does not exist, uuid: %s", gameUuid)
}

func ErrGameIsNil(gameUuid string) error {
	return fmt.Errorf("game with this uuid is nil\t uuid: %s", gameUuid)
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

// Attack Errors

func ErrXorYOutOfGridBound(x, y int) error {
	return fmt.Errorf("incoming x or y is out of game grid bound\tx: %d\ty: %d", x, y)
}

func ErrAttackPositionAlreadyFilled(x, y int) error {
	return fmt.Errorf("current position in grid already taken\tx: %d\ty: %d", x, y)
}

func ErrNotTurnForAttacker(attackerId string) error {
	return fmt.Errorf("this is not the turn to attack for player %s", attackerId)
}

// DefenceGrid

func ErrDefenceGridPositionAlreadyHit(x, y int) error {
	return fmt.Errorf("this position is already hit by the attacker in previous rounds\tx: %d\ty: %d", x, y)
}

func ErrDefenceGridPositionEmpty(x, y int) error {
	return fmt.Errorf("this position in defence grid is empty\tx: %d\ty: %d", x, y)
}

func ErrDefenceGridRowsOutOfBounds(rows, gameGridSize int) error {
	return fmt.Errorf("rows of defence grid must be %d \trows: %d", gameGridSize, rows)
}

func ErrDefenceGridColsOutOfBounds(cols, gameGridSize int) error {
	return fmt.Errorf("cols of defence grid must be %d \tcols: %d", gameGridSize, cols)
}

func ErrSessionNotFound(sessionId string) error {
	return fmt.Errorf("session not found\tID: %s", sessionId)
}
