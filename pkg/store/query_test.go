package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
	"github.com/veraison/eat"
)

func TestDigestQuery(t *testing.T) {
	bytes := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"digests.yaml": digestsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewDigestQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 7)

	query = NewDigestQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewDigestQuery().Owner("measurement", 1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewDigestQuery().OwnerType("measurement")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewDigestQuery().Value(bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDigestQuery().Digest(1, bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDigestQuery().Owner("measurement", 1).AlgID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, bytes, result[0].Value)

	digest := result[0]
	query = NewDigestQuery().OwnerFromModel(digest)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewDigestQuery().DigestFromModel(digest)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, *digest, *result[0])
}

func TestIntegrityRegisterQuery(t *testing.T) {
	bytes := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"digests.yaml":              digestsFixture,
		"integerity_registers.yaml": integrityRegistersFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewIntegrityRegisterQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewIntegrityRegisterQuery().ID(1, 2).DigestID(4)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewIntegrityRegisterQuery().IndexText("reg1")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].MeasurementID)

	query = NewIntegrityRegisterQuery().IndexUint(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(2), result[0].MeasurementID)

	query = NewIntegrityRegisterQuery().MeasurementID(2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewIntegrityRegisterQuery().DigestValue(1, bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	digest := model.Digest{AlgID: 1, Value: bytes}
	query = NewIntegrityRegisterQuery().DigestFromModel(&digest)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	idxUint := uint64(1)
	idxStr := "reg1"
	query = NewIntegrityRegisterQuery().
		UpdateFromModel(&model.IntegrityRegister{
			IndexText: &idxStr,
			Digests: []*model.Digest{
				{
					AlgID: 1,
					Value: []byte{
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
					},
				},
			},
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewIntegrityRegisterQuery().
		UpdateFromModel(&model.IntegrityRegister{
			IndexUint: &idxUint,
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestFlagQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"flags.yaml": flagsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewFlagQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewFlagQuery().MeasurementID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewFlagQuery().CodePoint(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewFlagQuery().Value(false)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewFlagQuery().ID(1, 2, 3)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewFlagQuery().CodePoint(2).MeasurementID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewFlagQuery().Flag(1, true)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewFlagQuery().CodePoint(3)
	result, err = query.Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
	assert.Len(t, result, 0)
}

func TestMeasurementValueQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"measurement_values.yaml": measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewMeasurementValueQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 7)

	query = NewMeasurementValueQuery().MeasurementID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewMeasurementValueQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "foo", *result[1].ValueText)

	query = NewMeasurementValueQuery().ValueType("int", "string")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewMeasurementValueQuery().Value("string", "foo")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().Value("int", "bar")
	result, err = query.Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
	assert.Len(t, result, 0)

	query = NewMeasurementValueQuery().Value("int", 1.0)
	result, err = query.Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
	assert.Len(t, result, 0)
	assert.Equal(t, "@ERROR: unexpected value: 1 (float64)@", query.valueTypes[0])

	err = NewMeasurementValueQuery().AddValue("int", 1.0)
	assert.ErrorContains(t, err, "unexpected value: 1 (float64)")

	valueBytes := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	}
	valueText := "foo"
	valueInt := int64(42)

	query = NewMeasurementValueQuery().
		CodePoint(4).
		ValueBytes(valueBytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().
		CodePoint(1, 11).
		ValueText(valueText)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().
		CodePoint(1).
		ValueInt(valueInt)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().
		UpdateFromModel(&model.MeasurementValueEntry{
			CodePoint:  4,
			ValueType:  "bytes",
			ValueBytes: &valueBytes,
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().
		UpdateFromModel(&model.MeasurementValueEntry{
			CodePoint: 1,
			ValueType: "exact-value",
			ValueInt:  &valueInt,
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementValueQuery().
		UpdateFromModel(&model.MeasurementValueEntry{
			CodePoint: 11,
			ValueType: "string",
			ValueText: &valueText,
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCryptoKeyQuery(t *testing.T) {
	ctx := context.Background()
	bytes := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"cryptokeys.yaml": cryptoKeysFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewCryptoKeyQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewCryptoKeyQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewCryptoKeyQuery().KeyType("bytes")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, bytes, result[0].KeyBytes)

	query = NewCryptoKeyQuery().KeyBytes(bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewCryptoKeyQuery().Key("bytes", bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewCryptoKeyQuery().KeyFromModel(result[0])
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewCryptoKeyQuery().OwnerType("key_triple_auth")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewCryptoKeyQuery().OwnerID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewCryptoKeyQuery().Owner("key_triple_auth", 1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewCryptoKeyQuery().OwnerFromModel(result[0])
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestMeasurementQuery(t *testing.T) {
	ctx := context.Background()
	bytes := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"cryptokeys.yaml":           cryptoKeysFixture,
		"digests.yaml":              digestsFixture,
		"flags.yaml":                flagsFixture,
		"integerity_registers.yaml": integrityRegistersFixture,
		"measurement_values.yaml":   measurementValuesFixture,
		"measurements.yaml":         measurementsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewMeasurementQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 6)

	query = NewMeasurementQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewMeasurementQuery().Owner("value_triple", 1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewMeasurementQuery().MkeyType("uint")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, []byte{0x01}, *result[0].KeyBytes)

	query = NewMeasurementQuery().DigestValue(bytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, *result[0].KeyBytes)

	query = NewMeasurementQuery().MVal(func(mvq *MeasurementValueQuery) {
		mvq.CodePoint(1)
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementQuery().IntegrityRegister(func(irq *IntegrityRegisterQuery) {
		irq.IndexText("reg1")
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewMeasurementQuery().Flag(func(fq *FlagQuery) {
		fq.CodePoint(1).Value(true)
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestLocatorQuery(t *testing.T) {
	ctx := context.Background()
	bytes := comid.MustHexDecode(t, "4041424344454647404142434445464740414243444546474041424344454647")
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"locators.yaml": locatorsFixture,
		"digests.yaml":  digestsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewLocatorQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewLocatorQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewLocatorQuery().
		ManifestID(1).
		Href("foo").
		Digests(func(dq *DigestQuery) {
			dq.AlgID(1).Value(bytes)
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestEntitiesQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"entities.yaml": entitiesFixture,
		"roles.yaml":    rolesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewEntityQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewEntityQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEntityQuery().OwnerID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewEntityQuery().Owner("manifest", 1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEntityQuery().OwnerType("module_tag")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEntityQuery().
		NameType("string").
		NameValue("foo").
		URI("http://example.com")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEntityQuery().
		Name("string", "bar").
		Role("manifestSigner")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestManifestQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml": manifestsFixture,
		"entities.yaml":  entitiesFixture,
		"locators.yaml":  locatorsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewManifestQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewManifestQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		ValidOn(time.Now())
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewManifestQuery().
		ValidOn(time.Date(2025, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewManifestQuery().
		ValidBefore(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		ValidAfter(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewManifestQuery().
		ValidAfter(time.Date(3000, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		ValidBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		AddedBefore(time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		AddedAfter(time.Date(2024, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewManifestQuery().
		AddedAfter(time.Date(2027, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
	assert.Len(t, result, 0)

	query = NewManifestQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewManifestQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	bytes := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")

	query = NewManifestQuery().
		Label("baz").
		Profile(model.URIProfile, "http://example.com").
		ManifestID(model.StringTagID, "foo").
		Digest(bytes).
		Entity(func(eq *EntityQuery) {
			eq.Name("string", "foo")
		}).
		DependentRIMs(func(lq *LocatorQuery) {
			lq.Href("foo")
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewManifestQuery().
		ProfileType(model.OIDProfile).
		ProfileValue("1.2.3.4").
		ManifestIDType(model.UUIDTagID).
		ManifestIDValue("03c5e92b-2950-440b-93f0-21ac612a40bd")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	oidProfile, err := eat.NewProfile("1.2.3.4")
	require.NoError(t, err)
	uriProfile, err := eat.NewProfile("http://acme.com")
	require.NoError(t, err)

	query = NewManifestQuery().ProfileFromEAT(oidProfile, uriProfile)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestLinkedTagQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"linked_tags.yaml": linkedTagsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewLinkedTagQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewLinkedTagQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewLinkedTagQuery().
		LinkedTagIDType(model.StringTagID).
		LinkedTagIDValue("zot").
		TagRelation(model.SupplementsRelation).
		ModuleID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewLinkedTagQuery().LinkedTagID(model.StringTagID, "zap")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestModuleTagQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml":   manifestsFixture,
		"module_tags.yaml": moduleTagsFixture,
		"entities.yaml":    entitiesFixture,
		"linked_tags.yaml": linkedTagsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewModuleTagQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		ModuleTagIDType(model.StringTagID).
		ModuleTagIDValue("foo").
		ManifestIDType(model.StringTagID).
		ManifestIDValue("foo").
		ModuleTagVersion(7).
		Language("en_GB").
		ManifestDbID(1).
		Label("baz").
		Profile(model.URIProfile, "http://example.com").
		ManifestID(model.StringTagID, "foo").
		LinkedTag(func(ltq *LinkedTagQuery) {
			ltq.LinkedTagIDValue("zot")
		}).
		Entity(func(eq *EntityQuery) {
			eq.Name("string", "qux")
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewModuleTagQuery().
		ProfileType(model.URIProfile).
		ProfileValue("http://example.com").
		ModuleTagID(model.StringTagID, "foo").
		ValidOn(time.Now())
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewModuleTagQuery().
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		ValidAfter(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		ValidBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewModuleTagQuery().
		AddedBefore(time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewModuleTagQuery().
		AddedAfter(time.Date(2024, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	oidProfile, err := eat.NewProfile("1.2.3.4")
	require.NoError(t, err)

	query = NewModuleTagQuery().ProfileFromEAT(oidProfile)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestEnvironmentQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewEnvironmentQuery(false)
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 5)

	bytes0 := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	bytes1 := comid.MustHexDecode(t, "1011121314151617101112131415161710111213141516171011121314151617")
	bytes2 := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	query = NewEnvironmentQuery(false).ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEnvironmentQuery(false).
		ClassIDType("bytes").
		ClassIDBytes(bytes0).
		Vendor("foo").
		Model("bar").
		Layer(1).
		Index(0).
		InstanceType("bytes").
		InstanceBytes(bytes1).
		GroupType("bytes").
		GroupBytes(bytes2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEnvironmentQuery(false).
		ClassID("bytes", bytes0).
		Instance("bytes", bytes1).
		Group("bytes", bytes2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEnvironmentQuery(false).
		ClassIDBytes(comid.MustHexDecode(t, "00010203040506078001020304050607"))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEnvironmentQuery(true).
		ClassIDBytes(comid.MustHexDecode(t, "00010203040506070801020304050607"))
	result, err = query.Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
	assert.Len(t, result, 0)

	query = NewEnvironmentQuery(true).
		ClassIDBytes(comid.MustHexDecode(t, "00010203040506078001020304050607")).
		InstanceBytes(comid.MustHexDecode(t, "0110111213141516171011121314151617"))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEnvironmentQuery(false).
		Class(
			func(cs *ClassSubquery) {
				cs.ClassID("bytes", bytes0).
					Vendor("foo").
					Model("bar").
					Layer(1).
					Index(0)
			},
			func(cs *ClassSubquery) {
				cs.ClassIDType("oid").
					ClassIDBytes(comid.MustHexDecode(t, "0001020304"))
			},
		).
		GroupType("bytes").
		GroupBytes(bytes2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestEnvironmentQuery_exact(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	classIDBytes := comid.MustHexDecode(t, "1011121314151617101112131415161710111213141516171011121314151617")

	query := NewEnvironmentQuery(true).
		Class(func(cs *ClassSubquery) {
			cs.ClassIDBytes(classIDBytes)
		})
	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	classIDBytes = comid.MustHexDecode(t, "0001020304")
	instanceIDBytes := comid.MustHexDecode(t, "10111213141516178011121314151617")
	groupIDBytes := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	query = NewEnvironmentQuery(true).
		ClassID("oid", classIDBytes).
		Vendor("baz").
		Vendor("qux").
		Layer(2).
		Index(1).
		Instance("uuid", instanceIDBytes).
		Group("bytes", groupIDBytes)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestKeyTripleQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml":    manifestsFixture,
		"module_tags.yaml":  moduleTagsFixture,
		"triples.yaml":      triplesFixture,
		"environments.yaml": environmentsFixture,
		"cryptokeys.yaml":   cryptoKeysFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewKeyTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewKeyTripleQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	bytes0 := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	bytes2 := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	query = NewKeyTripleQuery().
		TripleDbID(1).
		ManifestDbID(1).
		ModuleTagDbID(1).
		EnvironmentID(3).
		IsActive(true).
		TripleType(model.AttestKeyTriple).
		ManifestIDType(model.StringTagID).
		ManifestIDValue("foo").
		ModuleTagIDType(model.StringTagID).
		ModuleTagIDValue("foo").
		ModuleTagVersion(7).
		Language("en_GB").
		ProfileType(model.URIProfile).
		ProfileValue("http://example.com").
		ClassType("oid").
		ClassBytes(comid.MustHexDecode(t, "0001020304")).
		Vendor("baz").
		Model("qux").
		Layer(2).
		Index(1).
		InstanceType("uuid").
		InstanceBytes(comid.MustHexDecode(t, "10111213141516178011121314151617")).
		GroupType("bytes").
		GroupBytes(bytes2).
		CryptoKey(func(e *CryptoKeyQuery) {
			e.KeyType("pkix-base64-cert")
		}).
		AuthorizedBy(func(e *CryptoKeyQuery) {
			e.KeyBytes(bytes0)
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewKeyTripleQuery().
		ManifestID(model.StringTagID, "foo").
		ModuleTagID(model.StringTagID, "foo").
		Profile(model.URIProfile, "http://example.com").
		Environment(func(e *EnvironmentQuery) {
			e.Vendor("baz")
		}).
		ValidOn(time.Now())
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewKeyTripleQuery().
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewKeyTripleQuery().
		ValidAfter(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewKeyTripleQuery().
		ValidBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewKeyTripleQuery().
		AddedBefore(time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewKeyTripleQuery().
		AddedAfter(time.Date(2024, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewKeyTripleQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestValueTripleQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml":          manifestsFixture,
		"module_tags.yaml":        moduleTagsFixture,
		"triples.yaml":            triplesFixture,
		"environments.yaml":       environmentsFixture,
		"measurements.yaml":       measurementsFixture,
		"measurement_values.yaml": measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewValueTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewValueTripleQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	bytes0 := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	bytes1 := comid.MustHexDecode(t, "1011121314151617101112131415161710111213141516171011121314151617")
	bytes2 := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	query = NewValueTripleQuery().
		TripleDbID(1).
		ManifestDbID(1).
		ModuleTagDbID(1).
		EnvironmentID(1).
		IsActive(true).
		TripleType(model.ReferenceValueTriple).
		ManifestIDType(model.StringTagID).
		ManifestIDValue("foo").
		ModuleTagIDType(model.StringTagID).
		ModuleTagIDValue("foo").
		ProfileType(model.URIProfile).
		ProfileValue("http://example.com").
		ClassType("bytes").
		ClassBytes(bytes0).
		Vendor("foo").
		Model("bar").
		Layer(1).
		Index(0).
		InstanceType("bytes").
		InstanceBytes(bytes1).
		GroupType("bytes").
		GroupBytes(bytes2).
		Measurement(func(e *MeasurementQuery) {
			e.MkeyBytes(comid.MustHexDecode(t, "01020304"))
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewValueTripleQuery().
		ValidOn(time.Now())
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewValueTripleQuery().
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewValueTripleQuery().
		ValidAfter(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewValueTripleQuery().
		ValidBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewValueTripleQuery().
		AddedBefore(time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewValueTripleQuery().
		AddedAfter(time.Date(2024, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewValueTripleQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewValueTripleQuery().
		Measurement(func(e *MeasurementQuery) {
			e.MkeyBytes(comid.MustHexDecode(t, "01020304"))
		}).
		Measurement(func(e *MeasurementQuery) {
			e.MVal(func(mvq *MeasurementValueQuery) {
				mvq.CodePoint(8).ValueText("12345")
			})
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	oidProfile, err := eat.NewProfile("1.2.3.4")
	require.NoError(t, err)

	query = NewValueTripleQuery().ProfileFromEAT(oidProfile)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestTokenQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"tokens.yaml": tokensFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewTokenQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = query.ID(1, 2).Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewTokenQuery().
		IsSigned(true).
		ManifestID("cca-ref-plat").
		Authority(func(ckq *CryptoKeyQuery) {
			ckq.KeyType("cose-key")
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	_, err = NewTokenQuery().Data([]byte{0xde, 0xad, 0xbe, 0xef}).Run(ctx, db)
	assert.ErrorIs(t, err, ErrNoMatch)
}
