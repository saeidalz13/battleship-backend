package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/saeidalz13/battleship-backend/api"
	"github.com/saeidalz13/battleship-backend/db"
	"github.com/saeidalz13/battleship-backend/db/sqlc"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

func main() {
	if os.Getenv("STAGE") != "prod" {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	stage := os.Getenv("STAGE")
	stageCode, err := strconv.ParseInt(stage, 10, 8)
	if err != nil {
		panic(err)
	}
	stageCodeInt8 := uint8(stageCode)
	if stageCodeInt8 != api.DevStageCode && stageCodeInt8 != api.ProdStageCode {
		panic("stage must be either dev or prod")
	}
	port := os.Getenv("PORT")
	psqlUrl := os.Getenv("DATABASE_URL")
	psqlDb := db.MustConnectToDb(psqlUrl, stage)

	queries := sqlc.New(psqlDb)
	dbManager := sqlc.NewDbManager(queries)

	bsm := mc.NewBattleshipSessionManager()
	go bsm.CleanupPeriodically()

	bgm := mb.NewBattleshipGameManager()
	server := api.NewServer(dbManager, bsm, bgm, api.WithPort(port), api.WithStage(stageCodeInt8))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", server.HandleWs)

	log.Printf("Listening to port %s\n", port)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port, mux))
}
