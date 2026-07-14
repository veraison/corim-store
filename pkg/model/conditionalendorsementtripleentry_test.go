package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalEndorsementTripleEntry_Select(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": conditionalAndDomainSampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	cete := ConditionalEndorsementTripleEntry{TripleDbID: 1}
	err := cete.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "sample-manifest", cete.ManifestID)

	expectedEnv := Environment{ID: 1}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	cet, err := cete.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *cet.Conditions[0].Environment)

	manifest, err := cete.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, cete.ManifestID, manifest.ManifestID)

	moduleTag, err := cete.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, cete.ModuleTagID, moduleTag.TagID)
}

func TestConditionalEndorsementTripleEntry_model_methods(t *testing.T) {
	val := ConditionalEndorsementTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "conditional_endorsement_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}

func TestConditionalEndorsementTripleEntry_nok(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)

	val := ConditionalEndorsementTripleEntry{}
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
