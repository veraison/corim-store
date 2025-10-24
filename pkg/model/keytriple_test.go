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

	test_cases := []struct {
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

	for _, tc := range test_cases {
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
	test_cases := []struct {
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

	for _, tc := range test_cases {
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
