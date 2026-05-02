package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDigest_Select(t *testing.T) {
	var digest Digest
	db := NewTestDB(t)

	err := digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	digest.ID = 1
	err = digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
	assert.Equal(t, digest.ID, digest.DbID())
	assert.Equal(t, "digests", digest.TableName())
	assert.True(t, digest.IsTable())
}

func TestDigest_Delete(t *testing.T) {
	var digest Digest
	db := NewTestDB(t)

	err := digest.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	digest.ID = 1
	err = digest.Delete(context.Background(), db)
	assert.NoError(t, err)
}
