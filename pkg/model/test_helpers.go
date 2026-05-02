//go:build test

package model

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/migrate"
	dbpkg "github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/migrations"
)

//go:embed  fixtures/sample.yaml
var sampleFixture []byte

func NewTestDB(t *testing.T) *bun.DB {
	testDB := dbpkg.NewEmptyTestDB(t)
	ctx := context.Background()

	migrator := migrate.NewMigrator(testDB, migrations.Migrations)

	err := migrator.Init(ctx)
	require.NoError(t, err)

	err = migrator.Lock(ctx)
	require.NoError(t, err)
	defer require.NoError(t, migrator.Unlock(ctx))

	_, err = migrator.Migrate(ctx)
	require.NoError(t, err)

	require.NoError(t, ResetModels(ctx, testDB))

	return testDB
}

func NewTestDBWithFixtures(t *testing.T, fixtures map[string][]byte) *bun.DB {
	db := NewTestDB(t)
	RegisterModels(db)
	err := dbpkg.LoadTestFixtures(context.Background(), db, fixtures, dbfixture.WithTruncateTables())
	require.NoError(t, err)
	return db
}
