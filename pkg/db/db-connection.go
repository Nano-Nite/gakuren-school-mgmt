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
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
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
