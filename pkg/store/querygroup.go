package store

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"github.com/veraison/corim-store/pkg/model"
)

// QueryGroup allows multiple queries of the same time to be run together,
// concatenating their results, effectively constructing a disjunction (OR
// expression) between individual queries.
type QueryGroup[M model.Model, Q Query[M]] struct {
	subqueries []Q
}

func NewQueryGroup[M model.Model, T Query[M]](sub ...T) *QueryGroup[M, T] {
	return &QueryGroup[M, T]{sub}
}

func (o *QueryGroup[M, Q]) Add(sub ...Q) *QueryGroup[M, Q] {
	o.subqueries = append(o.subqueries, sub...)
	return o
}

func (o *QueryGroup[M, Q]) ForEach(updater func(v Q)) {
	for _, sub := range o.subqueries {
		updater(sub)
	}
}

func (o *QueryGroup[M, Q]) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		for _, sub := range o.subqueries {
			q.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				sub.UpdateSelectQuery(q, dialect)
				return q
			})
		}

		return q
	})
}

func (o *QueryGroup[M, Q]) Run(ctx context.Context, db *bun.DB) ([]M, error) {
	results, err := o.RunGroup(ctx, db)
	if err != nil {
		return nil, err
	}

	unique := make(map[int64]M)
	for _, result := range results {
		for _, val := range result {
			unique[val.DbID()] = val
		}
	}

	ret := make([]M, 0, len(unique))
	for _, val := range unique {
		ret = append(ret, val)
	}

	return ret, nil
}

func (o *QueryGroup[M, Q]) RunGroup(ctx context.Context, db *bun.DB) ([][]M, error) {
	ret := make([][]M, len(o.subqueries))
	for i, sub := range o.subqueries {
		subResult, err := sub.Run(ctx, db)
		if err != nil {
			if !errors.Is(err, ErrNoMatch) {
				return nil, err
			}

			continue
		}

		ret[i] = subResult
	}

	if len(ret) == 0 {
		return nil, ErrNoMatch
	}

	return ret, nil
}

func (o *QueryGroup[M, Q]) Length() int {
	return len(o.subqueries)
}

func (o *QueryGroup[M, Q]) IsEmpty() bool {
	for _, sub := range o.subqueries {
		if !sub.IsEmpty() {
			return false
		}
	}

	return true
}

type MeasurementQueryGroup = QueryGroup[*model.Measurement, *MeasurementQuery]

func NewMeasurementQueryGroup() *MeasurementQueryGroup {
	return NewQueryGroup[*model.Measurement, *MeasurementQuery]()
}

type KeyTripleQueryGroup = QueryGroup[*model.KeyTripleEntry, *KeyTripleQuery]

func NewKeyTripleQueryGroup() *KeyTripleQueryGroup {
	return NewQueryGroup[*model.KeyTripleEntry, *KeyTripleQuery]()
}

type ValueTripleQueryGroup = QueryGroup[*model.ValueTripleEntry, *ValueTripleQuery]

func NewValueTripleQueryGroup() *ValueTripleQueryGroup {
	return NewQueryGroup[*model.ValueTripleEntry, *ValueTripleQuery]()
}
