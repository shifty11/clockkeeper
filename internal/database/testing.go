package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

// SetupPostgreSQLContainer starts a Postgres container for testing and returns a Config.
func SetupPostgreSQLContainer(t *testing.T) *Config {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	return &Config{
		Host:     host,
		Port:     port.Port(),
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
	}
}

// Migrator wraps an Atlas executor to support incremental migration application.
// It tracks applied revisions in memory so successive ApplyN calls work correctly.
type Migrator struct {
	executor *migrate.Executor
}

// NewMigrator creates a Migrator for the given database.
func NewMigrator(t *testing.T, config *Config) *Migrator {
	t.Helper()

	migrationPath, err := findMigrationDir()
	if err != nil {
		t.Fatalf("failed to find migration directory: %v", err)
	}

	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		t.Fatalf("failed to open database for Atlas: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	driver, err := atlaspostgres.Open(db)
	if err != nil {
		t.Fatalf("failed to create Atlas driver: %v", err)
	}

	migrationDir, err := migrate.NewLocalDir(migrationPath)
	if err != nil {
		t.Fatalf("failed to load migration directory: %v", err)
	}

	rrw := &memRevisionReadWriter{}
	executor, err := migrate.NewExecutor(driver, migrationDir, rrw)
	if err != nil {
		t.Fatalf("failed to create Atlas executor: %v", err)
	}

	t.Logf("migrator ready (migrations: %s)", migrationPath)
	return &Migrator{executor: executor}
}

// ApplyN applies n migrations (or all remaining if n < 0).
func (m *Migrator) ApplyN(t *testing.T, n int) {
	t.Helper()
	if err := m.executor.ExecuteN(context.Background(), n); err != nil {
		t.Fatalf("failed to execute migrations (n=%d): %v", n, err)
	}
	t.Logf("applied %d migration(s)", n)
}

// memRevisionReadWriter is an in-memory RevisionReadWriter that tracks
// which migrations have been applied between successive executor calls.
type memRevisionReadWriter []*migrate.Revision

func (*memRevisionReadWriter) Ident() *migrate.TableIdent { return nil }

func (m *memRevisionReadWriter) ReadRevisions(context.Context) ([]*migrate.Revision, error) {
	return []*migrate.Revision(*m), nil
}

func (m *memRevisionReadWriter) ReadRevision(_ context.Context, v string) (*migrate.Revision, error) {
	for _, r := range *m {
		if r.Version == v {
			return r, nil
		}
	}
	return nil, migrate.ErrRevisionNotExist
}

func (m *memRevisionReadWriter) WriteRevision(_ context.Context, r *migrate.Revision) error {
	for i, rev := range *m {
		if rev.Version == r.Version {
			(*m)[i] = r
			return nil
		}
	}
	*m = append(*m, r)
	return nil
}

func (m *memRevisionReadWriter) DeleteRevision(_ context.Context, v string) error {
	for i, r := range *m {
		if r.Version == v {
			*m = slices.Delete(*m, i, i+1)
			return nil
		}
	}
	return migrate.ErrRevisionNotExist
}

// findMigrationDir walks up from cwd to find ent/migrate/migrations/.
func findMigrationDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for range 10 {
		candidate := filepath.Join(dir, "ent", "migrate", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find ent/migrate/migrations/ directory")
}
