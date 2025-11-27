package model

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
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
	testValueTriple := comid.ValueTriple{
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
	}
	testComid := comid.Comid{
		Language:    &testLanguage,
		Entities:    testEntities,
		TagIdentity: comid.TagIdentity{TagID: *swid.NewTagID("foo"), TagVersion: 1},
		Triples: comid.Triples{
			ReferenceValues: comid.NewValueTriples().Add(&testValueTriple),
		},
	}
	testComidCondEnd := comid.Comid{
		Language:    &testLanguage,
		Entities:    testEntities,
		TagIdentity: comid.TagIdentity{TagID: *swid.NewTagID("foo"), TagVersion: 1},
		Triples: comid.Triples{
			CondEndorseSeries: comid.NewCondEndorseSeriesTriples().
				Add(&comid.CondEndorseSeriesTriple{
					Condition: comid.StatefulEnv(testValueTriple),
					Series:    *comid.NewCondEndorseSeriesRecords(),
				}),
		},
	}
	testEpoch := time.Unix(0, 0)
	testCases := []struct {
		title string
		man   *corim.UnsignedCorim
		err   string
	}{
		{
			title: "ok string ID/OID profile",
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
		{
			title: "ok UUID ID/URI profile",
			man: corim.NewUnsignedCorim().
				SetID(uuid.UUID(comid.TestUUID)).
				SetProfile("http://example.com").
				AddComid(&testComid).
				SetRimValidity(time.Now(), &testEpoch),
		},
		{
			title: "nok CoSWID tag",
			man: corim.NewUnsignedCorim().
				SetID(uuid.UUID(comid.TestUUID)).
				SetProfile("http://example.com").
				AddCoswid(&swid.SoftwareIdentity{
					Entities: swid.Entities{
						swid.Entity{},
					},
				}),
			err: "tag 505 at index 0",
		},
		{
			title: "nok conditional endorsement series in CoMID",
			man: corim.NewUnsignedCorim().
				SetID(uuid.UUID(comid.TestUUID)).
				AddComid(&testComidCondEnd),
			err: "conditional endorsement series are not supported",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			mt, err := NewManifestFromCoRIM(tc.man)

			if tc.err == "" {
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
			} else {
				// tc.err != nil
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestManifest_ToCoRIM_nok(t *testing.T) {
	testCases := []struct {
		title    string
		manifest Manifest
		err      string
	}{
		{
			title: "bad UUID",
			manifest: Manifest{
				ManifestIDType: UUIDTagID,
				ManifestID:     "foo",
			},
			err: "invalid UUID",
		},
		{
			title: "unexpected ID type",
			manifest: Manifest{
				ManifestIDType: "foo",
			},
			err: "unexpected manifest ID type",
		},
		{
			title: "bad profile",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				Profile:        "@@@@/",
			},
			err: "profile string must be an absolute URL or an ASN.1 OID",
		},
		{
			title: "bad dependent RIMs",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				DependentRIMs: []*Locator{
					{
						Href:       "http://example.com",
						Thumbprint: []*Digest{{}, {}},
					},
				},
			},
			err: "locator at index 0",
		},
		{
			title: "bad entities",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				Entities:       []*Entity{{}},
			},
			err: "entity at index 0",
		},
		{
			title: "bad module tag",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				ModuleTags:     []*ModuleTag{{}},
			},
			err: "module tag at index 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err := tc.manifest.ToCoRIM()
			assert.ErrorContains(t, err, tc.err)
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
	testCases := []struct {
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

	for _, tc := range testCases {
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

func TestManifest_SetActive(t *testing.T) {
	manifest := Manifest{
		ModuleTags: []*ModuleTag{
			&ModuleTag{KeyTriples: []*KeyTriple{{}}},
			&ModuleTag{ValueTriples: []*ValueTriple{{}}},
		},
	}

	manifest.SetActive(true)
	assert.True(t, manifest.ModuleTags[0].KeyTriples[0].IsActive)
	assert.True(t, manifest.ModuleTags[1].ValueTriples[0].IsActive)
}

func TestManifest_Insert(t *testing.T) {
	db := test.NewTestDB(t)
	testModuleTag := ModuleTag{
		TagIDType: StringTagID,
		TagID:     "bar",
		ValueTriples: []*ValueTriple{
			{
				Type:         ReferenceValueTriple,
				Environment:  &Environment{},
				Measurements: []*Measurement{{}},
			},
		},
	}
	testCases := []struct {
		title    string
		manifest Manifest
		err      string
	}{
		{
			title: "ok manifest with extensions",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				ModuleTags:     []*ModuleTag{&testModuleTag},
				Entities: []*Entity{
					{
						RoleEntries: []RoleEntry{{}},
					},
				},
				DependentRIMs: []*Locator{{}},
				Extensions: []*ExtensionValue{
					{
						FieldKind: reflect.Int64,
						FieldName: "zot",
						ValueInt:  7,
					},
				},
			},
		},
		{
			title:    "nok invalid manifest",
			manifest: Manifest{},
			err:      "manifest ID not set",
		},
		{
			title: "nok entities insert error",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "qux",
				Entities: []*Entity{
					{
						ID:          1,
						RoleEntries: []RoleEntry{{}},
					},
				},
				ModuleTags: []*ModuleTag{&testModuleTag},
			},
			err: "UNIQUE constraint failed: entities.id",
		},
		{
			title: "nok dependent RIMs insert error",
			manifest: Manifest{
				ManifestIDType: StringTagID,
				ManifestID:     "qux",
				DependentRIMs:  []*Locator{{ID: 1}},
				ModuleTags:     []*ModuleTag{&testModuleTag},
			},
			err: "UNIQUE constraint failed: locators.id",
		},
		{
			title: "nok duplicate database ID",
			manifest: Manifest{
				ID:             1,
				ManifestIDType: StringTagID,
				ManifestID:     "foo",
				ModuleTags:     []*ModuleTag{&testModuleTag},
			},
			err: "UNIQUE constraint failed: manifests.id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			testModuleTag.ID = 0
			testModuleTag.ValueTriples[0].ID = 0
			testModuleTag.ValueTriples[0].Environment.ID = 0
			testModuleTag.ValueTriples[0].Measurements[0].ID = 0

			err := tc.manifest.Insert(context.Background(), db)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManifest_Select_nok(t *testing.T) {
	var manifest Manifest
	err := manifest.Select(context.TODO(), nil)
	assert.ErrorContains(t, err, "ID not set")
}

func TestManifest_Delete(t *testing.T) {
	var manifest Manifest
	db := test.NewTestDB(t)

	err := manifest.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	manifest.ID = 1
	err = manifest.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestNewManifestFromCoRIM_nil_input(t *testing.T) {
	_, err := NewManifestFromCoRIM(nil)
	assert.ErrorContains(t, err, "nil input")
}

func TestSelectManifest_bad(t *testing.T) {
	db := test.NewTestDB(t)
	_, err := SelectManifest(context.Background(), db, 1)
	assert.ErrorContains(t, err, "no rows")
}
