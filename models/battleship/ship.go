package battleship

const (
	PositionStateDefenceGridHit    = -1
	PositionStateDefenceGridEmpty  = 0
	PositionStateDefenceDestroyer  = 2
	PositionStateDefenceCruiser    = 3
	PositionStateDefenceBattleship = 4

	SunkenShipsToLose = 3
)

type Ship struct {
	Code   int
	length int
	hits   int
}

func NewShip(code, length int) *Ship {
	return &Ship{
		Code:   code,
		length: length,
		hits:   0,
	}
}

func NewShipsMap() map[int]*Ship {
	ships := make(map[int]*Ship, SunkenShipsToLose)
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
