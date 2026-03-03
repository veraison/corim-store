//go:build test

package model

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	dbpkg "github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/test"
)

//go:embed  fixtures/sample.yaml
var sampleFixture []byte

func NewTestDBWithFixtures(t *testing.T, fixtures map[string][]byte) *bun.DB {
	db := test.NewTestDB(t)
	RegisterModels(db)
	err := dbpkg.LoadTestFixtures(context.Background(), db, fixtures, dbfixture.WithTruncateTables())
	require.NoError(t, err)
	return db
}
