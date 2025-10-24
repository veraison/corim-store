package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
	"github.com/veraison/swid"
)

func TestManifest_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testEntities := comid.NewEntities()
	testHashBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	testEntities.Add(&comid.Entity{
		Name:  comid.MustNewEntityName("foo", "string"),
		Roles: *comid.NewRoles().Add(comid.RoleCreator),
	})
	testComid := comid.Comid{
		Language:    &testLanguage,
		Entities:    testEntities,
		TagIdentity: comid.TagIdentity{TagID: *swid.NewTagID("foo"), TagVersion: 1},
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
		},
	}
	testEpoch := time.Unix(0, 0)
	test_cases := []struct {
		title string
		man   *corim.UnsignedCorim
	}{
		{
			title: "ok",
			man: corim.NewUnsignedCorim().
				SetID("bar").
				SetProfile("1.2.3.4").
				AddComid(&testComid).
				AddDependentRim("qux", &swid.HashEntry{
					HashAlgID: swid.Sha256,
					HashValue: testHashBytes,
				}).
				AddEntity("zot", nil, corim.RoleManifestCreator).
				SetRimValidity(time.Now(), &testEpoch),
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			mt, err := NewManifestFromCoRIM(tc.man)
			assert.NoError(t, err)

			err = mt.Insert(ctx, db)
			require.NoError(t, err)

			selectedManifest, err := SelectManifest(ctx, db, mt.ID)
			require.NoError(t, err)

			selectedCoRIM, err := selectedManifest.ToCoRIM()
			assert.NoError(t, err)

			// time.Time has a complicated internal structure that does not
			// get fully preserved on round trip. We don't really care about
			// that as long as the actual timestamps match. So compare the
			// timestamps manually and then set actual validity to expected,
			// so that that we can do a single equality test on the rest of
			// the CoRIM.
			assert.Equal(t,
				tc.man.RimValidity.NotBefore.Unix(),
				selectedCoRIM.RimValidity.NotBefore.Unix(),
			)
			assert.Equal(t,
				tc.man.RimValidity.NotAfter.Unix(),
				selectedCoRIM.RimValidity.NotAfter.Unix(),
			)
			selectedCoRIM.RimValidity = tc.man.RimValidity

			assert.Equal(t, tc.man, selectedCoRIM)
		})
	}
}

func TestManifest_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testModule := ModuleTag{
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
	}
	test_cases := []struct {
		title string
		man   Manifest
		err   string
	}{
		{
			title: "ok",
			man: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "bar",
				ModuleTags:     []*ModuleTag{&testModule},
			},
		},
		{
			title: "no ID type",
			man: Manifest{
				ManifestID: "bar",
			},
			err: "manifest ID not set",
		},
		{
			title: "no ID value",
			man: Manifest{
				ManifestIDType: StringTagID,
			},
			err: "manifest ID not set",
		},
		{
			title: "no module tags",
			man: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "bar",
			},
			err: "no module tags",
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.man.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}
