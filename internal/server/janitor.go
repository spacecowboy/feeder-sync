package server

import (
	"context"
	"log"
	"time"
)

type Janitor struct {
	Interval time.Duration
	Stop     chan bool
}

func (j *Janitor) Run(server *FeederServer) {
	ctx := context.Background()
	j.Stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Running janitor")
			err := server.DeleteOldDevices(ctx)
			if err != nil {
				log.Printf("Error deleting old devices: %v", err)
			}
			err = server.DeleteUsersWithoutDevices(ctx)
			if err != nil {
				log.Printf("Error deleting users without devices: %v", err)
			}
			err = server.DeleteFullySyncedArticles(ctx)
			if err != nil {
				log.Printf("Error deleting fully synced articles: %v", err)
			}
		case <-j.Stop:
			return
		}
	}
}
