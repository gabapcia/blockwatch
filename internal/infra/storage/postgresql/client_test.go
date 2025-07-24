package postgresql

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgreSQLContainer(t *testing.T) (*client, func()) {
	t.Helper()

	ctx := t.Context()

	// Get the absolute path to migrations directory
	migrationsPath, err := filepath.Abs("migrations")
	require.NoError(t, err)

	// Start PostgreSQL container
	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:17-alpine",
		postgrescontainer.WithDatabase("blockwatch_test"),
		postgrescontainer.WithUsername("test_user"),
		postgrescontainer.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations using migrate container
	err = runMigrations(ctx, t, postgresContainer, migrationsPath)
	require.NoError(t, err)

	// Create client
	pgClient, err := New(ctx, connStr)
	require.NoError(t, err)

	// Return client and cleanup function
	cleanup := func() {
		pgClient.Close()
		postgresContainer.Terminate(ctx)
	}

	return pgClient, cleanup
}

func runMigrations(ctx context.Context, t *testing.T, postgresContainer *postgrescontainer.PostgresContainer, migrationsPath string) error {
	t.Helper()

	// Get the container's internal IP and use the default PostgreSQL port
	containerIP, err := postgresContainer.ContainerIP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get postgres container IP: %w", err)
	}

	// Build database URL using container IP (container-to-container communication)
	internalDatabaseURL := fmt.Sprintf("postgres://test_user:test_password@%s:5432/blockwatch_test?sslmode=disable", containerIP)

	// Create migrate container with bridge network mode
	migrateContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "migrate/migrate",
			Cmd: []string{
				"-path", "/migrations",
				"-database", internalDatabaseURL,
				"up",
			},
			Files: []testcontainers.ContainerFile{
				{
					HostFilePath:      migrationsPath,
					ContainerFilePath: "/migrations",
					FileMode:          0755,
				},
			},
			NetworkMode: "bridge", // Use bridge network for container-to-container communication
			WaitingFor:  wait.ForExit().WithExitTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create migrate container: %w", err)
	}
	defer migrateContainer.Terminate(ctx)

	// Wait for migration to complete
	exitCode, err := migrateContainer.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container state: %w", err)
	}

	if !exitCode.Running && exitCode.ExitCode != 0 {
		// Get logs for debugging
		logs, _ := migrateContainer.Logs(ctx)
		logBytes := make([]byte, 1024)
		logs.Read(logBytes)
		return fmt.Errorf("migration failed with exit code %d, logs: %s", exitCode.ExitCode, string(logBytes))
	}

	return nil
}

func TestNew(t *testing.T) {
	t.Run("should create client with valid DSN", func(t *testing.T) {
		// Start PostgreSQL container
		postgresContainer, err := postgrescontainer.Run(t.Context(),
			"postgres:17-alpine",
			postgrescontainer.WithDatabase("blockwatch_test"),
			postgrescontainer.WithUsername("test_user"),
			postgrescontainer.WithPassword("test_password"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second)),
		)
		require.NoError(t, err)
		defer postgresContainer.Terminate(t.Context())

		// Get connection string
		connStr, err := postgresContainer.ConnectionString(t.Context(), "sslmode=disable")
		require.NoError(t, err)

		// Test New function
		client, err := New(t.Context(), connStr)
		require.NoError(t, err)
		require.NotNil(t, client)
		defer client.Close()

		// Verify client implements the interface
		var _ Client = client

		// Verify internal components are initialized
		require.NotNil(t, client.pool)
		require.NotNil(t, client.monitoredWallets)
		require.NotNil(t, client.walletwatchIdempotency)
		require.NotNil(t, client.chainstreamCheckpoint)
	})

	t.Run("should return error with invalid DSN", func(t *testing.T) {
		invalidDSN := "invalid://connection/string"

		client, err := New(t.Context(), invalidDSN)
		require.Error(t, err)
		require.Nil(t, client)
	})

	t.Run("should return error with unreachable database", func(t *testing.T) {
		// Use a context with timeout to ensure the test doesn't hang
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()

		unreachableDSN := "postgres://user:pass@localhost:9999/nonexistent?sslmode=disable"

		client, err := New(ctx, unreachableDSN)
		if err == nil {
			// If New doesn't fail immediately, try to ping to force connection
			defer client.Close()
			err = client.pool.Ping(ctx)
		}
		require.Error(t, err)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("should close connection pool successfully", func(t *testing.T) {
		pgClient, cleanup := setupPostgreSQLContainer(t)
		defer cleanup()

		err := pgClient.Close()
		require.NoError(t, err)
	})

	t.Run("should handle Close after connection pool operations", func(t *testing.T) {
		pgClient, cleanup := setupPostgreSQLContainer(t)
		defer cleanup()

		// Test that the connection is working by pinging the database
		err := pgClient.pool.Ping(t.Context())
		require.NoError(t, err)

		err = pgClient.Close()
		require.NoError(t, err)
	})
}
