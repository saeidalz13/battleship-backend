package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	// "github.com/saeidalz13/battleship-backend/db"
	"github.com/saeidalz13/battleship-backend/api"
)

var DB *sql.DB

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
	// psqlUrl := os.Getenv("PSQL_URL")
	// DB = db.MustConnectToDb(psqlUrl)

	server := api.NewServer(api.WithPort(9191), api.WithStage(stage))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", server.HandleWs)

	log.Println("listening to port 9191...")
	log.Fatalln(http.ListenAndServe(":9191", mux))
}
