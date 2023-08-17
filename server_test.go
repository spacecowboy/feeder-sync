package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestJoinSyncChain(t *testing.T) {
	// TODO JSON
	// TODO deviceName in body
	// TODO UserId and DeviceId
	// TODO 401 auth
	t.Run("Joining returns a new UserId and DeviceId", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", nil)
		responseFirst := httptest.NewRecorder()

		FeederServer(responseFirst, request)

		var gotFirst UserDevice

		if err := json.Unmarshal(responseFirst.Body.Bytes(), &gotFirst); err != nil {
			t.Fatalf("Unable to parse response %q into UserDevice, '%v", responseFirst.Body, err)
		}

		if gotFirst.UserId == uuid.Nil {
			t.Errorf("UserId was nil: %q", gotFirst.UserId)
		}

		if gotFirst.DeviceId == uuid.Nil {
			t.Errorf("DeviceId was nil: %q", gotFirst.DeviceId)
		}

		if gotFirst.UserId == gotFirst.DeviceId {
			t.Errorf("UserId should be different from deviceId: %q", gotFirst.UserId)
		}

		response := httptest.NewRecorder()

		// Run again to generate another user
		FeederServer(response, request)

		var gotSecond UserDevice

		if err := json.Unmarshal(response.Body.Bytes(), &gotSecond); err != nil {
			t.Fatalf("Unable to parse response %q into UserDevice, '%v", response.Body, err)
		}

		if gotSecond.UserId == uuid.Nil {
			t.Errorf("UserId was nil: %q", gotSecond.UserId)
		}

		if gotSecond.DeviceId == uuid.Nil {
			t.Errorf("DeviceId was nil: %q", gotSecond.DeviceId)
		}

		if gotSecond.UserId == gotSecond.DeviceId {
			t.Errorf("UserId should be different from deviceId: %q", gotSecond.UserId)
		}

		if gotFirst.UserId == gotSecond.UserId {
			t.Errorf("got %q should be different from %q", gotFirst.UserId, gotSecond.UserId)
		}
	})
}
