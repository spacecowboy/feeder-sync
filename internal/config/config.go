package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	FEEDER_SYNC_POSTGRES_CONN = "FEEDER_SYNC_POSTGRES_CONN"
	DATABASE_URL              = "DATABASE_URL"
	LISTEN_ADDRESS            = "LISTEN_ADDRESS"
	JANITOR_INTERVAL          = "JANITOR_INTERVAL"
	READY_CALLBACK_URL        = "READY_CALLBACK_URL"
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

func GetJanitorInterval() (time.Duration, error) {
	interval := os.Getenv(JANITOR_INTERVAL)
	if interval == "" || interval == "-1" {
		return 0, errors.New("janitor interval not set")
	}

	return time.ParseDuration(interval)
}

func GetReadyCallbackUrl() string {
	return os.Getenv(READY_CALLBACK_URL)
}
