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

func NewJanitor(interval time.Duration) *Janitor {
	return &Janitor{
		Interval: interval,
	}
}

func (j *Janitor) Run(server *FeederServer) {
	if j.Interval <= 0 {
		log.Print("No janitor interval set")
		return
	}

	log.Print("Janitor starting...")
	j.Stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.RunOnce(server)
		case <-j.Stop:
			log.Print("Janitor stopped")
			return
		}
	}
}

func (j *Janitor) RunOnce(server *FeederServer) {
	log.Print("Janitor running")
	ctx := context.Background()

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

	log.Print("Janitor done")
}

func (j *Janitor) Halt() {
	j.Stop <- true
}
