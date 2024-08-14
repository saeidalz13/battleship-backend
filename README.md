<img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/ShipGame/Battleship/Assets.xcassets/AppIcon.appiconset/Battleship%20Logo%20v1%20iOS%402x%20alpha.png" alt="drawing" width="200" height="200" class="center"/>

# battleship-backend

Welcome to the **Battleship Game Backend**! This repository contains a self-hosted backend server written in Go for the classic Battleship game.

## Features

- **Multiplayer Support:** Play the classic Battleship game with your friends.
- **Websocket API:** Easy-to-use API for interacting with the game server.
- **Game State Persistence:** Ability to continue playing within 2-minute period of grace period (abnormal closure).
- **Real-Time Gameplay:** Smooth, responsive gameplay with real-time updates.
- **Docker Support:** Easily deploy the server using Docker.

## Screenshots

| Waiting Room | Choose Difficulty | Place Ships | Game Play I |
|--------------|--------------|--------------|--------------|
| <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/1.jpeg" width="200"/> | <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/2.jpeg" width="200"/> | <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/3.jpeg" width="200"/> | <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/4.jpeg" width="200"/> |

| Game Play II | End Match | Rematch |              |
|--------------|--------------|--------------|--------------|
| <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/5.jpeg" width="200"/> | <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/6.jpeg" width="200"/> | <img src="https://github.com/mori-ahk/Battleship-iOS/blob/main/Screenshots/7.jpeg" width="200"/> |


## Requirements

- Go (version 1.22 or later)
- Docker (optional, for containerized deployment)

## Installation

### Clone the Repository

```bash
git clone https://github.com/yourusername/battleship-backend.git
cd battleship-backend
```

### Build Server

Ensure you have Go installed on your machine. Then, you can build the server with:

```bash
go build -o battleship_server cmd/main.go
```

### Run Server

After building, you can start the server with:

```bash
./battleship-server
```

## Usage

### Websocat

Connect to the server with:

```bash
websocat ws://127.0.0.1:1313/battleship
```

Upon a successful websocket connection, you should receive a request as below:

```bash
{"code":0,"payload":{"session_id":"MjhkOTUzNWEtYjYxNC00MjM1LTk2YTgtZTRmMWEyYWNlYjIz"}}
```

where `session_id` is your unique id used for reconnection to the server upon abnormal closure.
To know what each `code` represent in this api, refer to `models/connection/signal.go`. Through
using the correct code, you can then create a game, select a grid, and attack the opponent.

In case of abnormal closure and wanting to reconnect to resume the game:

```bash
# note that session came from above
websocat ws://127.0.0.1:1313/battleship\?sessionID=MjhkOTUzNWEtYjYxNC00MjM1LTk2YTgtZTRmMWEyYWNlYjIz
```

For a smooth experience of gaming, a frontend is required which you can find here:

**[Frontend Swift Repo](https://github.com/mori-ahk/Battleship-iOS)** 🍏


## Environment Variables

You can configure the server using the following environment variables:

**STAGE:** Represents the stage of development. Choice of `dev` or `prod` (refer to `models/server/stage.go`)
**PORT:** The port on which the server will run (default is 8080).
**DATABASE_URL:** The connection string for the database (if applicable).

## Testing

To test the API, you can use `Makefile` templates:
```bash
# Equivalent to `go test -v ./test -count=1`  // count=1 bypasses the cached info
make test
```

## License

This project is licensed under the MIT License. See the `LICENSE` file for more details.


## Contact

For any questions or inquiries, feel free to reach out to `saeidalz96@gmail.com` or open an issue on GitHub.