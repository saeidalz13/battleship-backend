package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	s := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Listening to port %s\n", port)
		log.Fatalln(s.ListenAndServe())
	}()

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Println("Server termination signal from OS, graceful shutdown\treason:", sig)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	s.Shutdown(ctx)
}
