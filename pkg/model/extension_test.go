package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

func TestExtensionValue_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	extStruct := struct {
		Foo int64              `cbor:"0,keyasint,omitempty" json:"foo,omitempty"`
		Bar string             `cbor:"1,keyasint,omitempty" json:"bar,omitempty"`
		Qux struct{ Zap bool } `cbor:"2,keyasint,omitempty" json:"qux,omitempty"`
	}{
		Foo: 7,
		Bar: "baz",
		Qux: struct{ Zap bool }{true},
	}

	var original comid.Extensions
	original.Register(&extStruct)

	extVals, err := CoMIDExtensionsFromCoRIM(original)
	assert.NoError(t, err)
	assert.Len(t, extVals, 3)

	for _, ev := range extVals {
		ev.OwnerID = 1
		ev.OwnerType = "measurements"
		err = ev.Insert(ctx, db)
		assert.NoError(t, err)
	}

	var resVals []*ExtensionValue

	err = db.NewSelect().Model(&resVals).Scan(ctx)
	assert.NoError(t, err)
	assert.Len(t, resVals, 3)

	returnedExts, err := CoMIDExtensionsToCoRIM(resVals)
	assert.NoError(t, err)
	assert.Equal(t, 7, returnedExts.MustGetInt("foo"))
	assert.Equal(t, "baz", returnedExts.MustGetString("1"))

	val, err := returnedExts.Get("Qux")
	assert.NoError(t, err)
	assert.Equal(t, map[interface{}]interface{}{"Zap": true}, val)
}
