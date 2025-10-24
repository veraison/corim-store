package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

func TestModuleTag_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testEntities := comid.NewEntities()
	testEntities.Add(&comid.Entity{
		Name:  comid.MustNewEntityName("foo", "string"),
		Roles: *comid.NewRoles().Add(comid.RoleCreator),
	})
	test_cases := []struct {
		title string
		mt    comid.Comid
	}{
		{
			title: "ok",
			mt: comid.Comid{
				Language:    &testLanguage,
				Entities:    testEntities,
				TagIdentity: comid.TagIdentity{TagID: *swid.NewTagID("foo"), TagVersion: 1},
				LinkedTags: comid.NewLinkedTags().
					AddLinkedTag(*comid.NewLinkedTag().
						SetLinkedTag(*swid.NewTagID("bar")).
						SetRel(comid.RelSupplements),
					),
				Triples: comid.Triples{
					ReferenceValues: comid.NewValueTriples().Add(&comid.ValueTriple{
						Environment: comid.Environment{
							Instance: comid.MustNewUUIDInstance(comid.TestUUID),
						},
						Measurements: *comid.NewMeasurements().Add(&comid.Measurement{
							Val: comid.Mval{
								RawValue: comid.NewRawValue().
									SetBytes(
										comid.MustHexDecode(
											t,
											"deadbeef")),
							},
						}),
					}),
					EndorsedValues: comid.NewValueTriples().Add(&comid.ValueTriple{
						Environment: comid.Environment{
							Instance: comid.MustNewUUIDInstance(comid.TestUUID),
						},
						Measurements: *comid.NewMeasurements().Add(&comid.Measurement{
							Val: comid.Mval{
								SVN: comid.MustNewTaggedSVN(42),
							},
						}),
					}),
					AttestVerifKeys: &comid.KeyTriples{
						{
							Environment: comid.Environment{
								Instance: comid.MustNewUEIDInstance(
									comid.TestUEID),
							},
							VerifKeys: *comid.NewCryptoKeys().
								Add(comid.MustNewCryptoKey(
									comid.TestECPubKey,
									comid.PKIXBase64KeyType)),
						},
					},
					DevIdentityKeys: &comid.KeyTriples{
						{
							Environment: comid.Environment{
								Instance: comid.MustNewUEIDInstance(
									comid.TestUEID),
							},
							VerifKeys: *comid.NewCryptoKeys().
								Add(comid.MustNewCryptoKey(
									comid.TestCert,
									comid.PKIXBase64CertType)),
						},
					},
				},
			},
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			mt, err := NewModuleTagFromCoRIM(&tc.mt)
			assert.NoError(t, err)

			err = mt.Insert(ctx, db)
			require.NoError(t, err)

			selectedModule, err := SelectModuleTag(ctx, db, mt.ID)
			require.NoError(t, err)

			selectedCoMID, err := selectedModule.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.mt, selectedCoMID)
		})
	}
}

func TestModuleTag_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	test_cases := []struct {
		title string
		mt    ModuleTag
		err   string
	}{
		{
			title: "ok",
			mt: ModuleTag{
				TagIDType: StringTagID,
				TagID:     "foo",
				KeyTriples: []*KeyTriple{
					{
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
					},
				},
			},
		},
		{
			title: "missing tag ID (no type)",
			mt:    ModuleTag{},
			err:   "tag ID not set",
		},
		{
			title: "missing tag ID (no value)",
			mt:    ModuleTag{TagIDType: StringTagID},
			err:   "tag ID not set",
		},
		{
			title: "missing triples",
			mt:    ModuleTag{TagIDType: StringTagID, TagID: "foo"},
			err:   "no triples",
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.mt.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}
