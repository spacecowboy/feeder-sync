package server

import (
	"log"
	"time"
)

type Janitor struct {
	Interval time.Duration
	Stop     chan bool
}

func (j *Janitor) Run(server *FeederServer) {
	j.Stop = make(chan bool)
	tick := time.Tick(j.Interval)
	for {
		select {
		case <-tick:
			log.Printf("Running janitor")
			err := server.DeleteOldDevices()
			if err != nil {
				log.Printf("Error deleting old devices: %v", err)
			}
		case <-j.Stop:
			return
		}
	}
}
