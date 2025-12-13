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
	testStr := "acme"

	extStruct := struct {
		Foo int64              `cbor:"0,keyasint,omitempty" json:"foo,omitempty"`
		Bar string             `cbor:"1,keyasint,omitempty" json:"bar,omitempty"`
		Qux struct{ Zap bool } `cbor:"2,keyasint,omitempty" json:"qux,omitempty"`
		Baz uint               `cbor:"3,keyasint,omitempty" json:"baz,omitempty"`
		Zot bool               `cbor:"4,keyasint,omitempty" json:"zot,omitempty"`
		Zof bool               `cbor:"5,keyasint,omitempty" json:"zof,omitempty"`
		Bap *string            `cbor:"6,keyasint,omitempty" json:"bap,omitempty"`
		Fop float64            `cbor:"7,keyasint,omitempty" json:"fop,omitempty"`
	}{
		Foo: 7,
		Bar: "baz",
		Qux: struct{ Zap bool }{true},
		Baz: 42,
		Zot: true,
		Zof: false,
		Bap: &testStr,
		Fop: 1.7,
	}

	var original comid.Extensions
	original.Register(&extStruct)
	original.Cached = map[string]any{
		"-1":  uint64(7),
		"fum": false,
	}

	extVals, err := CoMIDExtensionsFromCoRIM(original)
	assert.NoError(t, err)
	assert.Len(t, extVals, 10)

	for _, ev := range extVals {
		ev.OwnerID = 1
		ev.OwnerType = "measurements"
		err = ev.Insert(ctx, db)
		assert.NoError(t, err)
	}

	var resVals []*ExtensionValue

	err = db.NewSelect().Model(&resVals).Scan(ctx)
	assert.NoError(t, err)
	assert.Len(t, resVals, 10)

	returnedExts, err := CoMIDExtensionsToCoRIM(resVals)
	assert.NoError(t, err)
	assert.Equal(t, 7, returnedExts.MustGetInt("foo"))
	assert.Equal(t, "baz", returnedExts.MustGetString("1"))

	val, err := returnedExts.Get("Qux")
	assert.NoError(t, err)
	assert.Equal(t, map[interface{}]interface{}{"Zap": true}, val)

	assert.Equal(t, original.Cached, returnedExts.Cached)
}

func TestExtensionValue_Select(t *testing.T) {
	var ev ExtensionValue
	db := test.NewTestDB(t)

	err := ev.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ev.ID = 1
	err = ev.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestExtensionValue_Delete(t *testing.T) {
	var ev ExtensionValue
	db := test.NewTestDB(t)

	err := ev.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ev.ID = 1
	err = ev.Delete(context.Background(), db)
	assert.NoError(t, err)
}
