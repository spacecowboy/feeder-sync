package test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
	"github.com/spacecowboy/feeder-sync/internal/store"
)

func TestDevicesV1(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("get missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
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

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/devices", nil)
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

		request, _ := http.NewRequest(http.MethodPost, "/api/v1/devices", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("delete missing basic auth fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", nil)
		request.SetBasicAuth("foo", feederServer.HARDCODED_PASSWORD)

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("delete basic auth wrong password fails 401", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, "foo")

		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 401

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method DELETE handler", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 405

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Uknown path", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/foo/%d", goodDeviceId), nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 404

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Unsupported method GET handler", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, "/api/v1/devices", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		gotCode1 := response.Code
		wantCode1 := 405

		if gotCode1 != wantCode1 {
			t.Errorf("want %d, got %d", wantCode1, gotCode1)
		}
	})

	t.Run("Get devices", func(t *testing.T) {
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

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
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

		devices := parseDevicesResponseV1(t, response)
		if len(devices.Devices) != 1 {
			t.Errorf("Wrong count of devices in result: %d", len(devices.Devices))
		}

		if devices.Devices[0].DeviceId != userDevice.LegacyDeviceId {
			t.Errorf("Expected %d but was %d", userDevice.LegacyDeviceId, devices.Devices[0].DeviceId)
		}
	})

	t.Run("Get devices no such device", func(t *testing.T) {
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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
		server := newServerWithOwnDb(t, ctx, container)
		defer server.Close()

		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
		request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
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

		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/devices/%d", goodDeviceId), nil)
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

		devices := parseDevicesResponseV1(t, response)
		if len(devices.Devices) != 0 {
			t.Errorf("Failed to delete device: %d", len(devices.Devices))
		}

		_, err = server.Store.GetLegacyDevice(ctx, goodSyncCode, goodDeviceId)
		if err != store.ErrNoSuchDevice {
			t.Errorf("Device is still in store: %q", err)
		}

		allDevices, err := server.Store.GetDevices(ctx, userDevice.UserId)
		if err != nil {
			t.Errorf("What? %s", err.Error())
		}

		if len(allDevices) != 0 {
			t.Errorf("Device count should be 0, not %d", len(allDevices))
		}
	})
}
