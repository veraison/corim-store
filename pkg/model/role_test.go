package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func TestParseCoRIMRole(t *testing.T) {
	testCases := []struct {
		role     string
		expected corim.Role
		err      string
	}{
		{
			role:     "manifestCreator",
			expected: corim.RoleManifestCreator,
		},
		{
			role:     "manifestSigner",
			expected: corim.RoleManifestSigner,
		},
		{
			role:     "Role(3)",
			expected: corim.Role(3),
		},
		{
			role: "foo",
			err:  "invalid CoRIM role: foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.role, func(t *testing.T) {
			role, err := ParseCoRIMRole(tc.role)
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, role)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestParseCoMIDRole(t *testing.T) {
	testCases := []struct {
		role     string
		expected comid.Role
		err      string
	}{
		{
			role:     "creator",
			expected: comid.RoleCreator,
		},
		{
			role:     "maintainer",
			expected: comid.RoleMaintainer,
		},
		{
			role:     "Role(3)",
			expected: comid.Role(3),
		},
		{
			role: "foo",
			err:  "invalid CoMID role: foo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.role, func(t *testing.T) {
			role, err := ParseCoMIDRole(tc.role)
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, role)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestMustNewCoRIMRoleEntry(t *testing.T) {
	entry := MustNewCoRIMRoleEntry("manifestCreator")
	assert.Equal(t, "manifestCreator", entry.Role)

	assert.Panics(t, func() { MustNewCoRIMRoleEntry("foo") })
}

func TestMustNewCoMIDRoleEntry(t *testing.T) {
	entry := MustNewCoMIDRoleEntry("creator")
	assert.Equal(t, "creator", entry.Role)

	assert.Panics(t, func() { MustNewCoMIDRoleEntry("foo") })
}
