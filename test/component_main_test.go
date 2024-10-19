package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/spacecowboy/feeder-sync/internal/migrations"
	"github.com/spacecowboy/feeder-sync/internal/repository"
	"github.com/spacecowboy/feeder-sync/internal/server"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	user     = "username"
	password = "password"
	dbname   = "feedertest"
)

var (
	connString    string
	listenAddress string
)

// ComponentTestMain sets up the testing framework for other test files.
func TestMain(m *testing.M) {
	// Setup code before running tests
	fmt.Println("Setting up the testing framework...")

	ctx := context.Background()
	container, err := NewContainer(ctx)
	if err != nil {
		fmt.Printf("Failed to start postgres container: %s\n", err.Error())
		os.Exit(1)
	}

	defer container.Terminate(ctx)

	// Wait for container to start
	if err := container.Start(ctx); err != nil {
		fmt.Printf("Failed to start container: %s\n", err.Error())
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Printf("Failed to get host: %s\n", err.Error())
		os.Exit(1)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("Failed to get port: %s\n", err.Error())
		os.Exit(1)
	}

	connString = fmt.Sprintf("postgresql://username:password@%s:%d/feedertest?sslmode=disable", host, port.Int())

	if err := migrations.RunMigrations(connString); err != nil {
		fmt.Printf("Failed to run migrations: %s\n", err.Error())
		os.Exit(1)
	}

	// Start the server
	srv, err := server.NewServerWithPostgres(connString)
	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err.Error())
		os.Exit(1)
	}

	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Printf("Failed to find an available port: %s\n", err.Error())
		os.Exit(1)
	}
	listenAddress = fmt.Sprintf("localhost:%d", listener.Addr().(*net.TCPAddr).Port)
	listener.Close()

	httpPort := fmt.Sprintf(":%d", listener.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := srv.Router.Run(httpPort); err != nil {
			log.Fatalf("Failed to run server: %s", err.Error())
		}
	}()

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Run the tests
	code := m.Run()

	// Teardown code after running tests
	fmt.Println("Tearing down the testing framework...")
	if err := container.Terminate(ctx); err != nil {
		fmt.Printf("Failed to terminate container: %s\n", err.Error())
	}

	// Exit with the code from the tests
	os.Exit(code)
}

func NewContainer(ctx context.Context) (*postgres.PostgresContainer, error) {
	log.Println("Starting postgres container...")
	container, err := postgres.Run(
		ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase(dbname),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		WithTmpfs(),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)

	if err != nil {
		fmt.Printf("Failed to start postgres container: %s\n", err.Error())
		return nil, err
	}

	return container, nil
}

func WithTmpfs() testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
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
		return nil
	}
}

func NewRepository(ctx context.Context, container *postgres.PostgresContainer) *repository.PostgresRepository {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return repository.NewPostgresRepository(conn)
}
