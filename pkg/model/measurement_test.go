package model

import (
	"context"
	"fmt"
	"math"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

func TestMeasurement_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	meas, err := comid.NewUUIDMeasurement(comid.TestUUID)
	require.NoError(t, err)
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	mac := comid.MACaddr(comid.MustHexDecode(t, "deadbeefdeadbeef"))
	ip := net.IP(comid.MustHexDecode(t, "deadbeefdeadbeef"))

	meas = meas.SetSVN(7).
		SetVersion("0.0.1", swid.VersionSchemeSemVer).
		SetFlagsTrue(comid.FlagIsConfigured).
		AddDigest(swid.Sha256, testBytes).
		SetRawValueBytes(testBytes, nil).
		SetMACaddr(mac).
		SetIPaddr(ip).
		SetSerialNumber("foo").
		SetUEID(comid.TestUEID).
		SetUUID(comid.TestUUID).
		SetName("bar")

	digests := comid.NewDigests().AddDigest(swid.Sha256, testBytes)
	regs := comid.NewIntegrityRegisters()
	assert.NoError(t, regs.AddDigests(comid.IRegisterIndex("baz"), *digests))
	meas.Val.IntegrityRegisters = regs

	extStruct := struct {
		Foo int64 `cbor:"0,keyasint,omitempty" json:"foo,omitempty"`
	}{Foo: 7}
	meas.Val.Register(&extStruct)

	meas.AuthorizedBy = comid.NewCryptoKeys().
		Add(comid.MustNewCryptoKey(comid.TestECPubKey, comid.PKIXBase64KeyType))

	var model Measurement
	err = model.FromCoRIM(meas)
	assert.NoError(t, err)

	err = model.Insert(ctx, db)
	assert.NoError(t, err)

	selectedModel, err := SelectMeasurement(ctx, db, model.ID)
	assert.NoError(t, err)

	selectedMeas, err := selectedModel.ToCoRIM()
	assert.NoError(t, err)

	assert.Equal(t, comid.MustNewTaggedSVN(7), selectedMeas.Val.SVN)
	assert.Equal(t, true, *selectedMeas.Val.Flags.IsConfigured)
	assert.Equal(t, testBytes, (*selectedMeas.Val.Digests)[0].HashValue)
	assert.Equal(t, &mac, selectedMeas.Val.MACAddr)
	assert.Equal(t, &ip, selectedMeas.Val.IPAddr)
	assert.Equal(t, "foo", *selectedMeas.Val.SerialNumber)
	assert.Equal(t, comid.TestUEID, *selectedMeas.Val.UEID)
	assert.Equal(t, comid.TestUUID, *selectedMeas.Val.UUID)
	assert.Equal(t, "bar", *selectedMeas.Val.Name)

	rawValBytes, err := selectedMeas.Val.RawValue.GetBytes()
	assert.NoError(t, err)
	assert.Equal(t, testBytes, rawValBytes)

	selectedDigests, ok := selectedMeas.Val.IntegrityRegisters.IndexMap[comid.IRegisterIndex("baz")]
	assert.True(t, ok)
	assert.Equal(t, *digests, selectedDigests)

	assert.NotNil(t, selectedMeas.AuthorizedBy)
	assert.Equal(t, comid.TestECPubKey, (*selectedMeas.AuthorizedBy)[0].Value.String())
}

func newBoolMkey(val any) (*comid.Mkey, error) {
	ret := boolMkeyType(false)
	if val == nil {
		return &comid.Mkey{Value: &ret}, nil
	}

	b, ok := val.(bool)
	if !ok {
		return nil, fmt.Errorf("invalid boolMkeyType value: %v", val)
	}
	ret = boolMkeyType(b)

	return &comid.Mkey{Value: &ret}, nil
}

type boolMkeyType bool

func (o boolMkeyType) String() string {
	return fmt.Sprint(bool(o))
}

func (o boolMkeyType) Valid() error {
	return nil
}

func (o boolMkeyType) Type() string {
	return "bool"
}

var (
	uuidType   = comid.UUIDType
	stringType = comid.StringType
	uintType   = comid.UintType
	oidType    = comid.OIDType
	boolType   = "bool"
)

func TestMeasurement_convert(t *testing.T) {
	boolMkey := boolMkeyType(true)
	svnInt := int64(7)
	largeSvnInternal := int64(math.MinInt64)
	testCases := []struct {
		title    string
		origin   comid.Measurement
		expected Measurement
		err      string
	}{
		{
			title: "ok uint mkey",
			origin: comid.Measurement{
				Key: comid.MustNewMkey(uint(7), comid.UintType),
			},
			expected: Measurement{KeyType: &uintType, KeyBytes: &[]byte{0x07}},
		},
		{
			title: "ok string mkey",
			origin: comid.Measurement{
				Key: comid.MustNewMkey("foo", comid.StringType),
			},
			expected: Measurement{KeyType: &stringType, KeyBytes: &[]byte{0x66, 0x6f, 0x6f}},
		},
		{
			title: "ok OID mkey",
			origin: comid.Measurement{
				Key: comid.MustNewMkey("0.1.2.3", comid.OIDType),
			},
			expected: Measurement{KeyType: &oidType, KeyBytes: &[]byte{0x01, 0x02, 0x03}},
		},
		{
			title: "ok extension mkey",
			origin: comid.Measurement{
				Key: &comid.Mkey{Value: &boolMkey},
			},
			expected: Measurement{KeyType: &boolType, KeyBytes: &[]byte{0xd9, 0x03, 0x0d, 0xf5}},
		},
		{
			title: "ok MinSVN",
			origin: comid.Measurement{
				Val: comid.Mval{
					SVN: comid.MustNewTaggedMinSVN(7),
				},
			},
			expected: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalSvn,
						ValueType: "min-value",
						ValueInt:  &svnInt,
					},
				},
			},
		},
		{
			title: "ok SVN  large",
			origin: comid.Measurement{
				Val: comid.Mval{
					SVN: comid.MustNewTaggedSVN(uint64(math.MaxInt64 + 1)),
				},
			},
			expected: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalSvn,
						ValueType: "exact-value",
						ValueInt:  &largeSvnInternal,
					},
				},
			},
		},
	}

	err := comid.RegisterMkeyType(781, newBoolMkey)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			mea, err := NewMeasurementFromCoRIM(&tc.origin)
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, *mea)

				other, err := mea.ToCoRIM()
				assert.NoError(t, err)
				assert.Equal(t, tc.origin, *other)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestMeasurement_Delete(t *testing.T) {
	db := test.NewTestDB(t)
	testCases := []struct {
		title string
		mea   Measurement
		err   string
	}{
		{
			title: "ok minimal",
			mea:   Measurement{ID: 1},
		},
		{
			title: "ok full",
			mea: Measurement{
				ID:                 1,
				AuthorizedBy:       []*CryptoKey{{ID: 1}},
				Digests:            []*Digest{{ID: 1}},
				Flags:              []*Flag{{ID: 1}},
				IntegrityRegisters: []*IntegrityRegister{{ID: 1}},
				ValueEntries:       []*MeasurementValueEntry{{ID: 1}},
				Extensions:         []*ExtensionValue{{ID: 1}},
			},
		},
		{
			title: "nok no ID",
			mea:   Measurement{},
			err:   "ID not set",
		},
		{
			title: "nok bad AuthorizedBy",
			mea: Measurement{
				ID:           1,
				AuthorizedBy: []*CryptoKey{{}},
			},
			err: "authorized-by key at index 0: ID not set",
		},
		{
			title: "nok bad Digests",
			mea: Measurement{
				ID:      1,
				Digests: []*Digest{{}},
			},
			err: "digest at index 0: ID not set",
		},
		{
			title: "nok bad Flags",
			mea: Measurement{
				ID:    1,
				Flags: []*Flag{{}},
			},
			err: "flag at index 0: ID not set",
		},
		{
			title: "nok bad IntegrityRegisters",
			mea: Measurement{
				ID:                 1,
				IntegrityRegisters: []*IntegrityRegister{{}},
			},
			err: "integrity register at index 0: ID not set",
		},
		{
			title: "nok bad ValueEntries",
			mea: Measurement{
				ID:           1,
				ValueEntries: []*MeasurementValueEntry{{}},
			},
			err: "value entry at index 0: ID not set",
		},
		{
			title: "nok bad Extensions",
			mea: Measurement{
				ID:         1,
				Extensions: []*ExtensionValue{{}},
			},
			err: "extension at index 0: ID not set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.mea.Delete(context.Background(), db)
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestMeasurement_ToCoRIM_nok(t *testing.T) {
	testText := "foo"
	testInt := int64(7)
	testCases := []struct {
		title string
		mea   Measurement
		err   string
	}{
		{
			title: "bad UUID mkey",
			mea: Measurement{
				KeyType:  &uuidType,
				KeyBytes: &[]byte{0x1},
			},
			err: "unexpected size for UUID",
		},
		{
			title: "bad string mkey",
			mea: Measurement{
				KeyType:  &stringType,
				KeyBytes: &[]byte{0xff},
			},
			err: "invalid utf-8 string",
		},
		{
			title: "missing version data",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalVersion,
					},
				},
			},
			err: "missing version data",
		},
		{
			title: "bad version scheme",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalVersion,
						ValueText: &testText,
						ValueType: "foo",
					},
				},
			},
			err: "invalid version scheme: foo",
		},
		{
			title: "bad SVN type",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalSvn,
						ValueInt:  &testInt,
						ValueType: "foo",
					},
				},
			},
			err: "unexpected SVN type: foo",
		},
		{
			title: "missing name data",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{CodePoint: MvalName},
				},
			},
			err: "missing Name data",
		},
		{
			title: "unexpected name type",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{
						CodePoint: MvalName,
						ValueText: &testText,
						ValueType: "bar",
					},
				},
			},
			err: "unexpected Name type",
		},
		{
			title: "unexpected code point",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{CodePoint: MvalDigests},
				},
			},
			err: "unexpected value entry for code point",
		},
		{
			title: "invalid code point",
			mea: Measurement{
				ValueEntries: []*MeasurementValueEntry{
					{CodePoint: -1},
				},
			},
			err: "unexpected code point: -1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err := tc.mea.ToCoRIM()
			assert.ErrorContains(t, err, tc.err)
		})
	}
}

func TestMeasurementValueEntry_Delete(t *testing.T) {
	var mve MeasurementValueEntry
	db := test.NewTestDB(t)

	err := mve.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	mve.ID = 1
	err = mve.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestMeasurementsFromCoRIM(t *testing.T) {
	ret, err := MeasurementsFromCoRIM(*comid.NewMeasurements())
	assert.Nil(t, ret)
	assert.NoError(t, err)

	_, err = MeasurementsFromCoRIM(*comid.NewMeasurements().
		Add(&comid.Measurement{Val: comid.Mval{SVN: &comid.SVN{}}}))
	assert.ErrorContains(t, err, "could not construct measurement at index 0: unexpected SVN type: <nil>")
}

func TestMeasurementsToCoRIM_nok(t *testing.T) {
	testKT := "foo"
	_, err := MeasurementsToCoRIM([]*Measurement{{KeyType: &testKT}})
	assert.ErrorContains(t, err, "could not convert measurement at index 0")
}

func TestParseVersionScheme(t *testing.T) {
	testCases := []struct {
		scheme string
		err    string
	}{
		{scheme: "multipartnumeric"},
		{scheme: "multipartnumeric+suffix"},
		{scheme: "alphanumeric"},
		{scheme: "decimal"},
		{scheme: "semver"},
		{scheme: "version-scheme(1)"},
		{
			scheme: "foo",
			err:    "invalid version scheme: foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.scheme, func(t *testing.T) {
			_, err := parseVersionScheme(tc.scheme)
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}
