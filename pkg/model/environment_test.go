package model

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
)

var testLanguage = "en-GB"
var testVendor = "test vendor"
var testModel = "test model"
var testLayer = uint64(1)
var testIndex = uint64(0)

type IntInstance struct {
	V int64 `cbor:"1,keyasint"`
}

func NewIntInstance(v int64) *IntInstance {
	return &IntInstance{v}
}

func (o IntInstance) Type() string {
	return "int64"
}

func (o IntInstance) Valid() error {
	return nil
}

func (o IntInstance) String() string {
	return fmt.Sprintf("%d", o)
}

func (o IntInstance) Bytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(o.V))
	return buf
}

func TestEnvironment_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	factory := func(v any) (*comid.Instance, error) {
		if v == nil {
			return &comid.Instance{Value: NewIntInstance(0)}, nil
		}

		switch t := v.(type) {
		case int:
			return &comid.Instance{Value: NewIntInstance(int64(t))}, nil
		case int64:
			return &comid.Instance{Value: NewIntInstance(t)}, nil
		default:
			return nil, fmt.Errorf("invalid IntInstance: %+v", t)
		}
	}

	err := comid.RegisterInstanceType(777, factory)
	require.NoError(t, err)

	testExtInstance, err := factory(7)
	require.NoError(t, err)

	test_cases := []struct {
		title string
		env   comid.Environment
	}{
		{
			title: "full",
			env: comid.Environment{
				Class: &comid.Class{
					ClassID: comid.MustNewOIDClassID(comid.TestOID),
					Vendor:  &testVendor,
					Model:   &testModel,
					Layer:   &testLayer,
					Index:   &testIndex,
				},
				Instance: comid.MustNewUEIDInstance(comid.TestUEID),
				Group:    comid.MustNewUUIDGroup(comid.TestUUID),
			},
		},
		{
			title: "minimal",
			env: comid.Environment{
				Instance: comid.MustNewUEIDInstance(comid.TestUEID),
			},
		},
		{
			title: "extension",
			env: comid.Environment{
				Instance: testExtInstance,
			},
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			env, err := NewEnvironmentFromCoRIM(&tc.env)
			assert.NoError(t, err)

			err = env.Insert(ctx, db)
			require.NoError(t, err)

			selectedEnv, err := SelectEnvironment(ctx, db, env.ID)
			require.NoError(t, err)

			selectedCorimEnv, err := selectedEnv.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.env, selectedCorimEnv)
		})
	}
}

func TestEnvironment_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeef")
	test_cases := []struct {
		title string
		env   Environment
		err   string
	}{
		{
			title: "ok",
			env: Environment{
				ClassType:  &testType,
				ClassBytes: &testBytes,
			},
			err: "",
		},
		{
			title: "ok empty",
			env:   Environment{},
			err:   "",
		},
		{
			title: "bad class",
			env: Environment{
				ClassType: &testType,
			},
			err: "ClassType and ClassBytes must be set together",
		},
		{
			title: "bad instance",
			env: Environment{
				InstanceBytes: &testBytes,
			},
			err: "InstanceType and InstanceBytes must be set together",
		},
		{
			title: "bad group",
			env: Environment{
				GroupBytes: &testBytes,
			},
			err: "GroupType and GroupBytes must be set together",
		},
	}

	for _, tc := range test_cases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.env.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestEnvironment_uniqueness(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testVendor := "acme"
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeef")

	env := Environment{
		Vendor:        &testVendor,
		InstanceType:  &testType,
		InstanceBytes: &testBytes,
	}

	assert.NoError(t, env.Insert(ctx, db))
	origID := env.ID
	env.ID = 0
	assert.NoError(t, env.Insert(ctx, db))
	assert.Equal(t, origID, env.ID)

	var allEnvs []*Environment
	assert.NoError(t, db.NewSelect().Model(&allEnvs).Scan(ctx))
	assert.Len(t, allEnvs, 1)

	envOther := Environment{
		Vendor:     &testVendor,
		GroupType:  &testType,
		GroupBytes: &testBytes,
	}
	assert.NoError(t, envOther.Insert(ctx, db))
	assert.NotEqual(t, env.ID, envOther.ID)

	assert.NoError(t, db.NewSelect().Model(&allEnvs).Scan(ctx))
	assert.Len(t, allEnvs, 2)
}
