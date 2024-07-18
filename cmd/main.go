package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/saeidalz13/battleship-backend/api"
	"github.com/saeidalz13/battleship-backend/db"
	"github.com/saeidalz13/battleship-backend/db/sqlc"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
	ms "github.com/saeidalz13/battleship-backend/models/server"
)

func main() {
	if os.Getenv("STAGE") != "prod" {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	stage := os.Getenv("STAGE")
	if stage != ms.DevStageCode && stage != ms.ProdStageCode {
		panic("stage must be either dev or prod")
	}

	port := os.Getenv("PORT")
	psqlUrl := os.Getenv("DATABASE_URL")

	psqlDb := db.MustConnectToDb(psqlUrl)

	querier := sqlc.New(psqlDb)

	bsm := mc.NewBattleshipSessionManager()
	go bsm.CleanupPeriodically()

	bgm := mb.NewBattleshipGameManager()

	requestProcessor := api.NewRequestProcessor(bsm, bgm, querier)

	mux := http.NewServeMux()
	mux.Handle("GET /battleship", requestProcessor)

	log.Printf("Listening to port %s\n", port)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port, mux))
}
