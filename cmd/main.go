package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/saeidalz13/battleship-backend/api"
	"github.com/saeidalz13/battleship-backend/db"
)

func main() {
	if os.Getenv("STAGE") != "prod" {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	stage := os.Getenv("STAGE")
	if stage != "dev" && stage != "prod" {
		panic("stage must be either dev or prod")
	}
	port := os.Getenv("PORT")
	psqlUrl := os.Getenv("DATABASE_URL")
	psqlDb := db.MustConnectToDb(psqlUrl)

	server := api.NewServer(api.WithPort(port), api.WithStage(stage), api.WithDb(psqlDb))

	go server.SessionManager.ManageCommunication()
	go server.SessionManager.CleanUpPeriodically()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", server.HandleWs)

	log.Printf("Listening to port %s\n", port)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port, mux))
}
