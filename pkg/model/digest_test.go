package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim/comid"
)

func TestDigest_convervsion_text(t *testing.T) {
	bytes := []byte{0xde, 0xad, 0xbe, 0xef}
	origin := comid.Digests{*comid.NewDigestStringAlg("foo", bytes)}

	digests, err := DigestsFromCoRIM(&origin)
	assert.NoError(t, err)
	assert.Equal(t, "foo", digests[0].AlgIDText)
	assert.Equal(t, bytes, digests[0].Value)

	out, err := DigestsToCoRIM(digests)
	assert.NoError(t, err)
	assert.Equal(t, "foo", (*out)[0].Algorithm.String())
	assert.Equal(t, bytes, (*out)[0].Value)
}

func TestDigest_algorithms(t *testing.T) {
	bytes := []byte{0xde, 0xad, 0xbe, 0xef}
	digest := NewDigestText("foo", bytes)
	assert.Equal(t, "foo", digest.AlgID())
	assert.Equal(t, "foo", digest.AlgIDString())

	digest = NewDigestInt(1, bytes)
	assert.Equal(t, int64(1), digest.AlgID())
	assert.Equal(t, "1", digest.AlgIDString())
}

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
