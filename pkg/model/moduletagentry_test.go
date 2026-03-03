package model

import (
	"context"
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

func TestModuleTagEntry(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testUUID := uuid.Must(uuid.Parse(comid.TestUUIDString))
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
	testComid2 := comid.Comid{
		Entities:    testEntities,
		TagIdentity: comid.TagIdentity{TagID: *swid.NewTagID(testUUID), TagVersion: 1},
		Triples: comid.Triples{
			ReferenceValues: comid.NewValueTriples().Add(&testValueTriple),
		},
	}

	testEpoch := time.Unix(0, 0)
	origin := corim.NewUnsignedCorim().
		SetID("bar").
		SetProfile("1.2.3.4").
		AddComid(&testComid).
		AddComid(&testComid2).
		AddDependentRim("qux", &swid.HashEntry{
			HashAlgID: swid.Sha256,
			HashValue: testHashBytes,
		}).
		AddEntity("zot", nil, corim.RoleManifestCreator).
		SetRimValidity(time.Now(), &testEpoch)

	mt, err := NewManifestFromCoRIM(origin)
	require.NoError(t, err)
	err = mt.Insert(ctx, db)
	require.NoError(t, err)

	modEntry := ModuleTagEntry{}

	err = modEntry.Select(ctx, db)
	assert.ErrorContains(t, err, "ModuleTagDbID not set")

	_, err = modEntry.ToManifest(ctx, db)
	assert.ErrorContains(t, err, "ManifestDbID not set")

	_, err = modEntry.ToModuleTag(ctx, db)
	assert.ErrorContains(t, err, "ModuleTagDbID not set")

	modEntry.ModuleTagDbID = 1

	err = modEntry.Select(ctx, db)
	assert.NoError(t, err)

	manifest, err := modEntry.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, mt.ManifestID, manifest.ManifestID)

	_, err = modEntry.ToModuleTag(ctx, db)
	assert.NoError(t, err)

	assert.Equal(t, mt.Label, modEntry.Label)
	assert.Equal(t, mt.ManifestIDType, modEntry.ManifestIDType)
	assert.Equal(t, mt.ManifestID, modEntry.ManifestID)
	assert.Equal(t, mt.ProfileType, modEntry.ProfileType)
	assert.Equal(t, mt.Profile, modEntry.Profile)
	assert.Equal(t, mt.NotBefore.Unix(), modEntry.NotBefore.Unix())
	assert.Equal(t, mt.NotAfter.Unix(), modEntry.NotAfter.Unix())
	assert.Equal(t, testComid.TagIdentity.TagVersion, modEntry.ModuleTagVersion)
	assert.Equal(t, testComid.Language, modEntry.Language)

	modEntry2 := ModuleTagEntry{ModuleTagDbID: 2}
	err = modEntry2.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, UUIDTagID, modEntry2.ModuleTagIDType)
	assert.Equal(t, testUUID.String(), modEntry2.ModuleTagID)

}

func TestModuleTagEntry_model_methods(t *testing.T) {
	val := ModuleTagEntry{ModuleTagDbID: 1}
	assert.Equal(t, val.ModuleTagDbID, val.DbID())
	assert.Equal(t, "module_tag_entries", val.TableName())
	assert.False(t, val.IsTable())
}
