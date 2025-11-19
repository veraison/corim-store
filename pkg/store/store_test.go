package store

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

func TestStore_roundtrip(t *testing.T) {
	db := test.NewTestDB(t)
	store, err := OpenWithDB(context.Background(), db)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, store.Close()) }()

	for _, path := range []string{
		"../../sample/corim/unsigned-cca-ref-plat.cbor",
		"../../sample/corim/unsigned-cca-ref-realm.cbor",
		"../../sample/corim/unsigned-cca-ta.cbor",
	} {
		bytes, err := os.ReadFile(path)
		require.NoError(t, err)

		err = store.AddBytes(bytes, "cca", false)
		require.NoError(t, err)
	}

	classIDBytes := comid.MustHexDecode(t,
		"d9025858207f454c4602010100000000000000000003003e00010000005058000000000000")
	platRefLookup := model.Environment{ClassBytes: &classIDBytes}

	valueTriples, err := store.GetValueTriples(&platRefLookup, "cca", false)
	assert.NoError(t, err)
	assert.Len(t, valueTriples, 1)

	instanceIDBytes := comid.MustHexDecode(t,
		"0107060504030201000f0e0d0c0b0a090817161514131211101f1e1d1c1b1a1918")
	taLookup := model.Environment{InstanceBytes: &instanceIDBytes}

	keyTriples, err := store.GetKeyTriples(&taLookup, "cca", false)
	assert.NoError(t, err)
	assert.Len(t, keyTriples, 1)
}
