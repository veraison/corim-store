//go:build test

package db

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
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

var trace bool

func init() {
	flag.BoolVar(&trace, "trace", false, "enable SQL statement tracing")
}
