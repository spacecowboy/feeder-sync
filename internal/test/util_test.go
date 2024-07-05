package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	feederServer "github.com/spacecowboy/feeder-sync/internal/server"
	postgresqlStore "github.com/spacecowboy/feeder-sync/internal/store/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	goodSyncCode = "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	goodDeviceId = int64(1234)
)

func newServerWithOwnDb(t *testing.T, ctx context.Context, container *postgres.PostgresContainer) *feederServer.FeederServer {
	db := postgresqlStore.PostgresStore{
		Db: NewDb(t, ctx, container),
	}

	server, err := feederServer.NewServerWithStore(&db)
	if err != nil {
		server.Close()
		t.Fatalf("It blew up %v", err.Error())
	}

	return server
}

func NewContainer(t *testing.T, ctx context.Context) *postgres.PostgresContainer {
	t.Helper()

	container, err := postgres.RunContainer(
		ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase(dbname),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		WithTmpfs(),
		// postgres.WithInitScripts(
		// 	"../../../migrations_postgres/1_create_tables.up.sql",
		// 	"../../../migrations_postgres/2_create_articles.up.sql",
		// 	"../../../migrations_postgres/3_add_updated_at.up.sql",
		// 	"../../../migrations_postgres/4_create_legacy_feeds.up.sql",
		// 	"../../../migrations_postgres/5_add_updated_at_index.up.sql",
		// ),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)

	if err != nil {
		t.Fatalf("Failed to start postgres container: %s", err.Error())
	}

	// Clean up the container after the test is complete
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	return container
}

func WithTmpfs() testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) {
		req.Tmpfs = map[string]string{"/var/lib/postgresql/data": "rw"}
		req.Env["PGDATA"] = "/var/lib/postgresql/data"
		req.Cmd = []string{
			"postgres",
			"-c",
			// turn off fsync for speed
			"fsync=off",
			"-c",
			// log everything for debugging
			"log_statement=all",
		}
	}
}

func NewDb(t *testing.T, ctx context.Context, container *postgres.PostgresContainer) *sql.DB {
	t.Helper()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get host: %s", err.Error())
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("Failed to get port: %s", err.Error())
	}

	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       user,
		Password:   password,
		Database:   dbname,
		Host:       host,
		Port:       port.Port(),
		Options:    "sslmode=disable",
	}

	migrator := golangmigrator.New("../../migrations_postgres")

	return pgtestdb.New(t, conf, migrator)
}

func newJoinChainRequestV1(t *testing.T, syncCode string, deviceName string) *http.Request {
	jsonRequest := feederServer.JoinChainRequestV1{
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v1/join", bytes.NewReader(jsonBody))

	result.Header.Add("X-FEEDER-ID", syncCode)
	result.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
	return result
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
	body := feederServer.CreateChainRequestV2{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v2/create", bytes.NewReader(jsonBody))
	request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
	return request
}

func parseCreateResponseV2(t *testing.T, response *httptest.ResponseRecorder) feederServer.UserDeviceResponseV2 {
	var got feederServer.UserDeviceResponseV2

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UserDeviceResponse, '%v", response.Body, err)
	}

	return got
}

func createSyncChainV2(t *testing.T, server *feederServer.FeederServer) uuid.UUID {
	request := newCreateRequestV2(t, "device1", "", 0)
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

	var userDevice feederServer.UserDeviceResponseV2 = parseCreateResponseV2(t, response)

	return userDevice.UserId
}

func newJoinChainRequestV2(t *testing.T, userId uuid.UUID, deviceName string) *http.Request {
	jsonRequest := feederServer.JoinChainRequestV2{
		UserId:     userId,
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v2/join", bytes.NewReader(jsonBody))
	result.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
	return result
}

func newMigrateRequestV2(t *testing.T, syncCode string, deviceId int64, deviceName string) *http.Request {
	jsonRequest := feederServer.MigrateRequestV2{
		SyncCode:   syncCode,
		DeviceId:   deviceId,
		DeviceName: deviceName,
	}

	jsonBody, _ := json.Marshal(jsonRequest)
	result, _ := http.NewRequest(http.MethodPost, "/api/v2/migrate", bytes.NewReader(jsonBody))
	result.Header.Add("Cf-Worker", "nononsenseapps.com")
	return result
}

func newCreateRequestV1(t *testing.T, deviceName string) *http.Request {
	body := feederServer.CreateChainRequestV1{
		DeviceName: deviceName,
	}
	jsonBody, _ := json.Marshal(body)
	request, _ := http.NewRequest(http.MethodPost, "/api/v1/create", bytes.NewReader(jsonBody))
	request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
	return request
}

func parseCreateResponseV1(t *testing.T, response *httptest.ResponseRecorder) feederServer.JoinChainResponseV1 {
	var got feederServer.JoinChainResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into JoinChainResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseDevicesResponseV1(t *testing.T, response *httptest.ResponseRecorder) feederServer.DeviceListResponseV1 {
	var got feederServer.DeviceListResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into DeviceListResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseGetReadmarksResponseV1(t *testing.T, response *httptest.ResponseRecorder) feederServer.GetReadmarksResponseV1 {
	var got feederServer.GetReadmarksResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into GetReadmarksResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseGetFeedsResponseV1(t *testing.T, response *httptest.ResponseRecorder) feederServer.GetFeedsResponseV1 {
	var got feederServer.GetFeedsResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into GetFeedsResponseV1, '%v", response.Body, err)
	}

	return got
}

func parseUpdateFeedsResponseV1(t *testing.T, response *httptest.ResponseRecorder) feederServer.UpdateFeedsResponseV1 {
	var got feederServer.UpdateFeedsResponseV1

	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unable to parse response %q into UpdateFeedsResponseV1, '%v", response.Body, err)
	}

	return got
}

func createSyncChainV1(t *testing.T, server *feederServer.FeederServer) feederServer.JoinChainResponseV1 {
	request := newCreateRequestV1(t, "device1")
	request.SetBasicAuth(feederServer.HARDCODED_USER, feederServer.HARDCODED_PASSWORD)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	got := response.Code
	want := 200

	if got != want {
		t.Fatalf("want %d, got %d", want, got)
	}

	var result feederServer.JoinChainResponseV1 = parseCreateResponseV1(t, response)

	return result
}
