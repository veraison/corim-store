package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim-store/pkg/test"
)

func TestLocator_conversion(t *testing.T) {
	ret, err := LocatorsFromCoRIM(nil)
	assert.Nil(t, ret)
	assert.NoError(t, err)

	other, err := LocatorsToCoRIM(nil)
	assert.Nil(t, other)
	assert.NoError(t, err)

	locs := []*Locator{
		{
			Href:       "http://example.com",
			Thumbprint: []*Digest{{}, {}},
		},
	}
	_, err = LocatorsToCoRIM(locs)
	assert.ErrorContains(t, err, "locator at index 0: multiple digests not supported")
}

func TestLocator_Select(t *testing.T) {
	var loc Locator
	db := test.NewTestDB(t)

	err := loc.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	loc.ID = 1
	err = loc.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestLocator_Delete(t *testing.T) {
	var loc Locator
	db := test.NewTestDB(t)

	err := loc.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	loc.ID = 1
	err = loc.Delete(context.Background(), db)
	assert.NoError(t, err)
}
