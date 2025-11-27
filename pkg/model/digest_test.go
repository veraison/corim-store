package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim-store/pkg/test"
)

func TestDigest_Select(t *testing.T) {
	var digest Digest
	db := test.NewTestDB(t)

	err := digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	digest.ID = 1
	err = digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestDigest_Delete(t *testing.T) {
	var digest Digest
	db := test.NewTestDB(t)

	err := digest.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	digest.ID = 1
	err = digest.Delete(context.Background(), db)
	assert.NoError(t, err)
}
