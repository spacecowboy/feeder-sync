package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GetCallbackUrl makes a GET request to the given URL
// If the URL is empty, it returns nil.
// If the request fails, it returns an error.
func GetCallbackUrl(c context.Context, client *http.Client, url string) error {
	if url == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(c, 1*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("readyCallback req: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("readyCallback do: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("readyCallback do: %d", resp.StatusCode)
	}

	return nil
}
