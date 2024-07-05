package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
)

func TestJoinSyncChainV1(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Join with no body 400", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join with bad body 400", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewBufferString("Bad"))
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Joining a missing chain 404", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request := newJoinChainRequestV1(t, "ffffffffffffffffffffff", "deviceJoin")
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 404

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join a sync chain works", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		goodSyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		goodDeviceId := int64(1234)
		_, err := server.Store.EnsureMigration(ctx, goodSyncCode, goodDeviceId, "foodevice")
		if err != nil {
			t.Fatalf("Failed to insert device: %s", err.Error())
		}
		userDevice, err := server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		request := newJoinChainRequestV1(t, userDevice.LegacySyncCode, "deviceJoin")
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 200

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}

		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		devices, err := server.Store.GetDevices(ctx, userDevice.UserId)

		if err != nil {
			t.Fatalf(err.Error())
		}

		var found bool

		for _, device := range devices {
			if device.DeviceName == "deviceJoin" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Joined successfully but device was not in store")
		}
	})
}
