package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim/comid"
)

func TestIntMatcher_round_trip(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct{
		title string
		matcher *comid.NumberMatcher
	}{
		{
			title: "unit",
			matcher: comid.NewNumberMatcher(comid.NumberFromInt(1)),
		},
		{
			title: "set",
			matcher: comid.NewNumberMatcher(
				comid.NumberFromInt(1),
				comid.NumberFromInt(2),
			),
		},
		{
			title: "range",
			matcher: comid.MustNewNumberRange(
				comid.NumberFromInt(1),
				comid.NumberFromInt(2),
			),
		},
		{
			title: "lower bound",
			matcher: comid.NewNumberLowerBound(comid.NumberFromInt(1)),
		},
		{
			title: "upper bound",
			matcher: comid.NewNumberUpperBound(comid.NumberFromInt(2)),
		},
	}


	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			matcher, err := NewIntMatcherFromCoRIM(tc.matcher)
			assert.NoError(t, err)

			err = matcher.Insert(ctx, db)
			require.NoError(t, err)

			selectedModel, err := SelectIntMatcher(ctx, db, matcher.ID)
			require.NoError(t, err)

			selectedCorimMatcher, err := selectedModel.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, tc.matcher, selectedCorimMatcher)
		})
	}
}

func TestIntMatcher_CRUD(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	matcher := IntMatcher{
		Type: comid.UnitType,
		Values: []*IntMatcherElement{
			{
				Value: 1,
			},
		},
	}

	err := matcher.Delete(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	err = matcher.Insert(ctx, db)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, matcher.ID)

	other := IntMatcher{}
	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	other.ID = 1
	err = other.Select(ctx, db)
	assert.NoError(t, err)

	err = matcher.Delete(ctx, db)
	assert.NoError(t, err)

	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "sql: no rows in result set")

	elem := IntMatcherElement{}
	err = elem.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

}

func TestIntMatcher_model_methods(t *testing.T) {
	val := IntMatcher{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "int_matchers", val.TableName())
	assert.True(t, val.IsTable())
}

func TestIntMatcherElement_model_methods(t *testing.T) {
	val := IntMatcherElement{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "int_matcher_elements", val.TableName())
	assert.True(t, val.IsTable())
}

func TestFloatMatcher_round_trip(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct{
		title string
		matcher *comid.NumberMatcher
	}{
		{
			title: "unit",
			matcher: comid.NewNumberMatcher(comid.NumberFromFloat64(1.5)),
		},
		{
			title: "set",
			matcher: comid.NewNumberMatcher(
				comid.NumberFromFloat64(1.5),
				comid.NumberFromFloat64(2.5),
			),
		},
		{
			title: "range",
			matcher: comid.MustNewNumberRange(
				comid.NumberFromFloat64(1.5),
				comid.NumberFromFloat64(2.5),
			),
		},
		{
			title: "lower bound",
			matcher: comid.NewNumberLowerBound(comid.NumberFromFloat64(1.5)),
		},
		{
			title: "upper bound",
			matcher: comid.NewNumberUpperBound(comid.NumberFromFloat64(2.5)),
		},
	}


	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			matcher, err := NewFloatMatcherFromCoRIM(tc.matcher)
			assert.NoError(t, err)

			err = matcher.Insert(ctx, db)
			require.NoError(t, err)

			selectedModel, err := SelectFloatMatcher(ctx, db, matcher.ID)
			require.NoError(t, err)

			selectedCorimMatcher, err := selectedModel.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, tc.matcher, selectedCorimMatcher)
		})
	}
}

func TestFloatMatcher_CRUD(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	matcher := FloatMatcher{
		Type: comid.UnitType,
		Values: []*FloatMatcherElement{
			{
				Value: 1.0,
			},
		},
	}

	err := matcher.Delete(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	err = matcher.Insert(ctx, db)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, matcher.ID)

	other := FloatMatcher{}
	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	other.ID = 1
	err = other.Select(ctx, db)
	assert.NoError(t, err)

	err = matcher.Delete(ctx, db)
	assert.NoError(t, err)

	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "sql: no rows in result set")

	elem := FloatMatcherElement{}
	err = elem.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

}

func TestFloatMatcher_model_methods(t *testing.T) {
	val := FloatMatcher{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "float_matchers", val.TableName())
	assert.True(t, val.IsTable())
}

func TestFloatMatcherElement_model_methods(t *testing.T) {
	val := FloatMatcherElement{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "float_matcher_elements", val.TableName())
	assert.True(t, val.IsTable())
}

func TestTextMatcher_round_trip(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct{
		title string
		matcher *comid.TextMatcher
	}{
		{
			title: "unit",
			matcher: comid.NewTextMatcher("foo"),
		},
		{
			title: "set",
			matcher: comid.NewTextMatcher("foo", "bar"),
		},
	}


	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			matcher, err := NewTextMatcherFromCoRIM(tc.matcher)
			assert.NoError(t, err)

			err = matcher.Insert(ctx, db)
			require.NoError(t, err)

			selectedModel, err := SelectTextMatcher(ctx, db, matcher.ID)
			require.NoError(t, err)

			selectedCorimMatcher, err := selectedModel.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, tc.matcher, selectedCorimMatcher)
		})
	}
}

func TestTextMatcher_CRUD(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	matcher := TextMatcher{
		Type: comid.UnitType,
		Values: []*TextMatcherElement{
			{
				Value: "foo",
			},
		},
	}

	err := matcher.Delete(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	err = matcher.Insert(ctx, db)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, matcher.ID)

	other := TextMatcher{}
	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	other.ID = 1
	err = other.Select(ctx, db)
	assert.NoError(t, err)

	err = matcher.Delete(ctx, db)
	assert.NoError(t, err)

	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "sql: no rows in result set")

	elem := TextMatcherElement{}
	err = elem.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

}

func TestTextMatcher_model_methods(t *testing.T) {
	val := TextMatcher{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "text_matchers", val.TableName())
	assert.True(t, val.IsTable())
}

func TestTextMatcherElement_model_methods(t *testing.T) {
	val := TextMatcherElement{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "text_matcher_elements", val.TableName())
	assert.True(t, val.IsTable())
}

func TestBytesMatcher_round_trip(t *testing.T) {
	ctx := context.Background()
	db := NewTestDB(t)
	defer func() { assert.NoError(t, db.Close()) }()

	testCases := []struct{
		title string
		matcher *comid.BytesMatcher
	}{
		{
			title: "unit",
			matcher: comid.NewBytesMatcher([]byte{0xde, 0xad, 0xbe, 0xef}),
		},
		{
			title: "set",
			matcher: comid.NewBytesMatcher(
				[]byte{0xde, 0xad, 0xbe, 0xef},
				[]byte{0xc0, 0xde, 0xca, 0xfe},
			),
		},
	}


	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			matcher, err := NewBytesMatcherFromCoRIM(tc.matcher)
			assert.NoError(t, err)

			err = matcher.Insert(ctx, db)
			require.NoError(t, err)

			selectedModel, err := SelectBytesMatcher(ctx, db, matcher.ID)
			require.NoError(t, err)

			selectedCorimMatcher, err := selectedModel.ToCoRIM()
			assert.NoError(t, err)

			assert.Equal(t, tc.matcher, selectedCorimMatcher)
		})
	}
}

func TestBytesMatcher_CRUD(t *testing.T) { // nolint:dupl
	ctx := context.Background()
	db := NewTestDB(t)
	matcher := BytesMatcher{
		Type: comid.UnitType,
		Values: []*BytesMatcherElement{
			{
				Value: []byte{0xde, 0xad, 0xbe, 0xef},
			},
		},
	}

	err := matcher.Delete(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	err = matcher.Insert(ctx, db)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, matcher.ID)

	other := BytesMatcher{}
	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

	other.ID = 1
	err = other.Select(ctx, db)
	assert.NoError(t, err)

	err = matcher.Delete(ctx, db)
	assert.NoError(t, err)

	err = other.Select(ctx, db)
	assert.ErrorContains(t, err, "sql: no rows in result set")

	elem := BytesMatcherElement{}
	err = elem.Select(ctx, db)
	assert.ErrorContains(t, err, "ID not set")

}

func TestBytesMatcher_model_methods(t *testing.T) {
	val := BytesMatcher{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "bytes_matchers", val.TableName())
	assert.True(t, val.IsTable())
}

func TestBytesMatcherElement_model_methods(t *testing.T) {
	val := BytesMatcherElement{ID: 1}
	assert.Equal(t, val.ID, val.DbID())
	assert.Equal(t, "bytes_matcher_elements", val.TableName())
	assert.True(t, val.IsTable())
}
