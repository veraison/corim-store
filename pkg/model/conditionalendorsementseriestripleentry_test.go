package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalEndorsementSeriesTripleEntry_Select(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": conditionalAndDomainSampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	ceste := ConditionalEndorsementSeriesTripleEntry{TripleDbID: 1}
	err := ceste.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "sample-manifest", ceste.ManifestID)

	expectedEnv := Environment{ID: ceste.EnvironmentID}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	cest, err := ceste.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *cest.Environment)

	manifest, err := ceste.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ceste.ManifestID, manifest.ManifestID)

	moduleTag, err := ceste.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ceste.ModuleTagID, moduleTag.TagID)
}

func TestConditionalEndorsementSeriesTripleEntry_model_methods(t *testing.T) {
	val := ConditionalEndorsementSeriesTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "conditional_endorsement_series_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}

func TestConditionalEndorsementSeriesTripleEntry_nok(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)

	val := ConditionalEndorsementSeriesTripleEntry{}
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
