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

func TestMigrateV2(t *testing.T) {
	t.Run("Migrate calls store", func(t *testing.T) {
		request := newMigrateRequestV2(t, "foo sync", 999, "bar device")

		response := httptest.NewRecorder()

		store := InMemoryStore{
			userDevices: make(map[string][]UserDevice),
			calls:       make(map[string]int),
		}
		server := &FeederServer{
			store: store,
		}
		server.ServeHTTP(response, request)

		got := response.Code
		want := 204

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}

		calls := store.calls["EnsureMigration"]
		if calls != 1 {
			t.Errorf("EnsureMigration expected 1 call but was %d", calls)
		}
	})
}

func TestJoinSyncChainV2(t *testing.T) {
	t.Run("Join with no body 400", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v2/join", nil)

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v2/join", bytes.NewBufferString("Bad"))

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
		request := newJoinChainRequestV2(t, uuid.New(), "deviceJoin")

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
		userId := createSyncChainV2(t, server)
		request := newJoinChainRequestV2(t, userId, "deviceJoin")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 200

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})
}

func TestCreateSyncChainV2(t *testing.T) {
	// TODO 401 auth
	t.Run("When create fails then 500", func(t *testing.T) {
		request := newCreateRequestV2(t, "device1", "", 0)
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

	t.Run("Creating returns a new UserId and DeviceId", func(t *testing.T) {
		request := newCreateRequestV2(t, "device1", "", 0)
		responseFirst := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}

		var gotFirst UserDeviceResponseV2 = parseCreateResponseV2(t, responseFirst)

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
		request2 := newCreateRequestV2(t, "device2", "", 0)

		server.ServeHTTP(response, request2)

		gotCode2 := responseFirst.Code
		wantCode2 := 200

		if gotCode2 != wantCode2 {
			t.Errorf("want %d, got %d", wantCode2, gotCode2)
		}

		var gotSecond UserDeviceResponseV2 = parseCreateResponseV2(t, response)

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
		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", nil)
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
		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", bytes.NewBufferString("foo"))
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
		request := newCreateRequestV2(t, "device1", "", 0)
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

	t.Run("LegacyUserId is kept", func(t *testing.T) {
		request := newCreateRequestV2(t, "device1", "legacy", 0)
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
			userDevices: make(map[string][]UserDevice),
			calls:       make(map[string]int),
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

func newCreateRequestV2(t *testing.T, deviceName string, legacyUserId string, legacyDeviceId int64) *http.Request {
	body := CreateChainRequestV2{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", bytes.NewReader(jsonBody))
	return request
}

func parseCreateResponseV2(t *testing.T, response *httptest.ResponseRecorder) UserDeviceResponseV2 {
	var got UserDeviceResponseV2

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UserDeviceResponse, '%v", response.Body, err)
	}

	return got
}

func createSyncChainV2(t *testing.T, server *FeederServer) uuid.UUID {
	request := newCreateRequestV2(t, "device1", "", 0)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	got := response.Code
	want := 200

	if got != want {
		t.Fatalf("want %d, got %d", want, got)
	}

	var userDevice UserDeviceResponseV2 = parseCreateResponseV2(t, response)

	return userDevice.UserId
}

func newJoinChainRequestV2(t *testing.T, userId uuid.UUID, deviceName string) *http.Request {
	jsonRequest := JoinChainRequestV2{
		UserId:     userId,
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v2/join", bytes.NewReader(jsonBody))
	return result
}

func newMigrateRequestV2(t *testing.T, syncCode string, deviceId int64, deviceName string) *http.Request {
	jsonRequest := MigrateRequestV2{
		SyncCode:   syncCode,
		DeviceId:   deviceId,
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v2/migrate", bytes.NewReader(jsonBody))
	return result
}

type InMemoryStore struct {
	calls       map[string]int
	userDevices map[string][]UserDevice
}

func (s InMemoryStore) RegisterNewUser(deviceName string) (UserDevice, error) {
	userId := uuid.New()
	devices := make([]UserDevice, 2)

	device := UserDevice{
		UserId:         userId,
		DeviceId:       uuid.New(),
		DeviceName:     deviceName,
		LegacyUserId:   userId.String(),
		LegacyDeviceId: 5, //rand.Int63(),
	}

	devices = append(devices, device)
	s.userDevices[userId.String()] = devices

	return device, nil
}

func (s InMemoryStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error) {
	devices := s.userDevices[userId.String()]

	if devices == nil {
		return UserDevice{}, errors.New("No such user")
	}

	device := UserDevice{
		UserId:     userId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[userId.String()] = devices

	return device, nil
}

func (s InMemoryStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error) {
	devices := s.userDevices[syncCode]

	if devices == nil {
		return UserDevice{}, errors.New("No such user")
	}

	device := UserDevice{
		UserId:     devices[0].UserId,
		DeviceId:   uuid.New(),
		DeviceName: deviceName,
	}

	devices = append(devices, device)
	s.userDevices[syncCode] = devices

	return device, nil
}

func (s InMemoryStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) error {
	s.calls["EnsureMigration"] = 1 + s.calls["EnsureMigration"]
	return nil
}

type ExplodingStore struct{}

func (s ExplodingStore) RegisterNewUser(deviceName string) (UserDevice, error) {
	return UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (UserDevice, error) {
	return UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (UserDevice, error) {
	return UserDevice{}, errors.New("BOOM")
}

func (s ExplodingStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) error {
	return errors.New("BOOM")
}
