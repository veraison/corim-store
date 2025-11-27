package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/extensions"
)

type flagsMapExtension struct {
	Foo bool `cbor:"-1,keyasint" json:"foo"`
}

type badFlagsMapExtension struct {
	Bar string `cbor:"-1,keyasint" json:"bar"`
}

func TestFlags_round_trip(t *testing.T) {
	trueVal := true
	falseVal := false

	testCases := []struct {
		title    string
		flags    *comid.FlagsMap
		expected []*Flag
		err      string
	}{
		{
			title: "ok full",
			flags: &comid.FlagsMap{
				IsConfigured:               &trueVal,
				IsSecure:                   &trueVal,
				IsRecovery:                 &falseVal,
				IsDebug:                    &falseVal,
				IsReplayProtected:          &trueVal,
				IsIntegrityProtected:       &falseVal,
				IsRuntimeMeasured:          &trueVal,
				IsImmutable:                &trueVal,
				IsTcb:                      &trueVal,
				IsConfidentialityProtected: &trueVal,
			},
			expected: []*Flag{
				&Flag{CodePoint: IsConfiguredFlag, Value: true},
				&Flag{CodePoint: IsSecureFlag, Value: true},
				&Flag{CodePoint: IsRecoveryFlag, Value: false},
				&Flag{CodePoint: IsDebugFlag, Value: false},
				&Flag{CodePoint: IsReplayProtectedFlag, Value: true},
				&Flag{CodePoint: IsIntegrityProtectedFlag, Value: false},
				&Flag{CodePoint: IsRuntimeMeasuredFlag, Value: true},
				&Flag{CodePoint: IsImmutableFlag, Value: true},
				&Flag{CodePoint: IsTcbFlag, Value: true},
				&Flag{CodePoint: IsConfidentialityProtectedFlag, Value: true},
			},
		},
		{
			title: "ok minimal",
			flags: &comid.FlagsMap{
				IsSecure: &falseVal,
			},
			expected: []*Flag{
				&Flag{CodePoint: IsSecureFlag, Value: false},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			flags, err := FlagsFromCoRIM(tc.flags)
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, flags)

				other, err := FlagsToCoRIM(flags)
				assert.NoError(t, err)
				assert.Equal(t, tc.flags, other)

			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}

	flagsMap := comid.NewFlagsMap()
	err := flagsMap.RegisterExtensions(extensions.Map{
		comid.ExtFlags: &flagsMapExtension{},
	})
	require.NoError(t, err)
	err = flagsMap.Set("foo", true)
	require.NoError(t, err)

	flags, err := FlagsFromCoRIM(flagsMap)
	assert.NoError(t, err)
	assert.Equal(t, []*Flag{&Flag{CodePoint: -1, Value: true}}, flags)

	other, err := FlagsToCoRIM(flags)
	assert.NoError(t, err)

	val, err := other.GetBool("-1")
	assert.NoError(t, err)
	assert.Equal(t, true, val)

	flagsMap = comid.NewFlagsMap()
	err = flagsMap.RegisterExtensions(extensions.Map{
		comid.ExtFlags: &badFlagsMapExtension{},
	})
	require.NoError(t, err)
	err = flagsMap.Set("bar", "qux")
	require.NoError(t, err)

	_, err = FlagsFromCoRIM(flagsMap)
	assert.ErrorContains(t, err, "invalid Flags extension")
}

func TestFlag_Select(t *testing.T) {
	var digest Flag
	db := test.NewTestDB(t)

	err := digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	digest.ID = 1
	err = digest.Select(context.Background(), db)
	assert.ErrorContains(t, err, "no rows in result")
}

func TestFlag_Delete(t *testing.T) {
	var flag Flag
	db := test.NewTestDB(t)

	err := flag.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	flag.ID = 1
	err = flag.Delete(context.Background(), db)
	assert.NoError(t, err)
}
