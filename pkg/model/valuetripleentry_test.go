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
		"sample.yaml": sampleFixture,
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
}

func TestValueTripleEntry_model_methods(t *testing.T) {
	val := ValueTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "value_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}
