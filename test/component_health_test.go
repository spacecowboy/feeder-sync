package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Make a request to the /health endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/health", listenAddress))
	if err != nil {
		t.Fatalf("Failed to make request to /health: %s", err.Error())
	}
	defer resp.Body.Close()

	// Check that the status code is 200
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestReadyEndpoint(t *testing.T) {
	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Make a request to the /ready endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/ready", listenAddress))
	if err != nil {
		t.Fatalf("Failed to make request to /ready: %s", err.Error())
	}
	defer resp.Body.Close()

	// Check that the status code is 200
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
