package model

import (
	"context"
	"crypto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

func TestCrypoKey_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct {
		title string
		key   *comid.CryptoKey
	}{
		{
			title: comid.PKIXBase64KeyType,
			key:   comid.MustNewCryptoKey(comid.TestECPubKey, comid.PKIXBase64KeyType),
		},
		{
			title: comid.PKIXBase64CertType,
			key:   comid.MustNewCryptoKey(comid.TestCert, comid.PKIXBase64CertType),
		},
		{
			title: comid.PKIXBase64CertPathType,
			key:   comid.MustNewCryptoKey(comid.TestCertPath, comid.PKIXBase64CertPathType),
		},
		{
			title: comid.COSEKeyType,
			key:   comid.MustNewCryptoKey(comid.TestCOSEKey, comid.COSEKeyType),
		},
		{
			title: comid.ThumbprintType,
			key:   comid.MustNewCryptoKey(comid.TestThumbprint, comid.ThumbprintType),
		},
		{
			title: comid.CertThumbprintType,
			key:   comid.MustNewCryptoKey(comid.TestThumbprint, comid.CertThumbprintType),
		},
		{
			title: comid.CertPathThumbprintType,
			key:   comid.MustNewCryptoKey(comid.TestThumbprint, comid.CertPathThumbprintType),
		},
		{
			title: comid.BytesType,
			key:   comid.MustNewCryptoKey(comid.TestBytes, comid.BytesType),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			key, err := NewCryptoKeyFromCoRIM(tc.key)
			assert.NoError(t, err)

			err = key.Insert(ctx, db)
			assert.NoError(t, err)

			selectedKey, err := SelectCryptoKey(ctx, db, key.ID)
			require.NoError(t, err)

			selectedCorimKey, err := selectedKey.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, tc.key, selectedCorimKey)
		})
	}
}

func TestCryptoKey_Select(t *testing.T) {
	var ck CryptoKey
	db := test.NewTestDB(t)

	err := ck.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ck.ID = 1
	err = ck.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestCryptoKey_Delete(t *testing.T) {
	var ck CryptoKey
	db := test.NewTestDB(t)

	err := ck.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ck.ID = 1
	err = ck.Delete(context.Background(), db)
	assert.NoError(t, err)
}

type testCryptoKey [4]byte

func newTestCryptoKey(_ any) (*comid.CryptoKey, error) {
	return &comid.CryptoKey{Value: &testCryptoKey{0x74, 0x64, 0x73, 0x74}}, nil
}

func (o testCryptoKey) PublicKey() (crypto.PublicKey, error) {
	return crypto.PublicKey(o[:]), nil
}

func (o testCryptoKey) Type() string {
	return "test-crypto-key"
}

func (o testCryptoKey) String() string {
	return "test"
}

func (o testCryptoKey) Valid() error {
	return nil
}

func Test_RegisterCryptoKey(t *testing.T) {
	err := comid.RegisterCryptoKeyType(99998, newTestCryptoKey)
	require.NoError(t, err)

	origin, err := comid.NewCryptoKey(nil, "test-crypto-key")
	require.NoError(t, err)

	ck, err := NewCryptoKeyFromCoRIM(origin)
	assert.NoError(t, err)
	assert.Equal(t, "test-crypto-key", ck.KeyType)
	assert.Equal(t, []byte{
		0xda, 0x00, 0x01, 0x86, 0x9e, // tag(99998)
		0x44,                   // bstr(4)
		0x74, 0x64, 0x73, 0x74, // data
	}, ck.KeyBytes)

	other, err := ck.ToCoRIM()
	assert.NoError(t, err)
	assert.Equal(t, origin, other)
}
