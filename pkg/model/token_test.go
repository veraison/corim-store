package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

func TestToken_CRUD(t *testing.T) {
	corimKey := comid.MustNewCryptoKey(comid.TestECPubKey, comid.PKIXBase64KeyType)
	key, err := NewCryptoKeyFromCoRIM(corimKey)
	require.NoError(t, err)

	token := Token{
		IsSigned:  true,
		Data:      []byte{0xde, 0xad, 0xbe, 0xef},
		Authority: []*CryptoKey{key},
	}

	ctx := context.Background()
	db := test.NewTestDB(t)

	err = token.Insert(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), token.ID)

	other := Token{}
	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")
	err = other.Delete(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	other.ID = 1
	err = other.Select(ctx, db)
	assert.NoError(t, err)
	assert.True(t, token.IsSigned)
	assert.EqualValues(t, key, token.Authority[0])

	err = other.Delete(ctx, db)
	assert.NoError(t, err)

	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "no rows")
}

func TestToken_model(t *testing.T) {
	token := Token{ID: 1}

	assert.Equal(t, int64(1), token.DbID())
	assert.Equal(t, "tokens", token.TableName())
	assert.True(t, token.IsTable())
}
