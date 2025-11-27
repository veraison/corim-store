package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/db"
)

func TestRegisterModel(t *testing.T) {
	testDB, err := db.Open(&db.Config{DBMS: "sqlite", DSN: "file::memory:"})
	require.NoError(t, err)
	RegisterModels(testDB)
}
