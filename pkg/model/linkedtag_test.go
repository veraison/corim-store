package model

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

func TestLinkedTag_conversion(t *testing.T) {
	origin := &comid.LinkedTag{
		LinkedTagID: *swid.NewTagID(uuid.UUID(comid.TestUUID)),
		Rel:         comid.RelReplaces,
	}

	var lt LinkedTag
	err := lt.FromCoRIM(origin)
	assert.NoError(t, err)

	other, err := lt.ToCoRIM()
	assert.NoError(t, err)
	assert.Equal(t, origin, other)

	badCorim := &comid.LinkedTag{
		LinkedTagID: *swid.NewTagID(uuid.UUID(comid.TestUUID)),
		Rel:         comid.Rel(-1),
	}
	err = lt.FromCoRIM(badCorim)
	assert.ErrorContains(t, err, "unexpected tag relation")

	badTag := LinkedTag{
		LinkedTagIDType: "foo",
	}
	_, err = badTag.ToCoRIM()
	assert.ErrorContains(t, err, "unexpected linked tag ID type")

	badTag = LinkedTag{
		LinkedTagIDType: StringTagID,
		LinkedTagID:     "foo",
		TagRelation:     TagRelation("bar"),
	}
	_, err = badTag.ToCoRIM()
	assert.ErrorContains(t, err, "unexpected tag relation")
}

func TestLinkedTag_Select(t *testing.T) {
	var tag LinkedTag
	db := test.NewTestDB(t)

	err := tag.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	_, err = SelectLinkedTag(context.Background(), db, 1)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestLinkedTag_Delete(t *testing.T) {
	var tag LinkedTag
	db := test.NewTestDB(t)

	err := tag.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	tag.ID = 1
	err = tag.Delete(context.Background(), db)
	assert.NoError(t, err)
}
