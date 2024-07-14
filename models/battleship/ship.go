package battleship

const (
	PositionStateDefenceGridEmpty uint8 = iota
	PositionStateDefenceGridHit
	PositionStateDefenceDestroyer
	PositionStateDefenceCruiser
	PositionStateDefenceBattleship
)

const (
	sunkenShipsToLose uint8 = 3
)

type Ship struct {
	Code           uint8
	length         uint8
	hits           uint8
	hitCoordinates []Coordinates
}

func NewShip(code, length uint8) *Ship {
	return &Ship{
		Code:           code,
		length:         length,
		hits:           0,
		hitCoordinates: make([]Coordinates, 0, length),
	}
}

func NewShipsMap() map[uint8]*Ship {
	ships := make(map[uint8]*Ship, sunkenShipsToLose)
	ships[PositionStateDefenceDestroyer] = NewShip(PositionStateDefenceDestroyer, 2)
	ships[PositionStateDefenceCruiser] = NewShip(PositionStateDefenceCruiser, 3)
	ships[PositionStateDefenceBattleship] = NewShip(PositionStateDefenceBattleship, 4)

	return ships
}

func (sh *Ship) GotHit() {
	sh.hits++
}

func (sh *Ship) IsSunk() bool {
	return sh.hits == sh.length
}

func (sh *Ship) GetHitCoordinates() []Coordinates {
	return sh.hitCoordinates
}
