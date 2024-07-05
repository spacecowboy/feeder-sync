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

func TestReadMarkV1(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
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

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
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

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("post missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
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

		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("GET no id in header", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("GET all no such user", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

		var readMarks feederServer.GetReadmarksResponseV1 = parseGetReadmarksResponseV1(t, response)

		if len(readMarks.ReadMarks) != 0 {
			t.Error("Strange, got read marks in the result")
		}
	})

	t.Run("POST no body", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		request.Header.Add("X-FEEDER-ID", goodSyncCode)

		server.Store.EnsureMigration(ctx, goodSyncCode, goodDeviceId, "foodevice")
		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("POST empty body", func(t *testing.T) {
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

		jsonRequest := feederServer.SendReadMarksRequestV1{}

		jsonBody, _ := json.Marshal(jsonRequest)

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/ereadmark", bytes.NewReader(jsonBody))
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		updatedDevice, err := server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= userDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", updatedDevice.LastSeen, userDevice.LastSeen)
		}
	})

	t.Run("POST some items without sync id", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		jsonRequest := feederServer.SendReadMarksRequestV1{
			ReadMarks: []feederServer.SendReadMarkV1{
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
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 400

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("POST some items without device id", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		jsonRequest := feederServer.SendReadMarksRequestV1{
			ReadMarks: []feederServer.SendReadMarkV1{
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
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		jsonRequest := feederServer.SendReadMarksRequestV1{
			ReadMarks: []feederServer.SendReadMarkV1{
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
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

		jsonRequest := feederServer.SendReadMarksRequestV1{
			ReadMarks: []feederServer.SendReadMarkV1{
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
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		preGetDevice, err := server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if preGetDevice.LastSeen <= userDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", preGetDevice.LastSeen, userDevice.LastSeen)
		}

		getRequest, _ := http.NewRequest(http.MethodGet, "/api/v1/ereadmark", nil)
		getRequest.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

		var readMarks feederServer.GetReadmarksResponseV1 = parseGetReadmarksResponseV1(t, getResponse)

		if actual := len(readMarks.ReadMarks); actual != 2 {
			t.Errorf("Wrong number of read marks in response: %d", actual)
		}

		// Also check that lastSeen has been updated
		updatedDevice, err := server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != nil {
			t.Fatalf("Got error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= preGetDevice.LastSeen {
			t.Errorf("Last seen has not been updated: %d vs %d", updatedDevice.LastSeen, preGetDevice.LastSeen)
		}
	})
}
