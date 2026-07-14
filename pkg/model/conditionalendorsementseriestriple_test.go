package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/util"
	"github.com/veraison/corim/comid"
)

func TestConditionalEndorsementSeriesTriple_round_trip(t *testing.T) {
	testSvn, err := comid.NewTaggedSVN(42)
	require.NoError(t, err)

	testCases := []struct {
		title  string
		triple comid.CondEndorseSeriesTriple
	}{
		{
			title: "ok",
			triple: comid.CondEndorseSeriesTriple{
				Condition: comid.CondEndorseSeriesCondition{
					Environment: comid.Environment{
						Instance: comid.MustNewUEIDInstance(comid.TestUEID),
					},
					Measurements: *comid.NewMeasurements().
						Add(&comid.Measurement{
							Val: comid.Mval{
								SVN: testSvn,
							},
						}),
					AuthorizedBy: comid.NewCryptoKeys().Add(comid.MustNewCryptoKeyTaggedBytes(
						[]byte{0x01, 0x02, 0x03},
					)),
				},
				Series: *comid.NewCondEndorseSeriesRecords().Add(&comid.CondEndorseSeriesRecord{
					Selection: *comid.NewMeasurements().
						Add(&comid.Measurement{
							Val: comid.Mval{
								SVN: testSvn,
							},
						}),
					Addition: *comid.NewMeasurements().
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
			cet, err := NewConditionalEndorsementSeriesTripleFromCoRIM(&tc.triple)
			assert.NoError(t, err)

			err = cet.Insert(ctx, db)
			require.NoError(t, err)

			selectedTriple, err := SelectConditionalEndorsementSeriesTriple(ctx, db, cet.ID)
			require.NoError(t, err)

			selectedCorimTriple, err := selectedTriple.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, &tc.triple, selectedCorimTriple)
		})
	}
}

func TestConditionalEndorsementSeriesTriple_Validate(t *testing.T) {
	testType := comid.BytesType
	testBytes := comid.MustHexDecode(t, "deadbeefdeadbeefdeadbeefdeadbeef")
	testCases := []struct {
		title string
		cest  ConditionalEndorsementSeriesTriple
		err   string
	}{
		{
			title: "ok",
			cest: ConditionalEndorsementSeriesTriple{
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
				AuthorizedBy: []*CryptoKey{
					&CryptoKey{
						KeyType:  comid.PKIXBase64CertType,
						KeyBytes: []byte(comid.TestCert),
					},
				},
				Series: []*ConditionalEndorsementSeriesRecord{
					{
						Selection: []*Measurement{
							{
								Digests: []*Digest{
									{
										AlgIDInt: int64(comid.Sha256),
										Value:    testBytes,
									},
								},
							},
						},
						Addition: []*Measurement{
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
			title: "condition environment not set",
			cest:  ConditionalEndorsementSeriesTriple{},
			err:   "condition environment not set",
		},
		{
			title: "invalid condition environment",
			cest: ConditionalEndorsementSeriesTriple{
				Environment: &Environment{
					ClassType: util.Ptr("foo"),
				},
			},
			err: "ClassType and ClassBytes must be set together",
		},
		{
			title: "empty series",
			cest: ConditionalEndorsementSeriesTriple{
				Environment: &Environment{
					ClassType:  util.Ptr("bytes"),
					ClassBytes: &[]byte{0x01, 0x02, 0x03},
				},
			},
			err: "empty series",
		},
		{
			title: "invalid series selection",
			cest: ConditionalEndorsementSeriesTriple{
				Environment: &Environment{
					ClassType:  util.Ptr("bytes"),
					ClassBytes: &[]byte{0x01, 0x02, 0x03},
				},
				Series: []*ConditionalEndorsementSeriesRecord{{}},
			},
			err: "series[0]: empty selection",
		},
		{
			title: "invalid series additon",
			cest: ConditionalEndorsementSeriesTriple{
				Environment: &Environment{
					ClassType:  util.Ptr("bytes"),
					ClassBytes: &[]byte{0x01, 0x02, 0x03},
				},
				Series: []*ConditionalEndorsementSeriesRecord{{
					Selection: []*Measurement{{}},
				}},
			},
			err: "series[0]: empty addition",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			err := tc.cest.Validate()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		})
	}
}

func TestConditionalEndorsementSeriesTriple_Delete(t *testing.T) {
	var cest ConditionalEndorsementSeriesTriple
	db := NewTestDB(t)

	err := cest.Delete(context.Background(), db)
	assert.ErrorContains(t, err, "ID not set")

	cest = ConditionalEndorsementSeriesTriple{
		ID:          1,
		Environment: &Environment{ID: 1},
		Series:      []*ConditionalEndorsementSeriesRecord{{ID: 1}},
	}
	err = cest.Delete(context.Background(), db)
	assert.NoError(t, err)
}

func TestConditionalEndorsementSeriesTriple_model_methods(t *testing.T) {
	cet := ConditionalEndorsementSeriesTriple{ID: 1}
	assert.Equal(t, cet.ID, cet.DbID())
	assert.Equal(t, "conditional_endorsement_series_triples", cet.TableName())
	assert.True(t, cet.IsTable())
	assert.Equal(t, cet.ModuleID, cet.OwnerDbID())
	assert.Equal(t, "module_tag", cet.OwnerName())
}

func TestConditionalEndorsementSeriesRecord_model_methods(t *testing.T) {
	cet := ConditionalEndorsementSeriesRecord{ID: 1}
	assert.Equal(t, cet.ID, cet.DbID())
	assert.Equal(t, "conditional_endorsement_series_records", cet.TableName())
	assert.True(t, cet.IsTable())
	assert.Equal(t, cet.TripleID, cet.OwnerDbID())
	assert.Equal(t, "conditional_endorsement_series_triple", cet.OwnerName())
}

func TestConditionalEndorsementSeriesRecords_CoRIM_conversions_nok(t *testing.T) {
	comidVals := comid.NewCondEndorseSeriesRecords()
	_, err := ConditionalEndorsementSeriesRecordsFromCoRIM(*comidVals)
	assert.ErrorContains(t, err, "no records")
}

func TestConditionalEndorsementSeriesRecord_Select_nok(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)
	record := ConditionalEndorsementSeriesRecord{}

	err := record.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	record.ID = 1
	err = record.Select(ctx, db)
	assert.ErrorContains(t, err, "no rows in result set")
}
