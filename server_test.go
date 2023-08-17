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
		request := newCreateRequest(t)
		responseFirst := httptest.NewRecorder()

		FeederServer(responseFirst, request)

		var gotFirst UserDevice = parseCreateResponse(t, responseFirst)

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

		var gotSecond UserDevice = parseCreateResponse(t, response)

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

func newCreateRequest(t testing.TB) *http.Request {
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", nil)
	return request
}

func parseCreateResponse(t testing.TB, response *httptest.ResponseRecorder) UserDevice {
	var got UserDevice

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UserDevice, '%v", response.Body, err)
	}

	return got
}
