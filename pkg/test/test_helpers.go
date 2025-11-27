//go:build test

package test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/migrations"
)

var defaultTestDbFile = ":memory:"

func NewTestDB(t *testing.T) *bun.DB {
	testDbFile := os.Getenv("TEST_DB_FILE")
	if testDbFile == "" {
		testDbFile = defaultTestDbFile
	}

	if testDbFile != ":memory:" {
		_ = os.Remove(testDbFile)
	}

	testDB, err := db.Open(&db.Config{
		DBMS:     "sqlite",
		DSN:      fmt.Sprintf("file:%s", testDbFile),
		TraceSQL: trace,
	})
	require.NoError(t, err)

	ctx := context.Background()

	migrator := migrate.NewMigrator(testDB, migrations.Migrations)

	err = migrator.Init(ctx)
	require.NoError(t, err)

	err = migrator.Lock(ctx)
	require.NoError(t, err)
	defer require.NoError(t, migrator.Unlock(ctx))

	_, err = migrator.Migrate(ctx)
	require.NoError(t, err)

	return testDB
}

var trace bool

func init() {
	flag.BoolVar(&trace, "trace", false, "enable SQL statement tracing")
}
