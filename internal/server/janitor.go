package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/spacecowboy/feeder-sync/internal/repository"
)

type Janitor struct {
	Interval         time.Duration
	Stop             chan bool
	client           *http.Client
	startCallbackUrl string
	endCallbackUrl   string
}

func NewJanitor(
	interval time.Duration,
	startCallbackUrl string,
	endCallbackUrl string,
) *Janitor {
	return &Janitor{
		Interval:         interval,
		client:           &http.Client{},
		startCallbackUrl: startCallbackUrl,
		endCallbackUrl:   endCallbackUrl,
	}
}

func (j *Janitor) Run(repo repository.Repository) {
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
			j.RunOnce(repo)
		case <-j.Stop:
			log.Print("Janitor stopped")
			return
		}
	}
}

func (j *Janitor) RunOnce(repo repository.Repository) {
	log.Print("Janitor running")
	ctx := context.Background()

	if err := GetCallbackUrl(ctx, j.client, j.startCallbackUrl); err != nil {
		log.Printf("startCallback: %v", err)
	}

	err := repo.DeleteOldDevices(ctx)
	if err != nil {
		log.Printf("Error deleting old devices: %v", err)
	}

	err = DeleteUsersWithoutDevices(ctx, repo)
	if err != nil {
		log.Printf("Error deleting users without devices: %v", err)
	}

	err = repo.DeleteFullySyncedArticles(ctx)
	if err != nil {
		log.Printf("Error deleting fully synced articles: %v", err)
	}

	log.Print("Janitor done")

	if err := GetCallbackUrl(ctx, j.client, j.endCallbackUrl); err != nil {
		log.Printf("endCallback: %v", err)
	}
}

func (j *Janitor) Halt() {
	j.Stop <- true
}

func DeleteUsersWithoutDevices(ctx context.Context, repo repository.Repository) error {
	// Get users without devices
	users, err := repo.GetUsersWithoutDevices(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		// Delete user
		if _, err = repo.RemoveUser(ctx, user); err != nil {
			return err
		}
	}

	return nil
}
