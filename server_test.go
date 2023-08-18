package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestJoinSyncChain(t *testing.T) {
	t.Run("Join with no body 400", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)

		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join with bad body 400", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewBufferString("Bad"))

		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Joining a missing chain 404", func(t *testing.T) {
		request := newJoinChainRequest(t, uuid.New(), "deviceJoin")

		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 404

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join a sync chain works", func(t *testing.T) {
		server := newFeederServer()
		userId := createSyncChain(t, server)
		request := newJoinChainRequest(t, userId, "deviceJoin")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 200

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})
}

func TestCreateSyncChain(t *testing.T) {
	// TODO 401 auth
	t.Run("Creating returns a new UserId and DeviceId", func(t *testing.T) {
		request := newCreateRequest(t, "device1")
		responseFirst := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}

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

		if gotFirst.DeviceName != "device1" {
			t.Errorf("Got %q, Want %q", gotFirst.DeviceName, "device1")
		}

		response := httptest.NewRecorder()

		// Run again to generate another user
		request2 := newCreateRequest(t, "device2")

		server.ServeHTTP(response, request2)

		gotCode2 := responseFirst.Code
		wantCode2 := 200

		if gotCode2 != wantCode2 {
			t.Errorf("want %d, got %d", wantCode2, gotCode2)
		}

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

		if gotSecond.DeviceName != "device2" {
			t.Errorf("Want %q, Got %q", "device2", gotSecond.DeviceName)
		}
	})

	t.Run("Create chain with no body returns 400 code", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", nil)
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("Create chain with garbage body returns 400 code", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewBufferString("foo"))
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("If response encoding fails then 500 is returned", func(t *testing.T) {
		request := newCreateRequest(t, "device1")
		response := &BadResponseWriter{
			header: make(map[string][]string),
		}

		server := newFeederServer()
		server.ServeHTTP(response, request)

		got := response.Code
		want := 500

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})
}

func newFeederServer() *FeederServer {
	return &FeederServer{
		store: InMemoryStore{
			userDevices: make(map[uuid.UUID][]UserDevice),
		},
	}
}

type BadResponseWriter struct {
	header http.Header
	Code   int
}

func (w *BadResponseWriter) Header() http.Header {
	return w.header
}

func (w *BadResponseWriter) Write([]byte) (int, error) {
	return -1, errors.New("BOOM")
}

func (w *BadResponseWriter) WriteHeader(statusCode int) {
	w.Code = statusCode
}

func newCreateRequest(t *testing.T, deviceName string) *http.Request {
	body := CreateChainRequest{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))
	return request
}

func parseCreateResponse(t *testing.T, response *httptest.ResponseRecorder) UserDevice {
	var got UserDevice

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UserDevice, '%v", response.Body, err)
	}

	return got
}

func createSyncChain(t *testing.T, server *FeederServer) uuid.UUID {
	request := newCreateRequest(t, "device1")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	got := response.Code
	want := 200

	if got != want {
		t.Fatalf("want %d, got %d", want, got)
	}

	var userDevice UserDevice = parseCreateResponse(t, response)

	return userDevice.UserId
}

func newJoinChainRequest(t *testing.T, userId uuid.UUID, deviceName string) *http.Request {
	jsonRequest := JoinChainRequest{
		UserId:     userId,
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewReader(jsonBody))
	return result
}
