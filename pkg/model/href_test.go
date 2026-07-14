package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHref_Select(t *testing.T) {
	var href Href
	db := NewTestDB(t)

	err := href.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	href.ID = 1
	err = href.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestHref_model_methods(t *testing.T) {
	href := Href{ID: 1}
	assert.Equal(t, href.ID, href.DbID())
	assert.Equal(t, "hrefs", href.TableName())
	assert.True(t, href.IsTable())
	assert.Equal(t, href.LocatorID, href.OwnerDbID())
	assert.Equal(t, "locator", href.OwnerName())
}

func TestHerf_Delete(t *testing.T) {
	var href Href
	db := NewTestDB(t)

	err := href.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	href.ID = 1
	err = href.Delete(context.Background(), db)
	assert.NoError(t, err)
}
