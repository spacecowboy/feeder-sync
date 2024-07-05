package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
)

func TestFeedsV1(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth("foo", feederServer.HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("get basic auth wrong password fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/feeds", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "1")

		server.ServeHTTP(response, request)

		if want := http.StatusMethodNotAllowed; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}
	})

	t.Run("GET no id in header", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		if want := http.StatusBadRequest; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}
	})

	t.Run("GET no such device", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)
		request.Header.Add("X-FEEDER-DEVICE-ID", "0")
		server.ServeHTTP(response, request)

		if want := http.StatusBadRequest; response.Code != want {
			t.Fatalf("want %d, got %d", want, response.Code)
		}

		body := response.Body.String()

		// Used by client to self-leave
		if !strings.Contains(body, "Device not registered") {
			t.Errorf("Missing required body so devices can leave: %q", body)
		}
	})

	t.Run("Whole flow with etags", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		_, err := server.Store.EnsureMigration(ctx, goodSyncCode, goodDeviceId, "foodevice")
		if err != nil {
			t.Fatalf("Failed to insert device: %s", err.Error())
		}
		userDevice, err := server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		// First post some data
		func() {
			jsonBody, _ := json.Marshal(
				feederServer.UpdateFeedsRequestV1{
					ContentHash: 1,
					Encrypted:   "foo",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

			var feeds feederServer.UpdateFeedsResponseV1 = parseUpdateFeedsResponseV1(t, response)

			if feeds.ContentHash != 1 {
				t.Fatalf("Wrong content hash: %d", feeds.ContentHash)
			}
		}()
		// Then get with no etag to get it
		etag := func() string {
			request, _ := http.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

			var feeds feederServer.GetFeedsResponseV1 = parseGetFeedsResponseV1(t, response)

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
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
				feederServer.UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
				feederServer.UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
				feederServer.UpdateFeedsRequestV1{
					ContentHash: 3,
					Encrypted:   "foobar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
				feederServer.UpdateFeedsRequestV1{
					ContentHash: 2,
					Encrypted:   "bar",
				},
			)

			request, _ := http.NewRequest(http.MethodPost, "/api/v1/feeds", bytes.NewReader(jsonBody))
			request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
