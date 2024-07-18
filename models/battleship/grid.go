package battleship

const (
	PositionStateAttackGridEmpty uint8 = iota
	PositionStateAttackGridMiss
	PositionStateAttackGridHit
)

const (
	PositionStateDefenceGridEmpty uint8 = iota
	PositionStateDefenceGridHit

	// Ship codes in defence grid
	PositionStateDefenceDestroyer
	PositionStateDefenceCruiser
	PositionStateDefenceBattleship
)

// Chosse the max uint8 to make the
// mine code unique. Hitting this will
// cause player to lose
const PositionStateMine uint8 = 255

type Coordinates struct {
	X uint8 `json:"x"`
	Y uint8 `json:"y"`
}

func NewCoordinates(x, y uint8) Coordinates {
	return Coordinates{X: x, Y: y}
}

type Grid [][]uint8

// Creates a new default grid
// All indexes are zero/PositionStatusNeutral
func NewGrid(gridSize uint8) Grid {
	grid := make(Grid, gridSize)

	for i := uint8(0); i < gridSize; i++ {
		grid[i] = make([]uint8, gridSize)
	}
	return grid
}
