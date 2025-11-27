package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

func TestKeyTriple_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct {
		title string
		kt    comid.KeyTriple
	}{
		{
			title: "ok",
			kt: comid.KeyTriple{
				Environment: comid.Environment{
					Instance: comid.MustNewUEIDInstance(comid.TestUEID),
				},
				VerifKeys: *comid.NewCryptoKeys().
					Add(comid.MustNewCryptoKey(comid.TestECPubKey, comid.PKIXBase64KeyType)).
					Add(comid.MustNewCryptoKey(comid.TestCert, comid.PKIXBase64CertType)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			kt, err := NewKeyTripleFromCoRIM(&tc.kt)
			assert.NoError(t, err)

			// the type of the KeyTriple isn't part of the struct
			// but is dependent on where it used. Since we do not
			// have the winder context, set it arbitrarily here.
			kt.Type = IdentityKeyTriple

			err = kt.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectKeyTriple(ctx, db, kt.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.kt, selectedCorimTriple)
		})
	}
}

func TestKeyTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeef")
	testCases := []struct {
		title string
		kt    KeyTriple
		err   string
	}{
		{
			title: "ok",
			kt: KeyTriple{
				Type: AttestKeyTriple,
				Environment: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				KeyList: []*CryptoKey{
					&CryptoKey{
						KeyType:  comid.PKIXBase64KeyType,
						KeyBytes: []byte(comid.TestECPubKey),
					},
				},
				AuthorizedBy: []*CryptoKey{
					&CryptoKey{
						KeyType:  comid.PKIXBase64CertType,
						KeyBytes: []byte(comid.TestCert),
					},
				},
			},
		},
		{
			title: "missing type",
			kt:    KeyTriple{},
			err:   "key triple type not set",
		},
		{
			title: "missing environment",
			kt: KeyTriple{
				Type: AttestKeyTriple,
			},
			err: "environment not set",
		},
		{
			title: "missing keys",
			kt: KeyTriple{
				Type: AttestKeyTriple,
				Environment: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				KeyList: []*CryptoKey{},
			},
			err: "empty key list",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.kt.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestKeyTriple_CRUD(t *testing.T) {
	model := "foo"
	kt := KeyTriple{
		Environment: &Environment{
			Model: &model,
		},
		Type: AttestKeyTriple,
		KeyList: []*CryptoKey{
			{
				KeyType:  comid.PKIXBase64KeyType,
				KeyBytes: []byte{0x01, 0x02, 0x03, 0x04},
			},
		},
		AuthorizedBy: []*CryptoKey{
			{
				KeyType:  comid.PKIXBase64KeyType,
				KeyBytes: []byte{0x05, 0x06, 0x07, 0x08},
			},
		},
	}

	db := test.NewTestDB(t)

	err := kt.Insert(context.Background(), db)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), kt.ID)
	assert.Equal(t, int64(1), kt.EnvironmentID)
	assert.Equal(t, int64(1), kt.KeyList[0].OwnerID)
	assert.Equal(t, "key_triple", kt.KeyList[0].OwnerType)
	assert.Equal(t, int64(1), kt.AuthorizedBy[0].OwnerID)
	assert.Equal(t, "key_triple_auth", kt.AuthorizedBy[0].OwnerType)

	err = kt.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestKeyTriple_Select(t *testing.T) {
	var kt KeyTriple
	db := test.NewTestDB(t)

	err := kt.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	kt.ID = 1
	err = kt.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestKeyTriple_Delete(t *testing.T) {
	var kt KeyTriple
	db := test.NewTestDB(t)

	err := kt.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")
}

func TestKeyTriple_accessors(t *testing.T) {
	var kt KeyTriple
	assert.Equal(t, "key", kt.TripleType())
	assert.Equal(t, int64(0), kt.DatabaseID())
}
