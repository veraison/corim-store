package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainDependencyTripleEntry_Select(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": conditionalAndDomainSampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	ddte := DomainDependencyTripleEntry{TripleDbID: 1}
	err := ddte.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "sample-manifest", ddte.ManifestID)

	expectedEnv := Environment{ID: ddte.EnvironmentID}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	ddt, err := ddte.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *ddt.DomainID)

	manifest, err := ddte.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ddte.ManifestID, manifest.ManifestID)

	moduleTag, err := ddte.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ddte.ModuleTagID, moduleTag.TagID)
}

func TestDomainDependencyTripleEntry_model_methods(t *testing.T) {
	val := DomainDependencyTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "domain_dependency_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}

func TestDomainMembershipTripleEntry_Select(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDBWithFixtures(t, map[string][]byte{
		"sample.yaml": conditionalAndDomainSampleFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	ddte := DomainMembershipTripleEntry{TripleDbID: 1}
	err := ddte.Select(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, "sample-manifest", ddte.ManifestID)

	expectedEnv := Environment{ID: ddte.EnvironmentID}
	err = expectedEnv.Select(ctx, db)
	require.NoError(t, err)

	ddt, err := ddte.ToTriple(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, *ddt.DomainID)

	manifest, err := ddte.ToManifest(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ddte.ManifestID, manifest.ManifestID)

	moduleTag, err := ddte.ToModuleTag(ctx, db)
	assert.NoError(t, err)
	assert.Equal(t, ddte.ModuleTagID, moduleTag.TagID)
}

func TestDomainMembershipTripleEntry_model_methods(t *testing.T) {
	val := DomainMembershipTripleEntry{TripleDbID: 1}
	assert.Equal(t, val.TripleDbID, val.DbID())
	assert.Equal(t, "domain_membership_triple_entries", val.TableName())
	assert.False(t, val.IsTable())
}

func TestDomainDependencyTripleEntry_nok(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)

	val := DomainDependencyTripleEntry{}
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

func TestDomainMembershipTripleEntry_nok(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)

	val := DomainMembershipTripleEntry{}
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
