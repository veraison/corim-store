package model

import (
	"context"
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

	test_cases := []struct {
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

	for _, tc := range test_cases {
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
