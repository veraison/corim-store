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

type IntClassID struct {
	V int64 `cbor:"1,keyasint"`
}

func NewIntClassID(v int64) *IntClassID {
	return &IntClassID{v}
}

func (o IntClassID) Type() string {
	return "int64"
}

func (o IntClassID) Valid() error {
	return nil
}

func (o IntClassID) String() string {
	return fmt.Sprintf("%d", o)
}

func (o IntClassID) Bytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(o.V))
	return buf
}

type IntGroup struct {
	V int64 `cbor:"1,keyasint"`
}

func NewIntGroup(v int64) *IntGroup {
	return &IntGroup{v}
}

func (o IntGroup) Type() string {
	return "int64"
}

func (o IntGroup) Valid() error {
	return nil
}

func (o IntGroup) String() string {
	return fmt.Sprintf("%d", o)
}

func (o IntGroup) Bytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(o.V))
	return buf
}

func TestEnvironment_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	instanceFactory := func(v any) (*comid.Instance, error) {
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

	err := comid.RegisterInstanceType(777, instanceFactory)
	require.NoError(t, err)

	testExtInstance, err := instanceFactory(7)
	require.NoError(t, err)

	classIDFactory := func(v any) (*comid.ClassID, error) {
		if v == nil {
			return &comid.ClassID{Value: NewIntClassID(0)}, nil
		}

		switch t := v.(type) {
		case int:
			return &comid.ClassID{Value: NewIntClassID(int64(t))}, nil
		case int64:
			return &comid.ClassID{Value: NewIntClassID(t)}, nil
		default:
			return nil, fmt.Errorf("invalid IntClass: %+v", t)
		}
	}

	err = comid.RegisterClassIDType(778, classIDFactory)
	require.NoError(t, err)

	testExtClassID, err := classIDFactory(7)
	require.NoError(t, err)

	groupFactory := func(v any) (*comid.Group, error) {
		if v == nil {
			return &comid.Group{Value: NewIntGroup(0)}, nil
		}

		switch t := v.(type) {
		case int:
			return &comid.Group{Value: NewIntGroup(int64(t))}, nil
		case int64:
			return &comid.Group{Value: NewIntGroup(t)}, nil
		default:
			return nil, fmt.Errorf("invalid IntGroup: %+v", t)
		}
	}

	err = comid.RegisterGroupType(779, groupFactory)
	require.NoError(t, err)

	testExtGroup, err := groupFactory(7)
	require.NoError(t, err)

	testCases := []struct {
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
			title: "class UUID",
			env: comid.Environment{
				Class: &comid.Class{
					ClassID: comid.MustNewUUIDClassID(comid.TestUUID),
				},
			},
		},
		{
			title: "class bytes",
			env: comid.Environment{
				Class: &comid.Class{
					ClassID: comid.MustNewBytesClassID(comid.TestBytes),
				},
			},
		},
		{
			title: "class extension",
			env: comid.Environment{
				Class: &comid.Class{
					ClassID: testExtClassID,
				},
			},
		},
		{
			title: "instance UEID",
			env: comid.Environment{
				Instance: comid.MustNewUEIDInstance(comid.TestUEID),
			},
		},
		{
			title: "instance bytes",
			env: comid.Environment{
				Instance: comid.MustNewBytesInstance(comid.TestBytes),
			},
		},
		{
			title: "instance extension",
			env: comid.Environment{
				Instance: testExtInstance,
			},
		},
		{
			title: "group bytes",
			env: comid.Environment{
				Group: comid.MustNewBytesGroup(comid.TestBytes),
			},
		},
		{
			title: "group extension",
			env: comid.Environment{
				Group: testExtGroup,
			},
		},
	}

	for _, tc := range testCases {
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

func TestEnvironment_ToCoRIM_bad(t *testing.T) {
	uuidType := comid.UUIDType
	bytes := comid.MustHexDecode(t, "deadbeef")
	testCases := []struct {
		title string
		env   Environment
		err   string
	}{
		{
			title: "UUID class ID",
			env: Environment{
				ClassType:  &uuidType,
				ClassBytes: &bytes,
			},
			err: "could not initialize UUID class ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err := tc.env.ToCoRIM()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestEnvironment_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeef")
	testCases := []struct {
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

	for _, tc := range testCases {
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

func TestEnvironment_Select(t *testing.T) {
	var env Environment
	db := test.NewTestDB(t)

	err := env.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	env.ID = 1
	err = env.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestEnvironment_DeleteIfOrphaned(t *testing.T) {
	var env Environment
	db := test.NewTestDB(t)

	err := env.DeleteIfOrphaned(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	env.ID = 1
	err = env.DeleteIfOrphaned(context.Background(), db)
	assert.NoError(t, err)
}

func TestEnvironment_IsEmpty(t *testing.T) {
	var env Environment
	assert.True(t, env.IsEmpty())

	env.ClassBytes = &comid.TestBytes
	assert.False(t, env.IsEmpty())
}

func TestEnvironment_RenderParts(t *testing.T) {
	oidType := comid.OIDType
	ueidType := comid.UEIDType
	uuidType := comid.UUIDType
	vendor := "foo"
	model := "bar"
	layer := uint64(1)
	index := uint64(0)

	testCases := []struct {
		title    string
		env      Environment
		expected [][2]string
		err      string
	}{
		{
			title: "ok full",
			env: Environment{
				Vendor:       &vendor,
				Model:        &model,
				ClassType:    &oidType,
				ClassBytes:   &[]byte{0x01, 0x02, 0x03, 0x04},
				InstanceType: &uuidType,
				InstanceBytes: &[]byte{
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
					0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
				},
				GroupType:  &ueidType,
				GroupBytes: &[]byte{0x09, 0x0a, 0x0b, 0x0c},
				Layer:      &layer,
				Index:      &index,
			},
			expected: [][2]string{
				{"vendor", "foo"},
				{"model", "bar"},
				{"class", "0.1.2.3.4"},
				{"instance", "01020304-0506-0708-090a-0b0c0d0e0f10"},
				{"group", "090a0b0c"},
				{"layer", "1"},
				{"index", "0"},
			},
		},
		{
			title: "ok minimal",
			env: Environment{
				Model: &model,
			},
			expected: [][2]string{
				{"model", "bar"},
			},
		},
		{
			title:    "ok empty",
			env:      Environment{},
			expected: nil,
		},
		{
			title: "bad",
			env: Environment{
				ClassBytes: &[]byte{0x01, 0x02, 0x03, 0x04},
			},
			err: "ClassType and ClassBytes must be set together",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			res, err := tc.env.RenderParts()
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, res)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}
