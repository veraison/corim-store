package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/util"
	"github.com/veraison/corim/comid"
	"github.com/veraison/eat"
	"github.com/veraison/swid"
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

	query = NewDigestQuery().Owner("measurement", 1).IntAlgID(1)
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

	digest := model.Digest{AlgIDInt: 1, Value: bytes}
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
					AlgIDInt: 1,
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
	assert.Len(t, result, 8)

	query = NewMeasurementValueQuery().MeasurementID(1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewMeasurementValueQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "foo", *result[1].ValueText)

	query = NewMeasurementValueQuery().ValueType("int", "string")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

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
	assert.Len(t, result, 2)

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
	assert.Len(t, result, 2)

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
	assert.Len(t, result, 4)

	query = NewCryptoKeyQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewCryptoKeyQuery().KeyType("bytes")
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
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
	assert.Len(t, result, 4)

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
	assert.Len(t, result, 14)

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
	assert.Len(t, result, 9)
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
	assert.Len(t, result, 2)

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

func TestHrefQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"locators.yaml": locatorsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewHrefQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewHrefQuery().ID(1)
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
			dq.IntAlgID(1).Value(bytes)
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewLocatorQuery().
		Href("doesnotexist")
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "href: no match found")

	query = NewLocatorQuery().
		Digests(func(dq *DigestQuery) {
			dq.IntAlgID(99999)
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "digests: no match found")
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

	query = NewEntityQuery().
		Role("doesnotexist")
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "roles: no match found")
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

	query = NewManifestQuery().
		Entity(func(eq *EntityQuery) {
			eq.Name("string", "doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "entities: no match found")

	query = NewManifestQuery().
		DependentRIMs(func(lq *LocatorQuery) {
			lq.Href("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "dependent RIMs: href: no match found")
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
	assert.Len(t, result, 3)

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
	assert.Len(t, result, 3)

	query = NewModuleTagQuery().
		ValidAfter(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewModuleTagQuery().
		ValidBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		AddedBefore(time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		AddedAfter(time.Date(2024, time.February, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	query = NewModuleTagQuery().
		AddedBetween(
			time.Date(2024, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2026, time.February, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	oidProfile, err := eat.NewProfile("1.2.3.4")
	require.NoError(t, err)

	query = NewModuleTagQuery().ProfileFromEAT(oidProfile)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewModuleTagQuery().
		ModuleTagIDFromSWID(*swid.NewTagID("foo")).
		ManifestIDFromSWID(*swid.NewTagID("foo"))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewModuleTagQuery().
		Entity(func(eq *EntityQuery) {
			eq.NameValue("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "entities: no match found")

	query = NewModuleTagQuery().
		LinkedTag(func(ltq *LinkedTagQuery) {
			ltq.LinkedTagIDValue("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "linked tags: no match found")
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
	assert.Len(t, result, 9)

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

	query = NewEnvironmentQuery(false).UpdateFromModel(&model.Environment{
		ClassType:     util.Ptr("bytes"),
		ClassBytes:    &bytes0,
		Vendor:        util.Ptr("foo"),
		Model:         util.Ptr("bar"),
		Layer:         util.Ptr(uint64(1)),
		Index:         util.Ptr(uint64(0)),
		InstanceType:  util.Ptr("bytes"),
		InstanceBytes: &bytes1,
		GroupType:     util.Ptr("bytes"),
		GroupBytes:    &bytes2,
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEnvironmentQuery(false).UpdateFromModel(&model.Environment{
		ClassBytes:    &bytes0,
		InstanceBytes: &bytes1,
		GroupBytes:    &bytes2,
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEnvironmentQuery(false).UpdateFromModel(&model.Environment{
		ClassType:    util.Ptr("bytes"),
		InstanceType: util.Ptr("bytes"),
		GroupType:    util.Ptr("bytes"),
	})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

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
					ClassIDBytes(comid.MustHexDecode(t, "01020304"))
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

	classIDBytes = comid.MustHexDecode(t, "01020304")
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

func TestClassSubquery(t *testing.T) {
	query := &ClassSubquery{}
	assert.True(t, query.IsEmpty())

	query.UpdateFromCoRIM(&comid.Class{
		Vendor: util.Ptr("foo"),
		Model:  util.Ptr("bar"),
		Layer:  util.Ptr(uint64(1)),
		Index:  util.Ptr(uint64(2)),
	})
	assert.Equal(t, "foo", query.vendors[0])
	assert.Equal(t, "bar", query.models[0])
	assert.Equal(t, uint64(1), query.layers[0])
	assert.Equal(t, uint64(2), query.indexes[0])
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
		ClassBytes(comid.MustHexDecode(t, "01020304")).
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

	query = NewKeyTripleQuery().
		Environment(func(e *EnvironmentQuery) {
			e.Model("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "environment: no match found")

	query = NewKeyTripleQuery().
		CryptoKey(func(e *CryptoKeyQuery) {
			e.KeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "crypto keys: no match found")

	query = NewKeyTripleQuery().
		AuthorizedBy(func(e *CryptoKeyQuery) {
			e.KeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "auth by: no match found")

	query = NewKeyTripleQuery().
		Measurement(func(e *MeasurementQuery) {
			e.DigestTextAlgID("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "measurements: no match found")
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

func TestDomainEntryQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
		"triples.yaml":      triplesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewDomainEntryQuery()
	assert.True(t, query.IsEmpty())

	query = NewDomainEntryQuery().
		ID(1, 2).
		OwnerID(1).
		OwnerType("domain_dependency_triple").
		ExactEnvironment(false).
		EnvironmentID(8).
		Environment(func(e *EnvironmentQuery) {
			e.ClassIDType("bytes")
		})
	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainEntryQuery().
		ID(1, 2).
		OwnerID(1).
		OwnerType("domain_dependency_triple").
		ExactEnvironment(false).
		EnvironmentID(8).
		Environment(func(e *EnvironmentQuery) {
			e.ClassIDType("bytes")
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainEntryQuery().
		OwnerFromModel(&model.DomainEntry{
			OwnerID:   1,
			OwnerType: "domain_membership_triple",
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainEntryQuery().
		Environment(func(e *EnvironmentQuery) {
			e.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "environment: no match found")
}

func TestDomainDependencyTripleQuery(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
		"manifests.yaml":    manifestsFixture,
		"module_tags.yaml":  moduleTagsFixture,
		"triples.yaml":      triplesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewDomainDependencyTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = query.ID(1, 2).Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	bytes0 := comid.MustHexDecode(t, "3031323334353637303132333435363730313233343536373031323334353637")

	query = NewDomainDependencyTripleQuery().
		TripleDbID(1).
		ManifestDbID(1).
		ModuleTagDbID(1).
		DomainIDEnvironmentID(6).
		IsActive(true).
		ManifestIDType(model.StringTagID).
		ManifestIDValue("foo").
		ModuleTagIDType(model.StringTagID).
		ModuleTagIDValue("foo").
		ProfileType(model.URIProfile).
		ProfileValue("http://example.com").
		ExactDomainID(false).
		DomainID(func(e *EnvironmentQuery) {
			e.ClassIDBytes(bytes0)
		}).
		AddedBetween(
			time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		).
		ValidBetween(
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2028, time.January, 01, 00, 00, 00, 00, time.UTC),
		)

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainDependencyTripleQuery().
		ManifestID(model.StringTagID, "foo").
		ModuleTagID(model.StringTagID, "foo").
		ModuleTagVersion(7).
		Language("en_GB").
		Label("baz").
		Profile(model.URIProfile, "http://example.com").
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		ValidAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC))

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	profile, err := eat.NewProfile("http://example.com")
	assert.NoError(t, err)

	query = NewDomainDependencyTripleQuery().
		ProfileFromEAT(profile).
		ValidOn(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainDependencyTripleQuery().
		TrusteeExactEnvironment(false).
		TrusteeEnvironmentID(8).
		TrusteeEnvironment(func(e *EnvironmentQuery) {
			e.ClassIDType("bytes")
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainDependencyTripleQuery().
		EntryExactEnvironment(false).
		EntryEnvironmentID(8).
		EntryEnvironment(func(e *EnvironmentQuery) {
			e.ClassIDType("bytes")
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	query = NewDomainDependencyTripleQuery().
		DomainID(func(e *EnvironmentQuery) {
			e.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "domain ID: no match found")

	query = NewDomainDependencyTripleQuery().
		TrusteeEnvironmentID(9999)
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "entries: no match found")
}

func TestDomainMembershipTripleQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml": environmentsFixture,
		"manifests.yaml":    manifestsFixture,
		"module_tags.yaml":  moduleTagsFixture,
		"triples.yaml":      triplesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewDomainMembershipTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = query.ID(1, 2).Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	bytes0 := comid.MustHexDecode(t, "5051525354555657505152535455565750515253545556575051525354555657")

	query = NewDomainMembershipTripleQuery().
		TripleDbID(1).
		ManifestDbID(1).
		ModuleTagDbID(1).
		DomainIDEnvironmentID(8).
		IsActive(true).
		ManifestIDType(model.StringTagID).
		ManifestIDValue("foo").
		ModuleTagIDType(model.StringTagID).
		ModuleTagIDValue("foo").
		ProfileType(model.URIProfile).
		ProfileValue("http://example.com").
		DomainID(func(e *EnvironmentQuery) {
			e.ClassIDBytes(bytes0)
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewDomainMembershipTripleQuery().
		MemberExactEnvironment(false).
		MemberEnvironmentID(6).
		MemberEnvironment(func(e *EnvironmentQuery) {
			e.ClassIDType("bytes")
		})

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestStatefulEnvironmentQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"environments.yaml":         environmentsFixture,
		"statefu_environments.yaml": statefulEnvironmentsFixture,
		"measurements.yaml":         measurementsFixture,
		"measurement_values.yaml":   measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewStatefulEnvironmentQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	classBytes := []byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	}

	query = NewStatefulEnvironmentQuery().
		ID(1).
		TripleID(1).
		EnvironmentID(6).
		ExactEnvironment(false).
		ClassIDType(comid.BytesType).
		ClassIDBytes(classBytes)

	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	bytes0 := comid.MustHexDecode(t, "0001020304050607000102030405060700010203040506070001020304050607")
	bytes1 := comid.MustHexDecode(t, "1011121314151617101112131415161710111213141516171011121314151617")
	bytes2 := comid.MustHexDecode(t, "2021222324252627202122232425262720212223242526272021222324252627")

	query = NewStatefulEnvironmentQuery().
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

	query = NewStatefulEnvironmentQuery().
		ClassID("bytes", bytes0).
		Instance("bytes", bytes1).
		Group("bytes", bytes2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewStatefulEnvironmentQuery().
		Class(func(cs *ClassSubquery) {
			cs.ClassIDType("bytes")
		}).
		Measurement(func(m *MeasurementQuery) {
			m.MkeyType("uint")
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewStatefulEnvironmentQuery().
		Class(func(cs *ClassSubquery) {
			cs.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "environment: no match found")

	query = NewStatefulEnvironmentQuery().
		Measurement(func(m *MeasurementQuery) {
			m.MkeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "measurements: no match found")
}

func TestEndorsementQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"triples.yaml":            triplesFixture,
		"environments.yaml":       environmentsFixture,
		"measurements.yaml":       measurementsFixture,
		"measurement_values.yaml": measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewEndorsementQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 4)

	query = NewEndorsementQuery().ID(1, 2)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewEndorsementQuery().
		ID(3).
		EnvironmentID(1).
		OwnerID(1).
		OwnerType("conditional_endorsement_triple").
		Environment(func(eq *EnvironmentQuery) {
			eq.ClassIDType("bytes")
		}).
		Measurement(func(e *MeasurementQuery) {
			e.MkeyType("uuid")
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEndorsementQuery().
		Owner("conditional_endorsement_triple", 1)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewEndorsementQuery().
		Environment(func(eq *EnvironmentQuery) {
			eq.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "environment: no match found")

	query = NewEndorsementQuery().
		Measurement(func(e *MeasurementQuery) {
			e.MkeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "measurements: no match found")
}

func TestConditionalEndorsementTripleQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml":             manifestsFixture,
		"module_tags.yaml":           moduleTagsFixture,
		"triples.yaml":               triplesFixture,
		"environments.yaml":          environmentsFixture,
		"stateful_environments.yaml": statefulEnvironmentsFixture,
		"measurements.yaml":          measurementsFixture,
		"measurement_values.yaml":    measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewConditionalEndorsementTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementTripleQuery().
		TripleDbID(1, 2).
		ManifestDbID(2).
		ModuleTagDbID(3).
		IsActive(true).
		ManifestIDType("uuid").
		ManifestIDValue("03c5e92b-2950-440b-93f0-21ac612a40bd").
		ModuleTagIDType("string").
		ModuleTagIDValue("zot").
		ModuleTagVersion(1).
		Language("en_AU").
		Label("qux").
		ProfileType("oid").
		ProfileValue("1.2.3.4").
		Condition(func(seq *StatefulEnvironmentQuery) {
			seq.ClassIDType("bytes")
		}).
		Endorsement(func(eq *EndorsementQuery) {
			eq.EnvironmentID(1)
		}).
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		ValidAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewConditionalEndorsementTripleQuery().
		ManifestID("uuid", "03c5e92b-2950-440b-93f0-21ac612a40bd").
		ModuleTagID("string", "zot").
		Profile("oid", "1.2.3.4").
		AddedBetween(
			time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		).
		ValidBetween(
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2028, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	profile, err := eat.NewProfile("1.2.3.4")
	assert.NoError(t, err)

	query = NewConditionalEndorsementTripleQuery().
		ProfileFromEAT(profile).
		ValidOn(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementTripleQuery().
		Condition(func(seq *StatefulEnvironmentQuery) {
			seq.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "conditions: no match found")

	query = NewConditionalEndorsementTripleQuery().
		Endorsement(func(eq *EndorsementQuery) {
			eq.EnvironmentID(9999)
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "endorsements: no match found")
}

func TestConditionalEndorsementSeriesRecordQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"triples.yaml":      triplesFixture,
		"measurements.yaml": measurementsFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewConditionalEndorsementSeriesRecordQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementSeriesRecordQuery().
		ID(1, 2).
		TripleID(1, 2).
		Selection(func(mq *MeasurementQuery) {
			mq.MkeyType("uint")
		}).
		Addition(func(mq *MeasurementQuery) {
			mq.MkeyType("uint")
		})
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementSeriesRecordQuery().
		Selection(func(mq *MeasurementQuery) {
			mq.MkeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "selection: no match found")

	query = NewConditionalEndorsementSeriesRecordQuery().
		Addition(func(mq *MeasurementQuery) {
			mq.MkeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "addition: no match found")
}

func TestConditionalEndorsementSeriesTripleQuery(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"manifests.yaml":             manifestsFixture,
		"module_tags.yaml":           moduleTagsFixture,
		"triples.yaml":               triplesFixture,
		"environments.yaml":          environmentsFixture,
		"stateful_environments.yaml": statefulEnvironmentsFixture,
		"measurements.yaml":          measurementsFixture,
		"measurement_values.yaml":    measurementValuesFixture,
		"cryptokeys.yaml":            cryptoKeysFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	query := NewConditionalEndorsementSeriesTripleQuery()
	assert.True(t, query.IsEmpty())

	result, err := query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementSeriesTripleQuery().
		TripleDbID(1, 2).
		ManifestDbID(2).
		ModuleTagDbID(3).
		IsActive(true).
		ManifestIDType("uuid").
		ManifestIDValue("03c5e92b-2950-440b-93f0-21ac612a40bd").
		ModuleTagIDType("string").
		ModuleTagIDValue("zot").
		ModuleTagVersion(1).
		Language("en_AU").
		Label("qux").
		ProfileType("oid").
		ProfileValue("1.2.3.4").
		ExactEnvironment(false).
		Environment(func(eq *EnvironmentQuery) {
			eq.ClassIDType("bytes")
		}).
		Measurement(func(e *MeasurementQuery) {
			e.MkeyType("uint")
		}).
		AuthorizedBy(func(e *CryptoKeyQuery) {
			e.KeyType("bytes")
		}).
		Record(func(cesrq *ConditionalEndorsementSeriesRecordQuery) {
			cesrq.TripleID(1, 2)
		}).
		ValidBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		ValidAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedBefore(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC)).
		AddedAfter(time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	query = NewConditionalEndorsementSeriesTripleQuery().
		ManifestID("uuid", "03c5e92b-2950-440b-93f0-21ac612a40bd").
		ModuleTagID("string", "zot").
		Profile("oid", "1.2.3.4").
		AddedBetween(
			time.Date(2000, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
		).
		ValidBetween(
			time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC),
			time.Date(2028, time.January, 01, 00, 00, 00, 00, time.UTC),
		)
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	profile, err := eat.NewProfile("1.2.3.4")
	assert.NoError(t, err)

	query = NewConditionalEndorsementSeriesTripleQuery().
		ProfileFromEAT(profile).
		ValidOn(time.Date(2027, time.January, 01, 00, 00, 00, 00, time.UTC))
	result, err = query.Run(ctx, db)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	query = NewConditionalEndorsementSeriesTripleQuery().
		Environment(func(eq *EnvironmentQuery) {
			eq.ClassIDType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "condition environment: no match found")

	query = NewConditionalEndorsementSeriesTripleQuery().
		Measurement(func(e *MeasurementQuery) {
			e.MkeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "condition measurements: no match found")

	query = NewConditionalEndorsementSeriesTripleQuery().
		AuthorizedBy(func(e *CryptoKeyQuery) {
			e.KeyType("doesnotexist")
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "condition auth by: no match found")

	query = NewConditionalEndorsementSeriesTripleQuery().
		Record(func(cesrq *ConditionalEndorsementSeriesRecordQuery) {
			cesrq.TripleID(9999)
		})
	_, err = query.Run(ctx, db)
	assert.ErrorContains(t, err, "records: no match found")
}
