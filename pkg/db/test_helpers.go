//go:build test

//coverage:ignore file

package db

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

var defaultTestDbFile = ":memory:"

func NewEmptyTestDB(t *testing.T) *bun.DB {
	testDBMS := strings.ToLower(os.Getenv("TEST_DBMS"))
	switch testDBMS {
	case "sqlite", "sqlite3", "":
		return NewEmptySqlite3TestDB(t)
	case "postgres", "postgresql", "pg":
		return NewEmptyPostgresTestDB(t)
	case "mariadb", "mysql":
		return NewEmptyMariaDBTestDB(t)
	default:
		t.Fatalf("invalid TEST_DBMS: %s", testDBMS)
		return nil
	}
}

func NewEmptyPostgresTestDB(t *testing.T) *bun.DB {
	port := os.Getenv("TEST_DB_POSTGRES_PORT")
	if port == "" {
		port = "55432"
	}

	testDB, err := Open(&Config{
		DBMS: "postgres",
		DSN: fmt.Sprintf(
			"postgres://store_user:L3tM31n@localhost:%s/corim_store?sslmode=disable",
			port,
		),
		TraceSQL: trace,
	})
	require.NoError(t, err)

	return testDB
}

func NewEmptyMariaDBTestDB(t *testing.T) *bun.DB {
	port := os.Getenv("TEST_DB_MYSQL_PORT")
	if port == "" {
		port = "33306"
	}

	testDB, err := Open(&Config{
		DBMS: "mariadb",
		DSN: fmt.Sprintf(
			"store_user:L3tM31n@tcp(localhost:%s)/corim_store?parseTime=true",
			port,
		),
		TraceSQL: trace,
	})
	require.NoError(t, err)

	return testDB
}

func NewEmptySqlite3TestDB(t *testing.T) *bun.DB {
	testDbFile := strings.ReplaceAll(os.Getenv("TEST_DB_FILE"), "@test@", t.Name())
	if testDbFile == "" {
		testDbFile = defaultTestDbFile
	}

	if testDbFile != ":memory:" {
		_ = os.Remove(testDbFile)
	}

	testDB, err := Open(&Config{
		DBMS:     "sqlite",
		DSN:      fmt.Sprintf("file:%s", testDbFile),
		TraceSQL: trace,
	})
	require.NoError(t, err)

	return testDB
}

func LoadTestFixtures(
	ctx context.Context,
	db *bun.DB,
	fixtures map[string][]byte,
	opts ...dbfixture.FixtureOption,
) error {
	names := make([]string, 0, len(fixtures))
	mapFS := make(fstest.MapFS)
	for name, bytes := range fixtures {
		names = append(names, name)
		mapFS[name] = &fstest.MapFile{Data: bytes}
	}

	fixture := dbfixture.New(db, opts...)
	for _, name := range names {
		if err := fixture.Load(ctx, mapFS, name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}

	return nil
}

var trace bool

func init() {
	flag.BoolVar(&trace, "trace-sql", false, "enable SQL statement tracing")
}
