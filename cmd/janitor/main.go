package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spacecowboy/feeder-sync/internal/config"
	"github.com/spacecowboy/feeder-sync/internal/repository"
	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	conn, err := config.GetDatabaseConn()
	if err != nil {
		log.Fatalf(err.Error())
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		log.Fatalf("main: %v", err)
	}

	repo := repository.NewPostgresRepository(pool)
	defer repo.Close(ctx)

	// Interval doesn't matter for the janitor in this case
	janitor := server.NewJanitor(time.Hour)
	janitor.RunOnce(repo)
}
