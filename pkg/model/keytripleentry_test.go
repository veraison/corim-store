package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyTripleEntry_Select(t *testing.T) {
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": sampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	kte := KeyTripleEntry{TripleDbID: 1}
	err := kte.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "cca-ta", kte.ManifestID)
	assert.Equal(t, "en-GB", *kte.Language)
	assert.Equal(t, kte.TripleDbID, kte.DbID())
	assert.Equal(t, "key_triple_entries", kte.TableName())
	assert.False(t, kte.IsTable())

	expectedEnv := Environment{ID: kte.EnvironmentID}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	vt, err := kte.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *vt.Environment)

	manifest, err := kte.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, kte.ManifestID, manifest.ManifestID)

	moduleTag, err := kte.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, kte.ModuleTagID, moduleTag.TagID)
}
