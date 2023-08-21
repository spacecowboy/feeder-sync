package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJoinSyncChainV1(t *testing.T) {
	// t.Run("Join with no body 400", func(t *testing.T) {
	// 	request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)

	// 	response := httptest.NewRecorder()

	// 	server := newFeederServer()
	// 	server.ServeHTTP(response, request)

	// 	got := response.Code
	// 	want := 400

	// 	if got != want {
	// 		t.Fatalf("want %d, got %d", want, got)
	// 	}
	// })

	// t.Run("Join with bad body 400", func(t *testing.T) {
	// 	request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewBufferString("Bad"))

	// 	response := httptest.NewRecorder()

	// 	server := newFeederServer()
	// 	server.ServeHTTP(response, request)

	// 	got := response.Code
	// 	want := 400

	// 	if got != want {
	// 		t.Fatalf("want %d, got %d", want, got)
	// 	}
	// })

	// t.Run("Joining a missing chain 404", func(t *testing.T) {
	// 	request := newJoinChainRequestV1(t, uuid.New(), "deviceJoin")

	// 	response := httptest.NewRecorder()

	// 	server := newFeederServer()
	// 	server.ServeHTTP(response, request)

	// 	got := response.Code
	// 	want := 404

	// 	if got != want {
	// 		t.Fatalf("want %d, got %d", want, got)
	// 	}
	// })

	t.Run("Join a sync chain works", func(t *testing.T) {
		server := newFeederServer()
		createResponse := createSyncChainV1(t, server)
		request := newJoinChainRequestV1(t, createResponse.SyncCode, "deviceJoin")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 200

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})
}

func TestCreateSyncChainV1(t *testing.T) {
	// TODO 401 auth
	t.Run("When create fails then 500", func(t *testing.T) {
		request := newCreateRequestV1(t, "device1")
		responseFirst := httptest.NewRecorder()

		server := FeederServer{
			store: ExplodingStore{},
		}
		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 500

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("V1 Create chain with no body returns 400 code", func(t *testing.T) {
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

	t.Run("V1 Create chain with garbage body returns 400 code", func(t *testing.T) {
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

	t.Run("Creating returns a new UserId and DeviceId", func(t *testing.T) {
		request := newCreateRequestV1(t, "device1")
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}

		var gotFirst JoinChainResponseV1 = parseCreateResponseV1(t, response)

		if gotFirst.SyncCode == "" {
			t.Errorf("syncCode was: %q", gotFirst.SyncCode)
		}

		if gotFirst.DeviceId == 0 {
			t.Errorf("deviceId was: %d", gotFirst.DeviceId)
		}
	})
}

func newCreateRequestV1(t *testing.T, deviceName string) *http.Request {
	body := CreateChainRequestV1{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))
	return request
}

func parseCreateResponseV1(t *testing.T, response *httptest.ResponseRecorder) JoinChainResponseV1 {
	var got JoinChainResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into JoinChainResponseV1, '%v", response.Body, err)
	}

	return got
}

func createSyncChainV1(t *testing.T, server *FeederServer) JoinChainResponseV1 {
	request := newCreateRequestV1(t, "device1")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	got := response.Code
	want := 200

	if got != want {
		t.Fatalf("want %d, got %d", want, got)
	}

	var result JoinChainResponseV1 = parseCreateResponseV1(t, response)

	return result
}

func newJoinChainRequestV1(t *testing.T, syncCode string, deviceName string) *http.Request {
	jsonRequest := JoinChainRequestV1{
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewReader(jsonBody))

	result.Header.Add("X-FEEDER-ID", syncCode)
	return result
}
