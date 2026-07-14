package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/comid"
)

func TestDomainDependencyTriple_round_trip(t *testing.T) { // nolint:dupl
	testCases := []struct {
		title  string
		triple comid.DomainDependencyTriple
	}{
		{
			title: "ok",
			triple: comid.DomainDependencyTriple{
				DomainID: comid.Environment{
					Instance: comid.MustNewUEIDInstance(comid.TestUEID),
				},
				Trustees: []comid.Environment{
					{
						Instance: comid.MustNewUEIDInstance(comid.TestUEID),
					},
				},
			},
		},
	}

	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cet, err := NewDomainDependencyTripleFromCoRIM(&tc.triple)
			assert.NoError(t, err)

			err = cet.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectDomainDependencyTriple(ctx, db, cet.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.triple, selectedCorimTriple)
		})
	}
}

func TestDomainDependencyTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		ddt   DomainDependencyTriple
		err   string
	}{
		{
			title: "ok",
			ddt: DomainDependencyTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Trustees: []*DomainEntry{
					{
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
					},
				},
			},
		},
		{
			title: "domain ID not set",
			ddt:   DomainDependencyTriple{},
			err:   "domain ID not set",
		},
		{
			title: "invalid domain ID",
			ddt: DomainDependencyTriple{
				DomainID: &Environment{
					ClassBytes: &testBytes,
				},
			},
			err: "domain ID: ClassType and ClassBytes must be set together",
		},
		{
			title: "no trustees",
			ddt: DomainDependencyTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
			},
			err: "no trustees",
		},
		{
			title: "invalid trustee",
			ddt: DomainDependencyTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Trustees: []*DomainEntry{
					{
						Environment: &Environment{
							ClassBytes: &testBytes,
						},
					},
				},
			},
			err: "trustee[0]: ClassType and ClassBytes must be set together",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.ddt.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestDomainDependencyTriple_Delete(t *testing.T) {
	var ddt DomainDependencyTriple
	db := NewTestDB(t)

	err := ddt.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ddt = DomainDependencyTriple{
		ID:       1,
		DomainID: &Environment{ID: 1},
		Trustees: []*DomainEntry{{ID: 1}},
	}
	err = ddt.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestDomainDependencyTriple_model_methods(t *testing.T) {
	triple := DomainDependencyTriple{ID: 1}
	assert.Equal(t, triple.ID, triple.DbID())
	assert.Equal(t, "domain_dependency_triples", triple.TableName())
	assert.True(t, triple.IsTable())
	assert.Equal(t, triple.ModuleID, triple.OwnerDbID())
	assert.Equal(t, "module_tag", triple.OwnerName())
}

func TestDomainMembershipTriple_round_trip(t *testing.T) { // nolint:dupl
	testCases := []struct {
		title  string
		triple comid.DomainMembershipTriple
	}{
		{
			title: "ok",
			triple: comid.DomainMembershipTriple{
				DomainID: comid.Environment{
					Instance: comid.MustNewUEIDInstance(comid.TestUEID),
				},
				Members: []comid.Environment{
					{
						Instance: comid.MustNewUEIDInstance(comid.TestUEID),
					},
				},
			},
		},
	}

	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cet, err := NewDomainMembershipTripleFromCoRIM(&tc.triple)
			assert.NoError(t, err)

			err = cet.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectDomainMembershipTriple(ctx, db, cet.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.triple, selectedCorimTriple)
		})
	}
}

func TestDomainMembershipTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		ddt   DomainMembershipTriple
		err   string
	}{
		{
			title: "ok",
			ddt: DomainMembershipTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Members: []*DomainEntry{
					{
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
					},
				},
			},
		},
		{
			title: "domain ID not set",
			ddt:   DomainMembershipTriple{},
			err:   "domain ID not set",
		},
		{
			title: "invalid domain ID",
			ddt: DomainMembershipTriple{
				DomainID: &Environment{
					ClassBytes: &testBytes,
				},
			},
			err: "domain ID: ClassType and ClassBytes must be set together",
		},
		{
			title: "no members",
			ddt: DomainMembershipTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
			},
			err: "no members",
		},
		{
			title: "invalid trustee",
			ddt: DomainMembershipTriple{
				DomainID: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Members: []*DomainEntry{
					{
						Environment: &Environment{
							ClassBytes: &testBytes,
						},
					},
				},
			},
			err: "member[0]: ClassType and ClassBytes must be set together",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.ddt.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestDomainMembershipTriple_Delete(t *testing.T) {
	var ddt DomainMembershipTriple
	db := NewTestDB(t)

	err := ddt.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	ddt = DomainMembershipTriple{
		ID:       1,
		DomainID: &Environment{ID: 1},
		Members:  []*DomainEntry{{ID: 1}},
	}
	err = ddt.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestDomainMembershipTriple_model_methods(t *testing.T) {
	triple := DomainMembershipTriple{ID: 1}
	assert.Equal(t, triple.ID, triple.DbID())
	assert.Equal(t, "domain_membership_triples", triple.TableName())
	assert.True(t, triple.IsTable())
	assert.Equal(t, triple.ModuleID, triple.OwnerDbID())
	assert.Equal(t, "module_tag", triple.OwnerName())
}

func TestDomainEntry_model_methods(t *testing.T) {
	entry := DomainEntry{ID: 1}
	assert.Equal(t, entry.ID, entry.DbID())
	assert.Equal(t, "domain_entries", entry.TableName())
	assert.True(t, entry.IsTable())
	assert.Equal(t, entry.OwnerID, entry.OwnerDbID())
	assert.Equal(t, entry.OwnerType, entry.OwnerName())
}
