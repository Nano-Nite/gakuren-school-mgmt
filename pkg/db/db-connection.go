package db

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn
var DBCtx context.Context

func ConnectDB() {
	dbURL := strings.TrimSpace("postgres://" +
		os.Getenv("DATABASE_USERNAME") +
		":" +
		os.Getenv("DATABASE_PASSWORD") +
		"@" +
		os.Getenv("DATABASE_URL") +
		"/" +
		os.Getenv("DATABASE_NAME") +
		"?sslmode=disable",
	)
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set; configure the PostgreSQL connection string in the deployment environment")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	DBCtx = context.Background()
	Conn = conn
	// defer conn.Close(context.Background())

	log.Println("DB Connected")
}
