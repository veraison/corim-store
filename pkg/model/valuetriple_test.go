package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/test"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

func TestValueTriple_round_trip(t *testing.T) {
	ctx := context.Background()
	db := test.NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testSvn, err := comid.NewTaggedSVN(42)
	require.NoError(t, err)

	testCases := []struct {
		title string
		vt    comid.ValueTriple
	}{
		{
			title: "ok",
			vt: comid.ValueTriple{
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
			vt, err := NewValueTripleFromCoRIM(&tc.vt)
			assert.NoError(t, err)

			// the type of the ValueTriple isn't part of the struct
			// but is dependent on where it used. Since we do not
			// have the winder context, set it arbitrarily here.
			vt.Type = ReferenceValueTriple

			err = vt.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectValueTriple(ctx, db, vt.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.vt, selectedCorimTriple)
		})
	}
}

func TestValueTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		vt    ValueTriple
		err   string
	}{
		{
			title: "ok",
			vt: ValueTriple{
				Type: EndorsedValueTriple,
				Environment: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Measurements: []*Measurement{
					{
						Digests: []*Digest{
							{
								AlgID: swid.Sha256,
								Value: testBytes,
							},
						},
					},
				},
			},
		},
		{
			title: "missing type",
			vt:    ValueTriple{},
			err:   "value triple type not set",
		},
		{
			title: "missing environment",
			vt: ValueTriple{
				Type: ReferenceValueTriple,
			},
			err: "environment not set",
		},
		{
			title: "missing measurements",
			vt: ValueTriple{
				Type: EndorsedValueTriple,
				Environment: &Environment{
					ClassType:  &testType,
					ClassBytes: &testBytes,
				},
				Measurements: []*Measurement{},
			},
			err: "no measurements",
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

func TestValueTriple_Delete(t *testing.T) {
	var vt ValueTriple
	db := test.NewTestDB(t)

	err := vt.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	vt = ValueTriple{
		ID:           1,
		Measurements: []*Measurement{{ID: 1}},
		Environment:  &Environment{ID: 1},
	}
	err = vt.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestValueTriple_TripleType(t *testing.T) {
	var vt ValueTriple
	assert.Equal(t, "value", vt.TripleType())
}

func TestValueTriple_DatabaseID(t *testing.T) {
	var vt ValueTriple
	assert.Equal(t, int64(0), vt.DatabaseID())

	vt.ID = 1
	assert.Equal(t, int64(1), vt.DatabaseID())
}
