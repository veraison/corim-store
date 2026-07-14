package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/comid"
)

func TestConditionalEndorsementTriple_round_trip(t *testing.T) {
	testSvn, err := comid.NewTaggedSVN(42)
	require.NoError(t, err)

	testCases := []struct {
		title  string
		triple comid.CondEndorseTriple
	}{
		{
			title: "ok",
			triple: comid.CondEndorseTriple{
				Conditions: *comid.NewStatefulEnvironments().Add(&comid.StatefulEnvironment{
					Environment: comid.Environment{
						Instance: comid.MustNewUEIDInstance(comid.TestUEID),
					},
					Measurements: *comid.NewMeasurements().
						Add(&comid.Measurement{
							Val: comid.Mval{
								SVN: testSvn,
							},
						}),
				}),
				Endorsements: *comid.NewValueTriples().Add(&comid.ValueTriple{
					Environment: comid.Environment{
						Instance: comid.MustNewUEIDInstance(comid.TestUEID),
					},
					Measurements: *comid.NewMeasurements().
						Add(&comid.Measurement{
							Val: comid.Mval{
								SVN: testSvn,
							},
						}),
				}),
			},
		},
	}

	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			cet, err := NewConditionalEndorsementTripleFromCoRIM(&tc.triple)
			assert.NoError(t, err)

			err = cet.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectConditionalEndorsementTriple(ctx, db, cet.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.triple, selectedCorimTriple)
		})
	}
}

func TestConditionalEndorsementTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		cet   ConditionalEndorsementTriple
		err   string
	}{
		{
			title: "ok",
			cet: ConditionalEndorsementTriple{
				Conditions: []*StatefulEnvironment{
					{
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
					},
				},
				Endorsements: []*ValueTriple{
					{
						Type: EndorsedValueTriple,
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
						Measurements: []*Measurement{
							{
								Digests: []*Digest{
									{
										AlgIDInt: int64(comid.Sha256),
										Value:    testBytes,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			title: "conditions not set",
			cet:   ConditionalEndorsementTriple{},
			err:   "conditions not set",
		},
		{
			title: "bad conditions",
			cet: ConditionalEndorsementTriple{
				Conditions: []*StatefulEnvironment{
					{
						Environment: &Environment{
							ClassBytes: &testBytes,
						},
					},
				},
			},
			err: "conditions[0]: environment: ClassType and ClassBytes must be set together",
		},
		{
			title: "endorsements not set",
			cet: ConditionalEndorsementTriple{
				Conditions: []*StatefulEnvironment{
					{
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
					},
				},
			},
			err: "endorsements not set",
		},
		{
			title: "bad endorsements",
			cet: ConditionalEndorsementTriple{
				Conditions: []*StatefulEnvironment{
					{
						Environment: &Environment{
							ClassType:  &testType,
							ClassBytes: &testBytes,
						},
					},
				},
				Endorsements: []*ValueTriple{
					{
						Type: EndorsedValueTriple,
						Environment: &Environment{
							ClassBytes: &testBytes,
						},
					},
				},
			},
			err: "endorsements[0]: environment: ClassType and ClassBytes must be set together",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.cet.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestConditionalEndorsementTriple_Insert(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)

	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	triple := ConditionalEndorsementTriple{
		Conditions: []*StatefulEnvironment{
			{
				Environment: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
			},
		},
		Endorsements: []*ValueTriple{
			{
				Type: EndorsedValueTriple,
				Environment: &Environment{
					ClassBytes: &testBytes,
				},
			},
		},
	}

	err := triple.Insert(ctx, db)
	assert.ErrorContains(t, err, "ClassType and ClassBytes must be set together")
}

func TestConditionalEndorsementTriple_Select_nok(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)

	_, err := SelectConditionalEndorsementTriple(ctx, db, 1)
	assert.ErrorContains(t, err, "no rows in result set")

	triple := ConditionalEndorsementTriple{}
	err = triple.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")
}

func TestConditionalEndorsementTriple_Delete(t *testing.T) {
	var cet ConditionalEndorsementTriple
	db := NewTestDB(t)

	err := cet.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	cet = ConditionalEndorsementTriple{
		ID:           1,
		Conditions:   []*StatefulEnvironment{{ID: 1}},
		Endorsements: []*ValueTriple{{ID: 1}},
	}
	err = cet.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestConditionalEndorsementTriple_model_methods(t *testing.T) {
	cet := ConditionalEndorsementTriple{ID: 1}
	assert.Equal(t, cet.ID, cet.DbID())
	assert.Equal(t, "conditional_endorsement_triples", cet.TableName())
	assert.True(t, cet.IsTable())
	assert.Equal(t, cet.ModuleID, cet.OwnerDbID())
	assert.Equal(t, "module_tag", cet.OwnerName())
}
