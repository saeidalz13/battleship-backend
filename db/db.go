package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const (
	maxOpenConns = 300
	maxIdleConns = 100
	connMaxLife  = time.Minute * 15
)

func MustMigrate(db *sql.DB, migrationDir string) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: "battleship",
	})
	if err != nil {
		panic(err)
	}

	migrate, err := migrate.NewWithDatabaseInstance(migrationDir, "battleship", driver)
	if err != nil {
		panic(err)
	}

	version, dirty, err := migrate.Version()
	if err != nil {
		panic(err)
	}
	if dirty {
		panic("database is dirty")
	}
	log.Println("migration version:", version)

	if err = migrate.Up(); err != nil {
		if err.Error() == "no change" {
			return
		}
		panic(err)
	}
	log.Println("migration successful...")
}

func MustConnectToDb(psqlUrl string) *sql.DB {
	// open a database driver or instance 
	// Open may just validate its arguments without creating a connection to the database
	db, err := sql.Open("postgres", psqlUrl)
	if err != nil {
		panic(err)
	}

	// ping db to check connection
	if err := db.Ping(); err != nil {
		panic(err)
	}

	// set db pool custom configs
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLife)

	// there is a 'SchemeFromURL' function that splits the migrationDir by ':', so db/migration will be the URL
	MustMigrate(db, "files:db/migration")
	return db
}
