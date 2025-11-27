package model

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

var testDigest = Digest{
	AlgID: 1,
	Value: []byte{
		0xe4, 0x5b, 0x72, 0xf5, 0xc0, 0xc0, 0xb5, 0x72,
		0xdb, 0x4d, 0x8d, 0x3a, 0xb7, 0xe9, 0x7f, 0x36,
		0x8f, 0xf7, 0x4e, 0x62, 0x34, 0x7a, 0x82, 0x4d,
		0xec, 0xb6, 0x7a, 0x84, 0xe5, 0x22, 0x4d, 0x75,
	},
}

func TestIntegrityRegister_round_trip(t *testing.T) {
	originDigest := swid.HashEntry{
		HashAlgID: swid.Sha256,
		HashValue: comid.MustHexDecode(
			t,
			"e45b72f5c0c0b572db4d8d3ab7e97f368ff74e62347a824decb67a84e5224d75",
		),
	}
	idxOne := uint(1)
	idxOne64 := uint64(1)
	idxTwo := uint64(2)
	idxFoo := "foo"
	origin := comid.NewIntegrityRegisters()
	require.NoError(t, origin.AddDigest(idxFoo, originDigest))
	require.NoError(t, origin.AddDigest(idxOne, originDigest))
	require.NoError(t, origin.AddDigest(idxTwo, originDigest))

	regs, err := IntegerityRegistersFromCoRIM(origin)
	sort.SliceStable(regs, func(i, j int) bool {
		return regs[i].StringIndex() < regs[j].StringIndex()
	})
	assert.NoError(t, err)
	assert.Equal(t, []*IntegrityRegister{
		{IndexUint: &idxOne64, Digests: []*Digest{&testDigest}},
		{IndexUint: &idxTwo, Digests: []*Digest{&testDigest}},
		{IndexText: &idxFoo, Digests: []*Digest{&testDigest}},
	}, regs)

	other, err := IntegerityRegistersToCoRIM(regs)
	assert.NoError(t, err)
	assert.Equal(t, origin.IndexMap[idxFoo], other.IndexMap[idxFoo])
	// note: different index types because the store make no distinction
	// between different unsigned integer types -- they are all stored as
	// uint64.
	assert.Equal(t, origin.IndexMap[idxOne], other.IndexMap[idxOne64])
	assert.Equal(t, origin.IndexMap[idxTwo], other.IndexMap[idxTwo])
}

func TestIntegrityRegistersToCoRIM_bad(t *testing.T) {
	_, err := IntegerityRegistersToCoRIM([]*IntegrityRegister{{Digests: []*Digest{&testDigest}}})
	assert.ErrorContains(t, err, "neither index set")

	idx := uint64(1)
	_, err = IntegerityRegistersToCoRIM([]*IntegrityRegister{{IndexUint: &idx}})
	assert.ErrorContains(t, err, "no digests")
}

func TestIntegrityRegister_Select(t *testing.T) {
	var reg IntegrityRegister
	db := test.NewTestDB(t)

	err := reg.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	reg.ID = 1
	err = reg.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestIntegrityRegister_Delete(t *testing.T) {
	var reg IntegrityRegister
	db := test.NewTestDB(t)

	err := reg.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	reg.ID = 1
	err = reg.Delete(context.Background(), db)
	assert.NoError(t, err)
}
