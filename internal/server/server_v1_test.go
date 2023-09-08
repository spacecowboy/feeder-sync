package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spacecowboy/feeder-sync/internal/store"
	"golang.org/x/exp/slices"
)

func TestJoinSyncChainV1(t *testing.T) {
	tempdir := t.TempDir()
	server, err := NewSqliteServer(tempdir)
	if err != nil {
		t.Fatalf("It blew up %v", err.Error())
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("Failed to close store: %s", err.Error())
			return
		}
	}()
	goodSyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	goodDeviceId := int64(1234)
	_, err = server.store.EnsureMigration(goodSyncCode, goodDeviceId, "foodevice")
	if err != nil {
		t.Fatalf("Failed to insert device: %s", err.Error())
	}
	userDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
	if err != nil {
		t.Fatalf("Got error: %s", err.Error())
	}

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
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
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Join with no body 400", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join with bad body 400", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewBufferString("Bad"))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 400

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Joining a missing chain 404", func(t *testing.T) {
		request := newJoinChainRequestV1(t, "ffffffffffffffffffffff", "deviceJoin")
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		got := response.Code
		want := 404

		if got != want {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	t.Run("Join a sync chain works", func(t *testing.T) {
		request := newJoinChainRequestV1(t, userDevice.LegacySyncCode, "deviceJoin")
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
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

		devices, err := server.store.GetDevices(userDevice.UserId)

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

func TestFeedsV1(t *testing.T) {
	tempdir := t.TempDir()
	server, err := NewSqliteServer(tempdir)
	if err != nil {
		t.Fatalf("It blew up %v", err.Error())
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("Failed to close store: %s", err.Error())
			return
		}
	}()
	goodSyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	goodDeviceId := int64(1234)
	_, err = server.store.EnsureMigration(goodSyncCode, goodDeviceId, "foodevice")
	if err != nil {
		t.Fatalf("Failed to insert device: %s", err.Error())
	}
	userDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
	if err != nil {
		t.Fatalf("Got error: %s", err.Error())
	}

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/feeds", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
		server := newFeederServer()
		server.ServeHTTP(response, request)

		if want := http.StatusMethodNotAllowed; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}
	})

	t.Run("GET no id in header", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		if want := http.StatusBadRequest; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}
	})

	t.Run("GET no such device", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "0")
		server.ServeHTTP(response, request)

		if want := http.StatusBadRequest; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}

		body := string(response.Body.Bytes())

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("Whole flow with etags", func(t *testing.T) {
		// First post some data
		func() {
			jsonBody, _ := json.Marshal(
				UpdateFeedsRequestV1{
					ContentHash: 1,
					Encrypted:   "foo",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			server.ServeHTTP(response, request)

			if want := http.StatusOK; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
			if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Reponse is not json content-type but: %s", ct)
			}

			var feeds UpdateFeedsResponseV1 = parseUpdateFeedsResponseV1(t, response)

			if feeds.ContentHash != 1 {
				t.Fatalf("Wrong content hash: %d", feeds.ContentHash)
			}
		}()
		// Then get with no etag to get it
		etag := func() string {
			request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			server.ServeHTTP(response, request)

			if want := http.StatusOK; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
			if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Reponse is not json content-type but: %s", ct)
			}

			cacheControl := response.Header().Get("Cache-Control")
			if cacheControl != "private, must-revalidate" {
				t.Errorf("Response has wrong cache control: %q", cacheControl)
			}

			etag := response.Header().Get("ETag")
			if etag == "" {
				t.Fatalf("Etag was empty")
			}

			var feeds GetFeedsResponseV1 = parseGetFeedsResponseV1(t, response)

			if feeds.ContentHash != 1 {
				t.Fatalf("Wrong content hash: %d", feeds.ContentHash)
			}

			if feeds.Encrypted != "foo" {
				t.Fatalf("Wrong content: %s", feeds.Encrypted)
			}

			return etag
		}()

		// Also try with bad etag
		func() {
			request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			request.Header.Add("If-None-Match", "not good value")
			server.ServeHTTP(response, request)

			if want := http.StatusOK; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
			if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Reponse is not json content-type but: %s", ct)
			}

			cacheControl := response.Header().Get("Cache-Control")
			if cacheControl != "private, must-revalidate" {
				t.Errorf("Response has wrong cache control: %q", cacheControl)
			}

			etag := response.Header().Get("ETag")
			if etag == "" {
				t.Fatalf("Etag was empty")
			}
		}()

		// with matching etag
		func() {
			request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			request.Header.Add("If-None-Match", etag)
			server.ServeHTTP(response, request)

			if want := http.StatusNotModified; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
		}()

		// POSTS
		// Then missing etag
		func() {
			jsonBody, _ := json.Marshal(
				UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			server.ServeHTTP(response, request)

			if want := http.StatusPreconditionFailed; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
		}()

		// Wrong etag
		func() {
			jsonBody, _ := json.Marshal(
				UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			request.Header.Add("If-Match", "W/3")
			server.ServeHTTP(response, request)

			if want := http.StatusPreconditionFailed; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
		}()

		// Matching etag
		func() {
			jsonBody, _ := json.Marshal(
				UpdateFeedsRequestV1{
					ContentHash: 3,
					Encrypted:   "foobar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			request.Header.Add("If-Match", etag)
			server.ServeHTTP(response, request)

			if want := http.StatusOK; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
			if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Reponse is not json content-type but: %s", ct)
			}
		}()

		// Star etag
		func() {
			jsonBody, _ := json.Marshal(
				UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
			response := httptest.NewRecorder()

			request.Header.Add("X-FEEDER-ID", goodSyncCode)
			request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", userDevice.LegacyDeviceId))
			request.Header.Add("If-Match", "*")
			server.ServeHTTP(response, request)

			if want := http.StatusOK; response.Code != want {
				t.Fatalf("want %d, got %d", want, response.Code)
			}
			if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Reponse is not json content-type but: %s", ct)
			}
		}()
	})
}

func TestReadMarkV1(t *testing.T) {
	tempdir := t.TempDir()
	server, err := NewSqliteServer(tempdir)
	if err != nil {
		t.Fatalf("It blew up %v", err.Error())
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("Failed to close store: %s", err.Error())
			return
		}
	}()
	goodSyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	goodDeviceId := int64(1234)
	_, err = server.store.EnsureMigration(goodSyncCode, goodDeviceId, "foodevice")
	if err != nil {
		t.Fatalf("Failed to insert device: %s", err.Error())
	}
	userDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
	if err != nil {
		t.Fatalf("Got error: %s", err.Error())
	}

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("GET no id in header", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server := newFeederServer()
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("GET all no such user", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", "somebadcode")

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("GET all no such device", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "9999999")

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		body := string(response.Body.Bytes())

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("GET all but empty", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		var readMarks GetReadmarksResponseV1 = parseGetReadmarksResponseV1(t, response)

		if len(readMarks.ReadMarks) != 0 {
			t.Error("Strange, got read marks in the result")
		}
	})

	t.Run("POST no body", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		server := newFeederServer()
		server.store.EnsureMigration(goodSyncCode, goodDeviceId, "foodevice")
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("POST empty body", func(t *testing.T) {
		jsonRequest := SendReadMarksRequestV1{}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 204

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		// Also check that lastSeen has been updated
		updatedDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= userDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", updatedDevice.LastSeen, userDevice.LastSeen)
		}
	})

	t.Run("POST some items without sync id", func(t *testing.T) {
		jsonRequest := SendReadMarksRequestV1{
			ReadMarks: []SendReadMarkV1{
				{
					Encrypted: "foo",
				},
				{
					Encrypted: "bar",
				},
			},
		}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("POST some items without device id", func(t *testing.T) {
		jsonRequest := SendReadMarksRequestV1{
			ReadMarks: []SendReadMarkV1{
				{
					Encrypted: "foo",
				},
				{
					Encrypted: "bar",
				},
			},
		}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("POST some items invalid device id", func(t *testing.T) {
		jsonRequest := SendReadMarksRequestV1{
			ReadMarks: []SendReadMarkV1{
				{
					Encrypted: "foo",
				},
				{
					Encrypted: "bar",
				},
			},
		}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "99999")
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		body := string(response.Body.Bytes())

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("POST some items", func(t *testing.T) {
		jsonRequest := SendReadMarksRequestV1{
			ReadMarks: []SendReadMarkV1{
				{
					Encrypted: "foo",
				},
				{
					Encrypted: "bar",
				},
				{
					Encrypted: "foo",
				},
			},
		}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 204

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		// Also check that lastSeen has been updated
		preGetDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if preGetDevice.LastSeen <= userDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", preGetDevice.LastSeen, userDevice.LastSeen)
		}

		getRequest, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		getRequest.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		getResponse := httptest.NewRecorder()

		getRequest.Header.Add("X-FEEDER-ID", goodSyncCode)
		getRequest.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))
		server.ServeHTTP(getResponse, getRequest)

		if getResponse.Code != 200 {
			t.Fatalf("want %d, got %d", 200, getResponse.Code)
		}
		if ct := getResponse.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		var readMarks GetReadmarksResponseV1 = parseGetReadmarksResponseV1(t, getResponse)

		if actual := len(readMarks.ReadMarks); actual != 2 {
			t.Errorf("Wrong number of read marks in response: %d", actual)
		}

		if !slices.ContainsFunc[ReadMarkV1](readMarks.ReadMarks, func(readMark ReadMarkV1) bool {
			return readMark.Encrypted == "foo"
		}) {
			t.Error("foo not in result")
		}

		if !slices.ContainsFunc[ReadMarkV1](readMarks.ReadMarks, func(readMark ReadMarkV1) bool {
			return readMark.Encrypted == "bar"
		}) {
			t.Error("bar not in result")
		}

		// Also check that lastSeen has been updated
		updatedDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= preGetDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", updatedDevice.LastSeen, preGetDevice.LastSeen)
		}
	})
}

func TestCreateSyncChainV1(t *testing.T) {
	tempdir := t.TempDir()
	server, err := NewSqliteServer(tempdir)
	if err != nil {
		t.Fatalf("It blew up %v", err.Error())
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("Failed to close store: %s", err.Error())
			return
		}
	}()

	t.Run("Missing basic auth fails 401", func(t *testing.T) {
		body := CreateChainRequestV1{
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
		body := CreateChainRequestV1{
			DeviceName: "foo",
		}
		jsonBody, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))

		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("basic auth wrong password fails 401", func(t *testing.T) {
		body := CreateChainRequestV1{
			DeviceName: "foo",
		}
		jsonBody, _ := json.Marshal(body)
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))

		request.SetBasicAuth(HARDCODED_USER, "foo")

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
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)

		server, _ := NewServerWithStore(
			ExplodingStore{},
		)
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
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)

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
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)

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

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
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

func TestDevicesV1(t *testing.T) {
	tempdir := t.TempDir()
	server, err := NewSqliteServer(tempdir)
	if err != nil {
		t.Fatalf("It blew up %v", err.Error())
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("Failed to close store: %s", err.Error())
			return
		}
	}()
	goodSyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	goodDeviceId := int64(1234)
	_, err = server.store.EnsureMigration(goodSyncCode, goodDeviceId, "foodevice")
	if err != nil {
		t.Fatalf("Failed to insert device: %s", err.Error())
	}
	userDevice, err := server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
	if err != nil {
		t.Fatalf("Got error: %s", err.Error())
	}

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/devices", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/devices", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/api/v1/devices", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("delete missing basic auth fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("delete basic auth wrong user fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", nil)
		request.SetBasicAuth("foo", HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("delete basic auth wrong password fails 401", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", nil)
		request.SetBasicAuth(HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method DELETE handler", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 405

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Uknown path", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/foo/%d", goodDeviceId), nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 404

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method GET handler", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 405

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Get devices", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		devices := parseDevicesResponseV1(t, response)
		if len(devices.Devices) != 1 {
			t.Errorf("Wrong count of devices in result: %d", len(devices.Devices))
		}

		if devices.Devices[0].DeviceId != userDevice.LegacyDeviceId {
			t.Errorf("Expected %d but was %d", userDevice.LegacyDeviceId, devices.Devices[0].DeviceId)
		}
	})

	t.Run("Get devices no such device", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "9099734")
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		body := string(response.Body.Bytes())

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("Delete device no such device", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "900009999")
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}

		body := string(response.Body.Bytes())

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("Delete device", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
		request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", fmt.Sprintf("%d", goodDeviceId))
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 200

		if gotCode1 != wantCode1 {
			t.Fatalf("want %d, got %d", wantCode1, gotCode1)
		}
		if ct := response.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Reponse is not json content-type but: %s", ct)
		}

		devices := parseDevicesResponseV1(t, response)
		if len(devices.Devices) != 0 {
			t.Errorf("Failed to delete device: %d", len(devices.Devices))
		}

		_, err = server.store.GetLegacyDevice(goodSyncCode, goodDeviceId)
		if err != store.ErrNoSuchDevice {
			t.Errorf("Device is still in store: %q", err)
		}

		allDevices, err := server.store.GetDevices(userDevice.UserId)
		if err != nil {
			t.Errorf("What? %s", err.Error())
		}

		if len(allDevices) != 0 {
			t.Errorf("Device count should be 0, not %d", len(allDevices))
		}
	})
}

func newCreateRequestV1(t *testing.T, deviceName string) *http.Request {
	body := CreateChainRequestV1{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))
	request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
	return request
}

func parseCreateResponseV1(t *testing.T, response *httptest.ResponseRecorder) JoinChainResponseV1 {
	var got JoinChainResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into JoinChainResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseDevicesResponseV1(t *testing.T, response *httptest.ResponseRecorder) DeviceListResponseV1 {
	var got DeviceListResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into DeviceListResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseGetReadmarksResponseV1(t *testing.T, response *httptest.ResponseRecorder) GetReadmarksResponseV1 {
	var got GetReadmarksResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into GetReadmarksResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseGetFeedsResponseV1(t *testing.T, response *httptest.ResponseRecorder) GetFeedsResponseV1 {
	var got GetFeedsResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into GetFeedsResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseUpdateFeedsResponseV1(t *testing.T, response *httptest.ResponseRecorder) UpdateFeedsResponseV1 {
	var got UpdateFeedsResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UpdateFeedsResponseV1, '%v", response.Body, err)
	}

	return got
}

func createSyncChainV1(t *testing.T, server *FeederServer) JoinChainResponseV1 {
	request := newCreateRequestV1(t, "device1")
	request.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
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
	result.SetBasicAuth(HARDCODED_USER, HARDCODED_PASSWORD)
	return result
}
