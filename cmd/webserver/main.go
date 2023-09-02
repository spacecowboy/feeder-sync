package main

import (
	"log"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %q", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			return
		}
	}()

	wrappedServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(server, w, r)
		log.Printf(
			"%s %s (code=%d dt=%s written=%d)",
			r.Method,
			r.URL,
			m.Code,
			m.Duration,
			m.Written,
		)
	},
	)

	addr := ":34217"
	log.Printf("Serving on %q", addr)
	log.Fatal(http.ListenAndServe(addr, wrappedServer))
}
