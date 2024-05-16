# Data Structure Proposal

## Problem

With out current data model, each player has a `DefenceGrid` of type `[][]int`. It encapsulates the fact that if a position is occupied by a ship, then the coordinate (that specific x and y in the matrix) is set to 1 and if it's empty is set to 0. However, this lacks the logic of ships and the fact that they have specific length through which it can be determined whether they are sunken.
In addition, struct `Player` holds a field of `isReady` which is only accessed one and then it is hold in this struct without any use case. Since ready is mostly logic of the game and `Player` already has a field of `isTurn`, it would not make sense to keep it in `Player` struct.

## Solution

### Ship logic

In order to introduce the logic of ships to the game, we can define a struct `Ship` as below:

```Go
type Ship struct {
    Code   int
    length int
    hits   int
}
```

Each ship will have a specific code for identification. It captures the idea of `length` (the amount of hits it can take until sunken) and the number of `hits` it has taken.
It will have the receiver functions below to deal with the logic of Attack:

```Go

```
