# battleship-backend

Backend server for Battleship IOS game written in Golang. `gorilla/websocket` library is used for websocket connections

## Test

To test the API code:

```bash
make test
```

Alternatively, run `main.go` file to start the server. Then establish a websocket connection using `websocat` application.

```bash
websocat ws://localhost:SERVERPORT/battleship
```
