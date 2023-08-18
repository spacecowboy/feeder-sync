package main

import (
	"log"
	"net/http"
)

func main() {
	server := FeederServer{}
	log.Fatal(http.ListenAndServe(":5000", &server))
}
