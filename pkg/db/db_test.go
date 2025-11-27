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

	_, err = Open(&Config{DBMS: "mysql", DSN: "foo", TraceSQL: true})
	assert.ErrorContains(t, err, "invalid DSN")

	_, err = Open(&Config{DBMS: "pg", DSN: "foo", TraceSQL: true})
	assert.NoError(t, err)
}
