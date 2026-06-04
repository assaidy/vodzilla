package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get connection string: %v\n", err)
		os.Exit(1)
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	if _, err := testDB.Exec(`
		CREATE TABLE IF NOT EXISTS id_test (
			id SERIAL PRIMARY KEY,
			uuid_col UUID,
			varchar_col VARCHAR
		)
	`); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create test table: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := pgContainer.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

// --- Insert benchmarks ---

func resetTable(b *testing.B) {
	b.Helper()
	if _, err := testDB.Exec("TRUNCATE id_test"); err != nil {
		b.Fatalf("truncate: %v", err)
	}
}

func BenchmarkInsertUuidV4(b *testing.B) {
	resetTable(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := testDB.Exec(`INSERT INTO id_test (uuid_col) VALUES ($1)`, uuid.New()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertUuidV7(b *testing.B) {
	resetTable(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := testDB.Exec(`INSERT INTO id_test (uuid_col) VALUES ($1)`, uuid.Must(uuid.NewV7())); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertUlidInVarcharColumn(b *testing.B) {
	resetTable(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := testDB.Exec(`INSERT INTO id_test (varchar_col) VALUES ($1)`, ulid.Make().String()); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Read benchmarks ---

func readSetup(b *testing.B, insertFn func(int)) {
	b.Helper()
	if _, err := testDB.Exec("DELETE FROM id_test"); err != nil {
		b.Fatalf("delete: %v", err)
	}
	if _, err := testDB.Exec("ALTER SEQUENCE id_test_id_seq RESTART WITH 1"); err != nil {
		b.Fatalf("restart seq: %v", err)
	}
	for i := range 1000 {
		insertFn(i)
	}
}

func BenchmarkReadUuidV4(b *testing.B) {
	readSetup(b, func(i int) {
		if _, err := testDB.Exec(`INSERT INTO id_test (uuid_col) VALUES ($1)`, uuid.New()); err != nil {
			b.Fatal(err)
		}
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var uid uuid.UUID
		if err := testDB.QueryRow(`SELECT uuid_col FROM id_test WHERE id = $1`, (i%1000)+1).Scan(&uid); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUuidV7(b *testing.B) {
	readSetup(b, func(i int) {
		if _, err := testDB.Exec(`INSERT INTO id_test (uuid_col) VALUES ($1)`, uuid.Must(uuid.NewV7())); err != nil {
			b.Fatal(err)
		}
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var uid uuid.UUID
		if err := testDB.QueryRow(`SELECT uuid_col FROM id_test WHERE id = $1`, (i%1000)+1).Scan(&uid); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadUlidFromVarcharColumn(b *testing.B) {
	readSetup(b, func(i int) {
		if _, err := testDB.Exec(`INSERT INTO id_test (varchar_col) VALUES ($1)`, ulid.Make().String()); err != nil {
			b.Fatal(err)
		}
	})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s string
		if err := testDB.QueryRow(`SELECT varchar_col FROM id_test WHERE id = $1`, (i%1000)+1).Scan(&s); err != nil {
			b.Fatal(err)
		}
		if _, err := ulid.Parse(s); err != nil {
			b.Fatal(err)
		}
	}
}
