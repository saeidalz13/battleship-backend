# Design Proposal

## Problem

With our current data model, each player has a `DefenceGrid` of type `[][]int`. It encapsulates the fact that if a position is occupied by a ship, then the coordinate (that specific x and y in the matrix) is set to 1 and if it's empty is set to 0. However, this lacks the logic of ships and the fact that they have specific length through which their sunken status can be determined. \
In addition, struct `Player` holds a field of `isReady` which is only accessed one and then it is hold in this struct for both players without any use case. Since ready is the logic of the game and `Player` already has a field of `isTurn`, it would not make sense to keep it in `Player` struct. `Player` also needs the logic of whether it has lost the game which can be determined through identifying how many ships it has left unsunken.

## Solution

### Ship

In order to introduce the logic of ships to the game, we can define a struct `Ship` as below:

```Go
type Ship struct {
    Code   int  // This potentially can be removed but for now it makes each struct has an ID
    length int
    hits   int
}
```

Each ship will have a specific code for identification. It captures the idea of `length` (the amount of hits it can take until sunken) and the number of `hits` it has taken.
It will have the receiver functions below to deal with the logic of Attack:

```Go
func (sh *Ship) GotHit() {
	sh.hits++
}

func (sh *Ship) IsSunk() bool {
	return sh.hits == sh.length
}
```

### Player

The idea of player can be defined as below:

```Go
const (
    PlayerMatchStatusLost      = -1
    PlayerMatchStatusUndefined = 0
    PlayerMatchStatusWon       = 1
)

type Player struct {
    Uuid        string
    IsTurn      bool
    IsHost      bool
    MatchStatus int
    SunkenShips int
    AttackGrid  [][]int
    DefenceGrid [][]int
    Ships       map[int]*Ship
    WsConn      *websocket.Conn
}
```

Compared to before, `IsReady` has been removed and will be set to `Game` since being ready is a state that affect both of the players and would be an indicator of game commencement. `IsWinner` also shows us the status after the game ends. \
`SunkenShips` encapsulates the idea of how many ships this player has lost. `Ships` is an array of all the ships provided when the player sending its `DefenceGrid`.

#### Caveat

`DefenceGrid` is currently a matrix of 0s and 1s where 0 represents an empty position and 1 indicates the occupation by a ship. However, in this way we have no idea as to which ship is occupying which position. \
To solve this problem, we can define a set of codes that would help devise the logic of `DefenceGrid`, representing the type of the ship, which will the `Code` field of `Ship`, and Empty and Hit logic. The codes are as below:

```Go
// For Defence Grid
const (
    PositionStateDefenceGridHit   = -1
    PositionStateDefenceGridEmpty = 0
    PositionStateDefenceGridShip1 = 1
    PositionStateDefenceGridShip2 = 2
    PositionStateDefenceGridShip3 = 3
    PositionStateDefenceGridShip4 = 4
)

// For Attack Grid
const (
	PositionStateAttackGridMiss  = -1
	PositionStateAttackGridEmpty = 0
	PositionStateAttackGridHit   = 1
)

const SunkenShipsToLose = 4
```

When the client is sending the `DefenceGrid`, it needs to fill the matrix not only with 1 rather with the corresponding ship code to represent the position occupancy. for example:

```Go
// Assuming grid is 5 by 5
defenceGrid := [][]int{
    []int{0,0,1,1,1},
    []int{4,4,4,4,0},
    []int{0,0,2,2,0},
    []int{0,3,3,3,0},
    []int{0,0,0,0,0},
}
```

This example shows that row 1 is occupied by ship 1, row 2 by ship 4, row 3 by ship 2, row 4 by ship 3, and row 5 is empty.\
If the attacker chooses a coordinate of (1,3) as a target:

```Go
// (1,3) is set to -1 since it was occupied by ship 1 and was hit
// If the position is already hit or there is no ship in the position, it returns an error
shipCode, err := player.GotAttacked(x, y)

// It will set (1,3) position to -1 and returns 1 since ship1 was hit
// defenceGrid := [][]int{
//     []int{0,0,1,-1,1},
//     []int{4,4,4,4,0},
//     []int{0,0,2,2,0},
//     []int{0,3,3,3,0},
//     []int{0,0,0,0,0},
// }

if err != nil {
    // Since we know the occupied ship, we increase hit by one
    // we have checked that codes in the matrix are valid, so this key will ALWAYS have a corresponding ship
    player.Ships[shipCode].GotHit()

    // Now we check if this hit would cause sinking
    if player.Ships[shipCode].IsSunk() {

        // This could potentially have a getter or setter
        player.SunkenShips++
    }

    // Later on we can see if the game is over
    if player.SunkenShips == SunkenShipsToLose {
        // Here goes the logic to end the game
    }
}

```

With this method, we can achieve looking up the coordinate and also change the state of the position in **O(1)** complexity.

### Game

The new `Game` struct will have fields of `IsReady` and `IsFinished`.

```Go
type Game struct {
	Uuid       string
	HostPlayer *Player
	JoinPlayer *Player
    IsReady    bool
    IsFinished bool
}

```

`IsReady` will let the game start on the client side. `IsFinished` will signal the end of them game and the subsequent logics.

