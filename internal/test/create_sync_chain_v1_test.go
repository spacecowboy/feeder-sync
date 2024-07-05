package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
)

func TestCreateSyncChainV1(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("Missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		body := feederServer.CreateChainRequestV1{
			DeviceName: "foo",
		}
		jsonBody, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("basic auth wrong user fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		body := feederServer.CreateChainRequestV1{
			DeviceName: "foo",
		}
		jsonBody, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))

		request.SetBasicAuth("foo", feederServer.HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("basic auth wrong password fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		body := feederServer.CreateChainRequestV1{
			DeviceName: "foo",
		}
		jsonBody, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))

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
		request := newCreateRequestV1(t, "device1")
		responseFirst := httptest.NewRecorder()
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)

		server, _ := feederServer.NewServerWithStore(
			feederServer.ExplodingStore{},
		)
		server.ServeHTTP(responseFirst, request)

		gotCode1 := responseFirst.Code
		wantCode1 := 500

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("V1 Create chain with no body returns 400 code", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", nil)
		response := httptest.NewRecorder()
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("V1 Create chain with garbage body returns 400 code", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewBufferString("foo"))
		response := httptest.NewRecorder()
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	})

	t.Run("Creating returns a new UserId and DeviceId", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request := newCreateRequestV1(t, "device1")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		var gotFirst feederServer.JoinChainResponseV1 = parseCreateResponseV1(t, response)

		if gotFirst.SyncCode == "" {
			t.Errorf("syncCode was: %q", gotFirst.SyncCode)
		}

		if gotFirst.DeviceId == 0 {
			t.Errorf("deviceId was: %d", gotFirst.DeviceId)
		}
	})
}
