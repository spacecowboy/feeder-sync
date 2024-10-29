package main

import (
	"log"
	"time"

	"github.com/spacecowboy/feeder-sync/internal/config"
	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	conn, err := config.GetDatabaseConn()
	if err != nil {
		log.Fatalf(err.Error())
	}

	feederServer, err := server.NewServerWithPostgres(conn)
	if err != nil {
		log.Fatalf("main: %v", err)
	}
	defer feederServer.Close()

	// Interval doesn't matter for the janitor in this case
	janitor := server.NewJanitor(time.Hour)
	janitor.RunOnce(feederServer)
}
