package db

import (
	"context"
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

func TestExecTx(t *testing.T) {
	db := NewEmptyTestDB(t)

	_, err := db.Exec(`CREATE TABLE test (col1 TEXT, col2 INT) STRICT;`)
	assert.NoError(t, err)

	bad := []string{
		`INSERT INTO test(col1, col2) VALUES ("foo", 1);`,
		`INSERT INTO test(col1, col2) VALUES ("bar", "qux");`, // bad col2 type
		`INSERT INTO test(col1, col2) VALUES ("baz", 3);`,
	}

	good := []string{
		`INSERT INTO test(col1, col2) VALUES ("foo", 1);`,
		`INSERT INTO test(col1, col2) VALUES ("bar", 2);`,
		`INSERT INTO test(col1, col2) VALUES ("baz", 3);`,
	}

	ctx := context.Background()
	err = ExecTx(ctx, db, nil, bad)
	assert.ErrorContains(t, err, "cannot store TEXT value in INT column")

	var numRows []int
	err = db.NewSelect().Model(&numRows).Table("test").ColumnExpr("COUNT(*)").Scan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, numRows[0])

	err = ExecTx(ctx, db, nil, good)
	assert.NoError(t, err)

	err = db.NewSelect().Model(&numRows).Table("test").ColumnExpr("COUNT(*)").Scan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, numRows[0])
}
