package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
	"github.com/veraison/corim/extensions"
)

type entityExtension struct {
	Foo string `cbor:"-1,keyasint" json:"foo"`
}

func TestEntity_comid_roundtrip(t *testing.T) {
	origin := new(comid.Entity).
		SetName("test-entity").
		SetRegID("http://example.com").
		SetRoles(comid.RoleMaintainer)

	exts := extensions.Map{
		comid.ExtEntity: &entityExtension{},
	}
	err := origin.RegisterExtensions(exts)
	require.NoError(t, err)

	err = origin.Set("foo", "bar")
	require.NoError(t, err)

	entity, err := NewCoMIDEntityFromCoRIM(origin)
	assert.NoError(t, err)

	db := test.NewTestDB(t)
	err = entity.Insert(context.Background(), db)
	assert.NoError(t, err)

	var selection Entity
	err = selection.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	selection.ID = 1
	err = selection.Select(context.Background(), db)
	assert.NoError(t, err)

	assert.Equal(t, selection.Roles(), []string{"maintainer"})

	other, err := selection.ToCoMIDCoRIM()
	assert.NoError(t, err)

	assert.Equal(t, origin.Name, other.Name)
	assert.Equal(t, origin.RegID, other.RegID)
	assert.Equal(t, origin.Roles, other.Roles)
	assert.Equal(t, origin.MustGetString("foo"), other.MustGetString("foo"))

	err = selection.Delete(context.Background(), db)
	assert.NoError(t, err)

	selection.ID = 0
	err = selection.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	missing := Entity{ID: 1}
	err = missing.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows")
}

func TestEntity_corim_roundtrip(t *testing.T) {
	origin := new(corim.Entity).
		SetName("test-entity").
		SetRegID("http://example.com").
		SetRoles(corim.RoleManifestCreator)

	exts := extensions.Map{
		corim.ExtEntity: &entityExtension{},
	}
	err := origin.RegisterExtensions(exts)
	require.NoError(t, err)

	err = origin.Set("foo", "bar")
	require.NoError(t, err)

	entity, err := NewCoRIMEntityFromCoRIM(origin)
	assert.NoError(t, err)

	db := test.NewTestDB(t)
	err = entity.Insert(context.Background(), db)
	assert.NoError(t, err)

	var selection Entity
	err = selection.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	selection.ID = 1
	err = selection.Select(context.Background(), db)
	assert.NoError(t, err)

	other, err := selection.ToCoRIMCoRIM()
	assert.NoError(t, err)

	assert.Equal(t, origin.Name, other.Name)
	assert.Equal(t, origin.RegID, other.RegID)
	assert.Equal(t, origin.Roles, other.Roles)
	assert.Equal(t, origin.MustGetString("foo"), other.MustGetString("foo"))
}
