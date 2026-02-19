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
	"github.com/veraison/corim/corim"
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
		"7f454c4602010100000000000000000003003e00010000005058000000000000")
	implID, err := comid.NewImplIDClassID(classIDBytes)
	require.NoError(t, err)
	platRefLookup := comid.Environment{Class: &comid.Class{ClassID: implID}}

	valueTriples, err := store.GetValueTriples(&platRefLookup, "cca", false)
	assert.NoError(t, err)
	assert.Len(t, valueTriples, 1)

	instanceIDBytes := comid.MustHexDecode(t,
		"0107060504030201000f0e0d0c0b0a090817161514131211101f1e1d1c1b1a1918")
	taLookup := comid.Environment{Instance: comid.MustNewUEIDInstance(instanceIDBytes)}

	keyTriples, err := store.GetKeyTriples(&taLookup, "cca", false)
	assert.NoError(t, err)
	assert.Len(t, keyTriples, 1)
}

func TestStore_Open(t *testing.T) {
	testCases := []struct {
		title string
		cfg   *Config
		err   string
	}{
		{
			title: "ok",
			cfg:   NewConfig("sqlite", "file::memory:?cache=shared"),
		},
		{
			title: "bad DSN",
			cfg:   NewConfig("mysql", "foo"),
			err:   "invalid DSN: missing the slash separating the database name",
		},
		{
			title: "nil config",
			err:   "nil config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err := Open(context.Background(), tc.cfg)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestStore_initialization(t *testing.T) {
	testStore, err := Open(context.Background(), NewConfig("sqlite", "file::memory:?cache=shared"))
	require.NoError(t, err)

	_, err = testStore.DB.DB.Exec("select * from measurements")
	assert.ErrorContains(t, err, "no such table")

	err = testStore.Init()
	require.NoError(t, err)

	err = testStore.Migrate()
	require.NoError(t, err)

	_, err = testStore.DB.DB.Exec("select * from measurements")
	assert.NoError(t, err)
}

func TestStore_AddBytes(t *testing.T) {
	testCases := []struct {
		title    string
		path     string
		bytes    []byte
		opts     []ConfigOption
		label    string
		activate bool
		err      string
	}{
		{
			title: "ok unsigned/no label/no activate",
			path:  "../../sample/corim/unsigned-cca-ta.cbor",
		},
		{
			title:    "ok unsigned/label/activate",
			path:     "../../sample/corim/unsigned-cca-ta.cbor",
			label:    "foo",
			activate: true,
		},
		{
			title: "ok signed",
			path:  "../../sample/corim/signed-cca-ta.cose",
			opts:  []ConfigOption{OptionInsecure},
		},
		{
			title: "nok signed without insecure",
			path:  "../../sample/corim/signed-cca-ta.cose",
			err:   "signed CoRIM validation not supported",
		},
		{
			title: "nok input too short",
			bytes: []byte{0x01, 0x02},
			err:   "input too short",
		},
		{
			title: "nok unrecognized input format",
			bytes: []byte{0x01, 0x02, 0x03, 0x04},
			err:   "unrecognized input format",
		},
		{
			title: "nok bad unsigned",
			bytes: []byte{0xd9, 0x01, 0xf5, 0x01},
			err:   "found Major Type 0",
		},
		{
			title: "nok bad signed",
			bytes: []byte{0xd2, 0x01, 0x02, 0x03},
			opts:  []ConfigOption{OptionInsecure},
			err:   "failed CBOR decoding for COSE-Sign1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			var err error
			var bytes []byte

			if tc.path != "" {
				bytes, err = os.ReadFile(tc.path)
			} else {
				bytes = tc.bytes
			}
			require.NoError(t, err)

			cfg := NewConfig("sqlite", "file::memory:", tc.opts...)
			store, err := Open(context.Background(), cfg)
			require.NoError(t, err)
			require.NoError(t, store.Init())
			require.NoError(t, store.Migrate())

			err = store.AddBytes(bytes, tc.label, tc.activate)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}

			require.NoError(t, store.Close())
		})
	}
}

func TestStore_Manifest_CRUD(t *testing.T) {
	bytes, err := os.ReadFile("../../sample/corim/unsigned-cca-ta.cbor")
	require.NoError(t, err)

	var unsigned corim.UnsignedCorim
	err = unsigned.FromCBOR(bytes)
	require.NoError(t, err)

	manifest, err := model.NewManifestFromCoRIM(&unsigned)
	require.NoError(t, err)
	manifest.Digest = []byte{0xde, 0xad, 0xbe, 0xef}
	manifest.Label = "label"

	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	err = store.AddManifest(manifest)
	assert.NoError(t, err)

	otherManifest, err := store.GetManifest(manifest.ManifestID, "label")
	assert.NoError(t, err)

	// note: we cannot simply compare ModuleTag's for equality be cause the
	// original is not going to have database-internal files (e.g. ID,
	// OwnerID, etc) set.

	modOrig := manifest.ModuleTags[0]
	modDB := otherManifest.ModuleTags[0]
	assert.Equal(t, modOrig.TagIDType, modDB.TagIDType)
	assert.Equal(t, modOrig.TagID, modDB.TagID)
	assert.Equal(t, modOrig.TagID, modDB.TagID)

	envOrig := modOrig.KeyTriples[0].Environment
	envDB := modDB.KeyTriples[0].Environment
	assert.Equal(t, envOrig.ClassType, envDB.ClassType)
	assert.Equal(t, envOrig.ClassBytes, envDB.ClassBytes)
	assert.Equal(t, envOrig.Vendor, envDB.Vendor)
	assert.Equal(t, envOrig.Model, envDB.Model)
	assert.Equal(t, envOrig.Layer, envDB.Layer)
	assert.Equal(t, envOrig.Index, envDB.Index)
	assert.Equal(t, envOrig.InstanceType, envDB.InstanceType)
	assert.Equal(t, envOrig.InstanceBytes, envDB.InstanceBytes)
	assert.Equal(t, envOrig.GroupType, envDB.GroupType)
	assert.Equal(t, envOrig.GroupBytes, envDB.GroupBytes)

	env, err := envOrig.ToCoRIM()
	require.NoError(t, err)

	_, err = store.GetActiveKeyTriples(env, "", false)
	assert.NoError(t, err)

	_, err = store.GetActiveKeyTriples(env, "label", false)
	assert.NoError(t, err)

	_, err = store.GetActiveValueTriples(env, "label", false)
	assert.NoError(t, err)

	err = store.AddManifest(otherManifest)
	assert.ErrorContains(t, err, "already in store (digests match)")

	otherManifest.Digest = []byte{0x01, 0x02, 0x03, 0x04}
	err = store.AddManifest(otherManifest)
	assert.ErrorContains(t, err, "already in store but digests differ")

	store.cfg.Force = true
	err = store.AddManifest(otherManifest)
	assert.NoError(t, err)

	store.cfg.RequireLabel = true
	_, err = store.GetManifest(manifest.ManifestID, "")
	assert.ErrorContains(t, err, "a label must be specified")

	err = store.DeleteManifest(manifest.ManifestID, "label")
	assert.NoError(t, err)

	err = store.DeleteManifest(manifest.ManifestID, "label")
	assert.ErrorContains(t, err, "manifest with ID \"cca-ta\" not found")
}

func TestStore_Find_bad(t *testing.T) {
	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	testLayer := uint64(1)
	lookupEnv := model.Environment{Layer: &testLayer}
	_, err = store.FindEnvironmentIDs(&lookupEnv, false)
	assert.ErrorContains(t, err, "no matching environments found")

	_, err = store.FindModuleTagIDsForLabel("")
	assert.ErrorContains(t, err, "no label specified")
}

func TestStore_StringAggregatorExpr(t *testing.T) {
	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	ret := store.StringAggregatorExpr("foo")
	assert.Equal(t, "GROUP_CONCAT(foo, ', ')", ret)

	testCases := []struct {
		title    string
		expected string
	}{
		{
			title:    "mysql",
			expected: "GROUP_CONCAT(foo SEPARATOR ', ')",
		},
		{
			title:    "pg",
			expected: "STRING_AGG(foo, ', ')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			ret := StringAggregatorExprForDialect(tc.title, "foo")
			assert.Equal(t, tc.expected, ret)
		})
	}

	assert.Panics(t, func() { StringAggregatorExprForDialect("foo", "bar") })
}

func TestStore_ConcatExpr(t *testing.T) {
	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	ret := store.ConcatExpr("foo", "bar")
	assert.Equal(t, "foo || bar", ret)

	testCases := []struct {
		title    string
		expected string
	}{
		{
			title:    "mysql",
			expected: "CONCAT(foo, bar)",
		},
		{
			title:    "pg",
			expected: "foo || bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			ret := ConcatExprForDialect(tc.title, "foo", "bar")
			assert.Equal(t, tc.expected, ret)
		})
	}

	assert.Panics(t, func() { ConcatExprForDialect("foo", "bar", "qux") })
}

func TestStore_HexExpr(t *testing.T) {
	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	ret := store.HexExpr("foo")
	assert.Equal(t, "hex(foo)", ret)

	testCases := []struct {
		title    string
		expected string
	}{
		{
			title:    "mysql",
			expected: "hex(foo)",
		},
		{
			title:    "pg",
			expected: "encode(foo, 'hex')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			ret := HexExprForDialect(tc.title, "foo")
			assert.Equal(t, tc.expected, ret)
		})
	}

	assert.Panics(t, func() { HexExprForDialect("foo", "bar") })
}

func TestStore_Digest(t *testing.T) {
	test_input := []byte{0xde, 0xad, 0xbe, 0xef}
	testCases := []struct {
		title    string
		expected []byte
	}{
		{
			title: "md5",
			expected: []byte{
				0x2f, 0x24, 0x92, 0x30, 0xa8, 0xe7, 0xc2, 0xbf,
				0x60, 0x05, 0xcc, 0xd2, 0x67, 0x92, 0x59, 0xec,
			},
		},
		{
			title: "sha256",
			expected: []byte{
				0x5f, 0x78, 0xc3, 0x32, 0x74, 0xe4, 0x3f, 0xa9,
				0xde, 0x56, 0x59, 0x26, 0x5c, 0x1d, 0x91, 0x7e,
				0x25, 0xc0, 0x37, 0x22, 0xdc, 0xb0, 0xb8, 0xd2,
				0x7d, 0xb8, 0xd5, 0xfe, 0xaa, 0x81, 0x39, 0x53,
			},
		},
		{
			title: "sha512",
			expected: []byte{
				0x12, 0x84, 0xb2, 0xd5, 0x21, 0x53, 0x51, 0x96,
				0xf2, 0x21, 0x75, 0xd5, 0xf5, 0x58, 0x10, 0x42,
				0x20, 0xa6, 0xad, 0x76, 0x80, 0xe7, 0x8b, 0x49,
				0xfa, 0x6f, 0x20, 0xe5, 0x7e, 0xa7, 0xb1, 0x85,
				0xd7, 0x1e, 0xc1, 0xed, 0xb1, 0x37, 0xe7, 0x0e,
				0xba, 0x52, 0x8d, 0xed, 0xb1, 0x41, 0xf5, 0xd2,
				0xf8, 0xbb, 0x53, 0x14, 0x9d, 0x26, 0x29, 0x32,
				0xb2, 0x7c, 0xf4, 0x1f, 0xed, 0x96, 0xaa, 0x7f,
			},
		},
	}

	store, err := OpenWithDB(context.Background(), test.NewTestDB(t))
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			store.cfg.HashAlg = tc.title
			ret := store.Digest(test_input)
			assert.Equal(t, tc.expected, ret)
		})
	}

	store.cfg.HashAlg = "foo"
	assert.Panics(t, func() { store.Digest(test_input) })
}
