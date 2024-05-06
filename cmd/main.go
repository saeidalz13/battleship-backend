package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	// "github.com/saeidalz13/battleship-backend/db"
	"github.com/saeidalz13/battleship-backend/api"
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
	portEnv := os.Getenv("PORT")
	// psqlUrl := os.Getenv("PSQL_URL")
	// DB = db.MustConnectToDb(psqlUrl)
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		panic(err)
	}

	server := api.NewServer(api.WithPort(port), api.WithStage(stage))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", server.HandleWs)

	log.Printf("Listening to port %d\n", port)
	log.Fatalln(http.ListenAndServe(":"+ fmt.Sprintf("%d", port), mux))
}