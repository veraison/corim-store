package model

import (
	"context"
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
