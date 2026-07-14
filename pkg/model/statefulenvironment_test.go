package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/comid"
)

func TestStatefulEnvironment_round_trip(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testSvn, err := comid.NewTaggedSVN(42)
	require.NoError(t, err)

	testCases := []struct {
		title string
		senv  comid.StatefulEnvironment
	}{
		{
			title: "ok",
			senv: comid.StatefulEnvironment{
				Environment: comid.Environment{
					Instance: comid.MustNewUEIDInstance(comid.TestUEID),
				},
				Measurements: *comid.NewMeasurements().
					Add(&comid.Measurement{
						Val: comid.Mval{
							SVN: testSvn,
						},
					}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			senv, err := NewStatefulEnvironmentFromCoRIM(&tc.senv)
			assert.NoError(t, err)

			err = senv.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectStatefulEnvironment(ctx, db, senv.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.senv, selectedCorimTriple)
		})
	}
}

func TestStatefulEnvironment_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		vt    StatefulEnvironment
		err   string
	}{
		{
			title: "ok",
			vt: StatefulEnvironment{
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
		{
			title: "missing environment",
			vt:    StatefulEnvironment{},
			err:   "environment not set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.vt.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestStatefulEnvironment_Delete(t *testing.T) {
	var vt StatefulEnvironment
	db := NewTestDB(t)

	err := vt.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	vt = StatefulEnvironment{
		ID:           1,
		Measurements: []*Measurement{{ID: 1}},
		Environment:  &Environment{ID: 1},
	}
	err = vt.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestStatefulEnvironment_model_methods(t *testing.T) {
	val := StatefulEnvironment{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "stateful_environments", val.TableName())
	assert.True(t, val.IsTable())
	assert.Equal(t, val.TripleID, val.OwnerDbID())
	assert.Equal(t, "conditional_endorsement_triple", val.OwnerName())
}
