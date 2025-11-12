package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_Create(t *testing.T) {
	_, err := Open(&Config{DBMS: "sqlite3", DSN: "file::memory:?cache=shared", TraceSQL: true})
	assert.NoError(t, err)

	_, err = Open(&Config{DBMS: "foo"})
	assert.EqualError(t, err, "unsupported DBMS: foo")
}
