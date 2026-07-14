package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueTripleEntry_Select(t *testing.T) {
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": keyAndValueSampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	vte := ValueTripleEntry{TripleDbID: 1}
	err := vte.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "cca-ref-plat", vte.ManifestID)
	assert.Equal(t, "en-GB", *vte.Language)

	expectedEnv := Environment{ID: vte.EnvironmentID}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	vt, err := vte.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *vt.Environment)

	manifest, err := vte.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, vte.ManifestID, manifest.ManifestID)

	moduleTag, err := vte.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, vte.ModuleTagID, moduleTag.TagID)
}

func TestValueTripleEntry_model_methods(t *testing.T) {
	val := ValueTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "value_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}

func TestValueTripleEntry_nok(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)

	val := ValueTripleEntry{}
	err := val.Select(ctx, db)
	assert.ErrorContains(t, err, "TripleDbID not set")

	_, err = val.ToManifest(ctx, db)
	assert.ErrorContains(t, err, "ManifestDbID not set")

	_, err = val.ToModuleTag(ctx, db)
	assert.ErrorContains(t, err, "ModuleTagDbID not set")

	_, err = val.ToTriple(ctx, db)
	assert.ErrorContains(t, err, "TripleDbID not set")

	val.TripleDbID = 1
	err = val.Select(ctx, db)
	assert.ErrorContains(t, err, "no rows in result set")

	val.ManifestDbID = 1
	_, err = val.ToManifest(ctx, db)
	assert.ErrorContains(t, err, "no rows in result set")

	val.ModuleTagDbID = 1
	_, err = val.ToModuleTag(ctx, db)
	assert.ErrorContains(t, err, "no rows in result set")

	_, err = val.ToTriple(ctx, db)
	assert.ErrorContains(t, err, "no rows in result set")
}
