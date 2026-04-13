//go:build test

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/migrations"
)

func NewTestDB(t *testing.T) *bun.DB {
	testDB := db.NewEmptyTestDB(t)
	ctx := context.Background()

	migrator := migrate.NewMigrator(testDB, migrations.Migrations)

	err := migrator.Init(ctx)
	require.NoError(t, err)

	err = migrator.Lock(ctx)
	require.NoError(t, err)
	defer require.NoError(t, migrator.Unlock(ctx))

	_, err = migrator.Migrate(ctx)
	require.NoError(t, err)

	return testDB
}
