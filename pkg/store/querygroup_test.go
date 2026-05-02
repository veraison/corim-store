package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
)

func TestQueryGroup(t *testing.T) {
	qg := NewQueryGroup[*model.Environment, *EnvironmentQuery]()
	assert.True(t, qg.IsEmpty())

	bytes0 := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	bytes2 := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	qg.Add(
		NewEnvironmentQuery(false).
			ClassIDBytes(bytes0).
			Vendor("foo"),
		NewEnvironmentQuery(false).
			InstanceBytes(comid.MustHexDecode(t, "0110111213141516171011121314151617")),
		NewEnvironmentQuery(true).GroupBytes(bytes2),
	)
	assert.False(t, qg.IsEmpty())

	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	envs, err := qg.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, envs, 2)

	if db.Dialect().Name().String() != "sqlite" {
		t.SkipNow()
	}

	var ret []*model.Environment
	sel := db.NewSelect().Model(&ret)
	qg.UpdateSelectQuery(sel, db.Dialect())

	expected := `WHERE ((("class_bytes" = X'0001020304050607000102030405060700010203040506070001020304050607') AND ("vendor" = 'foo')) OR (("instance_bytes" = X'0110111213141516171011121314151617')) OR (("class_bytes" IS NULL) AND ("vendor" IS NULL) AND ("model" IS NULL) AND ("layer" IS NULL) AND ("index" IS NULL) AND ("group_bytes" = X'2021222324252627202122232425262720212223242526272021222324252627') AND ("instance_bytes" IS NULL)))`
	assert.Contains(t, sel.String(), expected)
}
