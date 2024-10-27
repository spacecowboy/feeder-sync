package config

import (
	"fmt"
	"os"
)

const (
	FEEDER_SYNC_POSTGRES_CONN = "FEEDER_SYNC_POSTGRES_CONN"
	DATABASE_URL              = "DATABASE_URL"
	LISTEN_ADDRESS            = "LISTEN_ADDRESS"
)

func GetDatabaseConn() (string, error) {
	conn := os.Getenv(DATABASE_URL)
	if conn == "" {
		// Fallback to the old environment variable
		conn = os.Getenv(FEEDER_SYNC_POSTGRES_CONN)
	}

	if conn == "" {
		return "", fmt.Errorf("%s not set", DATABASE_URL)
	}

	return conn, nil
}

func GetListenAddress() string {
	listenAddress := os.Getenv(LISTEN_ADDRESS)
	if listenAddress == "" {
		listenAddress = ":34217"
	}
	return listenAddress
}
