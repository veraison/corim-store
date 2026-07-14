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

func (o *QueryGroup[M, Q]) Run(ctx context.Context, db bun.IDB) ([]M, error) {
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

func (o *QueryGroup[M, Q]) RunGroup(ctx context.Context, db bun.IDB) ([][]M, error) {
	ret := make([][]M, 0, len(o.subqueries))
	for _, sub := range o.subqueries {
		subResult, err := sub.Run(ctx, db)
		if err != nil {
			if !errors.Is(err, ErrNoMatch) {
				return nil, err
			}

			continue
		}

		ret = append(ret, subResult)
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

// OwnedModelQueryGroup is a specialised query group whose members are queries
// on OwnedModels.
type OwnedModelQueryGroup[M model.OwnedModel, Q Query[M]] struct {
	QueryGroup[M, Q]

	// OwnerName is the name of the owner model. This is used to ensure
	// owners of consitent type when dealing with polymorphic ownership
	// (mixing owner IDs for owner of different types does not make sense).
	OwnerName string
}

func NewOwnedModelQueryGroup[M model.OwnedModel, Q Query[M]](ownerName string) *OwnedModelQueryGroup[M, Q] {
	return &OwnedModelQueryGroup[M, Q]{OwnerName: ownerName}
}

// RunOwnerConjunction performs a conjuction (logical AND operation) on owner
// IDs in its member sub-quries' results. I.e. it returns IDs of owners that
// feature in all members' results. For example, this can be used to identify
// IDs of ValueTriples that own measurements of multple categries (each
// category matched my the member of the OwnedModelQueryGroup).
func (o *OwnedModelQueryGroup[M, Q]) RunOwnerConjunction(ctx context.Context, db bun.IDB) ([]int64, error) {
	results, err := o.RunGroup(ctx, db)

	// as the group is conjunctive, every member of the group must match,
	// so we verify that the number of results is the same as the number
	// of group members.
	if err != nil || len(results) != o.Length() {
		if err == nil {
			err = ErrNoMatch
		}

		return nil, err
	}

	// We want the owner IDs that appear in every group member's result. To
	// do that, we first compute the unique set of owner IDs for each
	// result, and then increment a counter for each ID in the set. Once we
	// finish processing all group members, countMap will map an owner ID
	// onto the number of results it appeared in; if this number matches
	// the number of group members, that means the owner ID appears in the
	// matches for every member in the group.

	countMap := make(map[int64]int)
	modelMap := make(map[int64]M)
	for _, result := range results {
		ownerIDs := make(map[int64]bool)
		for _, ownedModel := range result {
			if ownedModel.OwnerName() != o.OwnerName {
				continue
			}

			ownerIDs[ownedModel.OwnerDbID()] = true
			modelMap[ownedModel.DbID()] = ownedModel
		}

		for ownerID := range ownerIDs {
			countMap[ownerID] += 1
		}
	}

	ret := make([]int64, 0, len(modelMap))
	for _, ownedModel := range modelMap {
		if countMap[ownedModel.OwnerDbID()] == o.Length() {
			ret = append(ret, ownedModel.OwnerDbID())
		}
	}

	if len(ret) == 0 {
		return nil, ErrNoMatch
	}

	return ret, nil
}

type KeyTripleQueryGroup = QueryGroup[*model.KeyTripleEntry, *KeyTripleQuery]

func NewKeyTripleQueryGroup() *KeyTripleQueryGroup {
	return NewQueryGroup[*model.KeyTripleEntry, *KeyTripleQuery]()
}

type ValueTripleQueryGroup = QueryGroup[*model.ValueTripleEntry, *ValueTripleQuery]

func NewValueTripleQueryGroup() *ValueTripleQueryGroup {
	return NewQueryGroup[*model.ValueTripleEntry, *ValueTripleQuery]()
}

type ConditionalEndorsementTripleQueryGroup = QueryGroup[
	*model.ConditionalEndorsementTripleEntry,
	*ConditionalEndorsementTripleQuery,
]

func NewConditionalEndorsementTripleQueryGroup() *ConditionalEndorsementTripleQueryGroup {
	return NewQueryGroup[*model.ConditionalEndorsementTripleEntry, *ConditionalEndorsementTripleQuery]()
}

type ConditionalEndorsementSeriesTripleQueryGroup = QueryGroup[
	*model.ConditionalEndorsementSeriesTripleEntry,
	*ConditionalEndorsementSeriesTripleQuery,
]

func NewConditionalEndorsementSeriesTripleQueryGroup() *ConditionalEndorsementSeriesTripleQueryGroup {
	return NewQueryGroup[
		*model.ConditionalEndorsementSeriesTripleEntry,
		*ConditionalEndorsementSeriesTripleQuery,
	]()
}

type DomainDependencyTripleQueryGroup = QueryGroup[
	*model.DomainDependencyTripleEntry,
	*DomainDependencyTripleQuery,
]

func NewDomainDependencyTripleQueryGroup() *DomainDependencyTripleQueryGroup {
	return NewQueryGroup[*model.DomainDependencyTripleEntry, *DomainDependencyTripleQuery]()
}

type DomainMembershipTripleQueryGroup = QueryGroup[
	*model.DomainMembershipTripleEntry,
	*DomainMembershipTripleQuery,
]

func NewDomainMembershipTripleQueryGroup() *DomainMembershipTripleQueryGroup {
	return NewQueryGroup[*model.DomainMembershipTripleEntry, *DomainMembershipTripleQuery]()
}

type ManifestQueryGroup = QueryGroup[*model.ManifestEntry, *ManifestQuery]

func NewManifestQueryGroup() *ManifestQueryGroup {
	return NewQueryGroup[*model.ManifestEntry, *ManifestQuery]()
}

type ModuleTagQueryGroup = QueryGroup[*model.ModuleTagEntry, *ModuleTagQuery]

func NewModuleTagQueryGroup() *ModuleTagQueryGroup {
	return NewQueryGroup[*model.ModuleTagEntry, *ModuleTagQuery]()
}

type MeasurementQueryGroup = OwnedModelQueryGroup[*model.Measurement, *MeasurementQuery]

func NewMeasurementQueryGroup(ownerName string) *MeasurementQueryGroup {
	return NewOwnedModelQueryGroup[*model.Measurement, *MeasurementQuery](ownerName)
}

type StatefulEnvironmentQueryGroup = OwnedModelQueryGroup[*model.StatefulEnvironment, *StatefulEnvironmentQuery]

func NewStatefulEnvironmentQueryGroup() *StatefulEnvironmentQueryGroup {
	return NewOwnedModelQueryGroup[*model.StatefulEnvironment, *StatefulEnvironmentQuery](
		"conditional_endorsement_triple",
	)
}

type EndorsementQueryGroup = OwnedModelQueryGroup[*model.ValueTriple, *EndorsementQuery]

func NewEndorsementQueryGroup() *EndorsementQueryGroup {
	return NewOwnedModelQueryGroup[*model.ValueTriple, *EndorsementQuery]("conditional_endorsement_triple")
}

type ConditionalEndorsementSeriesRecordQueryGroup = OwnedModelQueryGroup[
	*model.ConditionalEndorsementSeriesRecord,
	*ConditionalEndorsementSeriesRecordQuery,
]

func NewConditionalEndorsementSeriesRecordQueryGroup() *ConditionalEndorsementSeriesRecordQueryGroup {
	return NewOwnedModelQueryGroup[
		*model.ConditionalEndorsementSeriesRecord,
		*ConditionalEndorsementSeriesRecordQuery,
	]("conditional_endorsement_series_triple")
}
