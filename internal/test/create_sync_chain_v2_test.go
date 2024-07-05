package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
)

func TestCreateSyncChainV2(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong user fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", nil)
		request.SetBasicAuth("foo", feederServer.HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong password fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("When create fails then 500", func(t *testing.T) {
		server, _ := feederServer.NewServerWithStore(
			feederServer.ExplodingStore{},
		)

		request := newCreateRequestV2(t, "device1", "", 0)
		responseFirst := httptest.NewRecorder()

		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 500

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Creating returns a new UserId and DeviceId", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request := newCreateRequestV2(t, "device1", "", 0)
		responseFirst := httptest.NewRecorder()

		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := responseFirst.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		var gotFirst feederServer.UserDeviceResponseV2 = parseCreateResponseV2(t, responseFirst)

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
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		var gotSecond feederServer.UserDeviceResponseV2 = parseCreateResponseV2(t, response)

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})

	t.Run("Create chain with garbage body returns 400 code", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", bytes.NewBufferString("foo"))
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})

	t.Run("If response encoding fails then 500 is returned", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request := newCreateRequestV2(t, "device1", "", 0)
		response := &BadResponseWriter{
			header: make(map[string][]string),
		}

		server.ServeHTTP(response, request)

		got := response.Code
		want := 500

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})

	t.Run("LegacyUserId is kept", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request := newCreateRequestV2(t, "device1", "legacy", 0)
		response := &BadResponseWriter{
			header: make(map[string][]string),
		}

		server.ServeHTTP(response, request)

		got := response.Code
		want := 500

		if got != want {
			t.Errorf("want %d, got %d", want, got)
		}
	})
}
