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

func TestManifestEntry(t *testing.T) {
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

	testEpoch := time.Unix(0, 0)
	origin := corim.NewUnsignedCorim().
		SetID("bar").
		SetProfile("1.2.3.4").
		AddComid(&testComid).
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

	manEntry := ManifestEntry{}

	err = manEntry.Select(ctx, db)
	assert.ErrorContains(t, err, "ManifestDbID not set")

	_, err = manEntry.ToManifest(ctx, db)
	assert.ErrorContains(t, err, "ManifestDbID not set")

	manEntry.ManifestDbID = mt.ID

	_, err = manEntry.ToManifest(ctx, db)
	assert.NoError(t, err)

	err = manEntry.Select(ctx, db)
	assert.NoError(t, err)

	assert.Equal(t, mt.Label, manEntry.Label)
	assert.Equal(t, mt.ManifestIDType, manEntry.ManifestIDType)
	assert.Equal(t, mt.ManifestID, manEntry.ManifestID)
	assert.Equal(t, mt.ProfileType, manEntry.ProfileType)
	assert.Equal(t, mt.Profile, manEntry.Profile)
	assert.Equal(t, mt.NotBefore.Unix(), manEntry.NotBefore.Unix())
	assert.Equal(t, mt.NotAfter.Unix(), manEntry.NotAfter.Unix())
}

func TestManifestEntry_model_methods(t *testing.T) {
	val := ManifestEntry{ManifestDbID: 1}
	assert.Equal(t, val.ManifestDbID, val.DbID())
	assert.Equal(t, "manifest_entries", val.TableName())
	assert.False(t, val.IsTable())
}
