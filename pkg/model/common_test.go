package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/test"
)

func TestRegisterModel(t *testing.T) {
	testDB, err := db.Open(&db.Config{DBMS: "sqlite", DSN: "file::memory:"})
	require.NoError(t, err)
	RegisterModels(testDB)
}

func TestResetModels(t *testing.T) {
	db := test.NewTestDB(t)
	digest := NewDigest(1, []byte{0xde, 0xad, 0xbe, 0xef})
	ctx := context.Background()
	query := db.NewSelect().TableExpr("digests").ColumnExpr("id")

	require.NoError(t, digest.Insert(ctx, db))

	var res1 []int64
	require.NoError(t, query.Scan(ctx, &res1))
	assert.Equal(t, 1, len(res1))

	assert.NoError(t, ResetModels(ctx, db))

	var res2 []int64
	require.NoError(t, query.Scan(ctx, &res2))
	assert.Equal(t, 0, len(res2))
}
