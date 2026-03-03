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
	flag.BoolVar(&trace, "trace", false, "enable SQL statement tracing")
}
