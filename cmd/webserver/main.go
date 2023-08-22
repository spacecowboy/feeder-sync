package main

import (
	"github.com/spacecowboy/feeder-sync/server"
	"log"
	"net/http"
)

func main() {
	server := server.FeederServer{}
	log.Fatal(http.ListenAndServe(":5000", &server))
}
