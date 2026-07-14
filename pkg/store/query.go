package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
	"github.com/veraison/eat"
	"github.com/veraison/swid"
)

// Query is the interface implemented by all objects that can be used to query
// the contents of the Store.
type Query[T model.Model] interface {
	// UpdateSelectQuery updates the bun.SelectQuery using parameters
	// defined by this query object. This only includes the parameters for
	// the database model corresponding to T, and not any sub-queries that
	// might be part of this Query.
	UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect)
	// Run this Query against the provided database. If the Query contains
	// sub-queries, those are run first, and their results are used to
	// update the IDs in the main query.
	Run(ctx context.Context, db bun.IDB) ([]T, error)
	// IsEmpty returns true if this Query, and any sub-queries, does not
	// contain any parameters. When an empty query is run, all entries for
	// the model corresponding to T in the store are returned.
	IsEmpty() bool
}

type HrefQuery struct {
	modelQuery

	values     []string
	locatorIDs []int64
}

func NewHrefQuery() *HrefQuery {
	return &HrefQuery{}
}

func (o *HrefQuery) ID(value ...int64) *HrefQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *HrefQuery) Value(value ...string) *HrefQuery {
	o.values = append(o.values, value...)
	return o
}

func (o *HrefQuery) LocatorID(value ...int64) *HrefQuery {
	o.locatorIDs = append(o.locatorIDs, value...)
	return o
}

func (o *HrefQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("value", o.values, false, query, dialect)
	addOrGroupWhereClause("locator_id", o.locatorIDs, false, query, dialect)
}

func (o *HrefQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Href, error) {
	return runQuery(ctx, db, o)
}

func (o *HrefQuery) IsEmpty() bool {
	return len(o.values) == 0 &&
		len(o.locatorIDs) == 0 &&
		o.modelQuery.IsEmpty()
}

type LocatorQuery struct {
	modelQuery

	manifestIDs []int64

	hrefQuery    *HrefQuery
	digestsQuery *DigestQuery
}

func NewLocatorQuery() *LocatorQuery {
	return &LocatorQuery{}
}

func (o *LocatorQuery) ID(value ...int64) *LocatorQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *LocatorQuery) HrefSubquery() *HrefQuery {
	if o.hrefQuery == nil {
		o.hrefQuery = NewHrefQuery()
	}

	return o.hrefQuery
}

func (o *LocatorQuery) Href(value ...string) *LocatorQuery {
	o.HrefSubquery().Value(value...)
	return o
}

func (o *LocatorQuery) ManifestID(value ...int64) *LocatorQuery {
	o.manifestIDs = append(o.manifestIDs, value...)
	return o
}

func (o *LocatorQuery) DigestsSubquery() *DigestQuery {
	if o.digestsQuery == nil {
		o.digestsQuery = NewDigestQuery()
	}

	return o.digestsQuery
}

func (o *LocatorQuery) Digests(updater func(*DigestQuery)) *LocatorQuery {
	updater(o.DigestsSubquery())
	return o
}

func (o *LocatorQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("manifest_id", o.manifestIDs, false, query, dialect)
}

func (o *LocatorQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Locator, error) { //nolint:dupl
	if !o.DigestsSubquery().IsEmpty() {
		o.saveIDs()

		digests, err := o.DigestsSubquery().
			OwnerType("locator").
			OwnerID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.DigestsSubquery().ownerTypes = nil
		o.DigestsSubquery().ownerIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("digests: %w", err)
		}

		o.ids = make([]int64, len(digests))
		for i, digest := range digests {
			o.ids[i] = digest.OwnerID
		}
	}

	if !o.HrefSubquery().IsEmpty() {
		o.saveIDs()

		hrefs, err := o.HrefSubquery().
			LocatorID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.HrefSubquery().locatorIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("href: %w", err)
		}

		o.ids = make([]int64, len(hrefs))
		for i, href := range hrefs {
			o.ids[i] = href.LocatorID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	return ret, err
}

func (o *LocatorQuery) IsEmpty() bool {
	return len(o.manifestIDs) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.HrefSubquery().IsEmpty() &&
		o.DigestsSubquery().IsEmpty()
}

type EntityQuery struct {
	modelQuery
	ownedQuery

	nameTypes  []string
	nameValues []string
	names      []*nameQueryEntry

	uris  []string
	roles []string
}

func NewEntityQuery() *EntityQuery {
	return &EntityQuery{}
}

func (o *EntityQuery) ID(value ...int64) *EntityQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *EntityQuery) OwnerType(value ...string) *EntityQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *EntityQuery) OwnerID(value ...int64) *EntityQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *EntityQuery) Owner(typ string, id int64) *EntityQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *EntityQuery) NameType(value ...string) *EntityQuery {
	o.nameTypes = append(o.nameTypes, value...)
	return o
}

func (o *EntityQuery) NameValue(value ...string) *EntityQuery {
	o.nameValues = append(o.nameValues, value...)
	return o
}

func (o *EntityQuery) Name(typ string, value string) *EntityQuery {
	o.names = append(o.names, &nameQueryEntry{typ, value})
	return o
}

func (o *EntityQuery) URI(value ...string) *EntityQuery {
	o.uris = append(o.uris, value...)
	return o
}

func (o *EntityQuery) Role(value ...string) *EntityQuery {
	o.roles = append(o.roles, value...)
	return o
}

func (o *EntityQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("name_type", o.nameTypes, false, query, dialect)
	addOrGroupWhereClause("name", o.nameValues, false, query, dialect)
	updateQueryWithEntries(o.names, query, dialect)

	addOrGroupWhereClause("uri", o.uris, false, query, dialect)
}

func (o *EntityQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Entity, error) {
	if len(o.roles) != 0 {
		o.saveIDs()

		var selectedIDs []int64
		err := db.NewSelect().
			Model(&selectedIDs).
			Table("roles").
			Column("entity_id").
			Distinct().
			WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				for _, role := range o.roles {
					q.WhereOr("role = ?", role)
				}

				return q
			}).
			Scan(ctx)

		if len(selectedIDs) == 0 && err == nil {
			err = ErrNoMatch
		}

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("roles: %w", err)
		}

		o.ids = append(o.ids, selectedIDs...)
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	return ret, err
}

func (o *EntityQuery) IsEmpty() bool {
	return len(o.nameTypes) == 0 &&
		len(o.nameValues) == 0 &&
		len(o.names) == 0 &&
		len(o.uris) == 0 &&
		len(o.roles) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty()
}

type ManifestQuery struct {
	ManifestCommonQuery

	digests [][]byte

	entityQuery        *EntityQuery
	dependentRIMsQuery *LocatorQuery
}

func NewManifestQuery() *ManifestQuery {
	return &ManifestQuery{}
}

func (o *ManifestQuery) ID(value ...int64) *ManifestQuery {
	o.ManifestDbID(value...)
	return o
}

func (o *ManifestQuery) ManifestDbID(value ...int64) *ManifestQuery {
	o.ManifestCommonQuery.ManifestDbID(value...)
	return o
}

func (o *ManifestQuery) ProfileType(value ...model.ProfileType) *ManifestQuery {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *ManifestQuery) ProfileValue(value ...string) *ManifestQuery {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *ManifestQuery) Profile(typ model.ProfileType, value string) *ManifestQuery {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *ManifestQuery) ProfileFromEAT(value ...*eat.Profile) *ManifestQuery {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *ManifestQuery) ManifestIDType(value ...model.TagIDType) *ManifestQuery {
	o.ManifestCommonQuery.ManifestIDType(value...)
	return o
}

func (o *ManifestQuery) ManifestIDValue(value ...string) *ManifestQuery {
	o.ManifestCommonQuery.ManifestIDValue(value...)
	return o
}

func (o *ManifestQuery) ManifestIDFromSWID(tag swid.TagID) *ManifestQuery {
	o.ManifestCommonQuery.ManifestIDFromSWID(tag)
	return o
}
func (o *ManifestQuery) ManifestID(typ model.TagIDType, value string) *ManifestQuery {
	o.ManifestCommonQuery.ManifestID(typ, value)
	return o
}

func (o *ManifestQuery) Label(value ...string) *ManifestQuery {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *ManifestQuery) Digest(value ...[]byte) *ManifestQuery {
	o.digests = append(o.digests, value...)
	return o
}

func (o *ManifestQuery) AddedBefore(value time.Time) *ManifestQuery {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *ManifestQuery) AddedAfter(value time.Time) *ManifestQuery {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *ManifestQuery) AddedBetween(lower, upper time.Time) *ManifestQuery {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *ManifestQuery) ValidBefore(value time.Time) *ManifestQuery {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *ManifestQuery) ValidAfter(value time.Time) *ManifestQuery {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *ManifestQuery) ValidBetween(lower, upper time.Time) *ManifestQuery {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *ManifestQuery) ValidOn(value time.Time) *ManifestQuery {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *ManifestQuery) EntitiesSubquery() *EntityQuery {
	if o.entityQuery == nil {
		o.entityQuery = NewEntityQuery()
	}

	return o.entityQuery
}

func (o *ManifestQuery) Entity(updater func(*EntityQuery)) *ManifestQuery {
	updater(o.EntitiesSubquery())
	return o
}

func (o *ManifestQuery) DependentRIMsSubquery() *LocatorQuery {
	if o.dependentRIMsQuery == nil {
		o.dependentRIMsQuery = NewLocatorQuery()
	}

	return o.dependentRIMsQuery
}

func (o *ManifestQuery) DependentRIMs(updater func(*LocatorQuery)) *ManifestQuery {
	updater(o.DependentRIMsSubquery())
	return o
}

func (o *ManifestQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("digest", o.digests, false, query, dialect)
}

func (o *ManifestQuery) Run(ctx context.Context, db bun.IDB) ([]*model.ManifestEntry, error) { // nolint:dupl
	if !o.EntitiesSubquery().IsEmpty() {
		o.saveManifestDbIDs()

		entities, err := o.EntitiesSubquery().
			OwnerType("manifest").
			OwnerID(o.manifestDbIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.EntitiesSubquery().ownerTypes = nil
		o.EntitiesSubquery().ownerIDs = nil

		if err != nil {
			o.restoreManifestDbIDs()
			return nil, fmt.Errorf("entities: %w", err)
		}

		o.manifestDbIDs = make([]int64, len(entities))
		for i, entity := range entities {
			o.manifestDbIDs[i] = entity.OwnerID
		}
	}

	if !o.DependentRIMsSubquery().IsEmpty() {
		o.saveManifestDbIDs()

		locators, err := o.DependentRIMsSubquery().ManifestID(o.manifestDbIDs...).Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.DependentRIMsSubquery().manifestIDs = nil

		if err != nil {
			o.restoreManifestDbIDs()
			return nil, fmt.Errorf("dependent RIMs: %w", err)
		}

		o.manifestDbIDs = make([]int64, len(locators))
		for i, locator := range locators {
			o.manifestDbIDs[i] = locator.ManifestID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreManifestDbIDs()
	return ret, err
}

func (o *ManifestQuery) IsEmpty() bool {
	return len(o.digests) == 0 &&
		o.ManifestCommonQuery.IsEmpty() &&
		o.EntitiesSubquery().IsEmpty() &&
		o.DependentRIMsSubquery().IsEmpty()
}

type LinkedTagQuery struct {
	modelQuery

	linkedTagIDTypes  []model.TagIDType
	linkedTagIDValues []string
	linkedTagIDs      []*linkedTagIDQueryEntry
	tagRelations      []model.TagRelation

	moduleIDs []int64
}

func NewLinkedTagQuery() *LinkedTagQuery {
	return &LinkedTagQuery{}
}

func (o *LinkedTagQuery) ID(value ...int64) *LinkedTagQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *LinkedTagQuery) LinkedTagIDType(value ...model.TagIDType) *LinkedTagQuery {
	o.linkedTagIDTypes = append(o.linkedTagIDTypes, value...)
	return o
}

func (o *LinkedTagQuery) LinkedTagIDValue(value ...string) *LinkedTagQuery {
	o.linkedTagIDValues = append(o.linkedTagIDValues, value...)
	return o
}

func (o *LinkedTagQuery) LinkedTagID(typ model.TagIDType, value string) *LinkedTagQuery {
	o.linkedTagIDs = append(o.linkedTagIDs, &linkedTagIDQueryEntry{typ, value})
	return o
}

func (o *LinkedTagQuery) TagRelation(value ...model.TagRelation) *LinkedTagQuery {
	o.tagRelations = append(o.tagRelations, value...)
	return o
}

func (o *LinkedTagQuery) ModuleID(value ...int64) *LinkedTagQuery {
	o.moduleIDs = append(o.moduleIDs, value...)
	return o
}

func (o *LinkedTagQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("linked_tag_id_type", o.linkedTagIDTypes, false, query, dialect)
	addOrGroupWhereClause("linked_tag_id", o.linkedTagIDValues, false, query, dialect)
	addOrGroupWhereClause("tag_relation", o.tagRelations, false, query, dialect)
	updateQueryWithEntries(o.linkedTagIDs, query, dialect)

	addOrGroupWhereClause("module_id", o.moduleIDs, false, query, dialect)
}

func (o *LinkedTagQuery) Run(ctx context.Context, db bun.IDB) ([]*model.LinkedTag, error) {
	return runQuery(ctx, db, o)
}

func (o *LinkedTagQuery) IsEmpty() bool {
	return len(o.linkedTagIDTypes) == 0 &&
		len(o.linkedTagIDValues) == 0 &&
		len(o.linkedTagIDs) == 0 &&
		len(o.tagRelations) == 0 &&
		len(o.moduleIDs) == 0 &&
		o.modelQuery.IsEmpty()
}

type ModuleTagQuery struct {
	ManifestCommonQuery
	ModuleTagCommonQuery

	entitiesQuery   *EntityQuery
	linkedTagsQuery *LinkedTagQuery
}

func NewModuleTagQuery() *ModuleTagQuery {
	return &ModuleTagQuery{}
}

func (o *ModuleTagQuery) ID(value ...int64) *ModuleTagQuery {
	o.ModuleTagDbID(value...)
	return o
}

func (o *ModuleTagQuery) ModuleTagDbID(value ...int64) *ModuleTagQuery {
	o.ModuleTagCommonQuery.ModuleTagDbID(value...)
	return o
}

func (o *ModuleTagQuery) ManifestDbID(value ...int64) *ModuleTagQuery {
	o.ManifestCommonQuery.ManifestDbID(value...)
	return o
}

func (o *ModuleTagQuery) ManifestIDType(value ...model.TagIDType) *ModuleTagQuery {
	o.ManifestCommonQuery.ManifestIDType(value...)
	return o
}

func (o *ModuleTagQuery) ManifestIDValue(value ...string) *ModuleTagQuery {
	o.ManifestCommonQuery.ManifestIDValue(value...)
	return o
}

func (o *ModuleTagQuery) ManifestIDFromSWID(tag swid.TagID) *ModuleTagQuery {
	typ, value := model.ParseSWIDTagID(tag)
	return o.ManifestID(typ, value)
}

func (o *ModuleTagQuery) ManifestID(typ model.TagIDType, value string) *ModuleTagQuery {
	o.ManifestCommonQuery.ManifestID(typ, value)
	return o
}

func (o *ModuleTagQuery) Label(value ...string) *ModuleTagQuery {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *ModuleTagQuery) ProfileType(value ...model.ProfileType) *ModuleTagQuery {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *ModuleTagQuery) ProfileValue(value ...string) *ModuleTagQuery {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *ModuleTagQuery) Profile(typ model.ProfileType, value string) *ModuleTagQuery {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *ModuleTagQuery) ProfileFromEAT(value ...*eat.Profile) *ModuleTagQuery {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *ModuleTagQuery) AddedBefore(value time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *ModuleTagQuery) AddedAfter(value time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *ModuleTagQuery) AddedBetween(lower, upper time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *ModuleTagQuery) ValidBefore(value time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *ModuleTagQuery) ValidAfter(value time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *ModuleTagQuery) ValidBetween(lower, upper time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *ModuleTagQuery) ValidOn(value time.Time) *ModuleTagQuery {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *ModuleTagQuery) ModuleTagIDType(value ...model.TagIDType) *ModuleTagQuery {
	o.ModuleTagCommonQuery.ModuleTagIDType(value...)
	return o
}

func (o *ModuleTagQuery) ModuleTagIDValue(value ...string) *ModuleTagQuery {
	o.ModuleTagCommonQuery.ModuleTagIDValue(value...)
	return o
}

func (o *ModuleTagQuery) ModuleTagIDFromSWID(tag swid.TagID) *ModuleTagQuery {
	typ, value := model.ParseSWIDTagID(tag)
	return o.ModuleTagID(typ, value)
}

func (o *ModuleTagQuery) ModuleTagID(typ model.TagIDType, value string) *ModuleTagQuery {
	o.ModuleTagCommonQuery.ModuleTagID(typ, value)
	return o
}

func (o *ModuleTagQuery) Language(value ...string) *ModuleTagQuery {
	o.ModuleTagCommonQuery.Language(value...)
	return o
}

func (o *ModuleTagQuery) ModuleTagVersion(value ...uint) *ModuleTagQuery {
	o.moduleTagVersions = append(o.moduleTagVersions, value...)
	return o
}

func (o *ModuleTagQuery) EntitiesSubquery() *EntityQuery {
	if o.entitiesQuery == nil {
		o.entitiesQuery = NewEntityQuery()
	}

	return o.entitiesQuery
}

func (o *ModuleTagQuery) Entity(updater func(*EntityQuery)) *ModuleTagQuery {
	updater(o.EntitiesSubquery())
	return o
}

func (o *ModuleTagQuery) LinkedTagsSubquery() *LinkedTagQuery {
	if o.linkedTagsQuery == nil {
		o.linkedTagsQuery = NewLinkedTagQuery()
	}

	return o.linkedTagsQuery
}

func (o *ModuleTagQuery) LinkedTag(updater func(*LinkedTagQuery)) *ModuleTagQuery {
	updater(o.LinkedTagsSubquery())
	return o
}

func (o *ModuleTagQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	o.ModuleTagCommonQuery.UpdateSelectQuery(query, dialect)
}

func (o *ModuleTagQuery) Run(ctx context.Context, db bun.IDB) ([]*model.ModuleTagEntry, error) { // nolint:dupl
	if !o.EntitiesSubquery().IsEmpty() {
		o.saveModuleTagDbIDs()

		entities, err := o.EntitiesSubquery().
			OwnerType("module_tag").
			OwnerID(o.moduleTagDbIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.EntitiesSubquery().ownerTypes = nil
		o.EntitiesSubquery().ownerIDs = nil

		if err != nil {
			o.restoreModuleTagDbIDs()
			return nil, fmt.Errorf("entities: %w", err)
		}

		o.moduleTagDbIDs = make([]int64, len(entities))
		for i, entity := range entities {
			o.moduleTagDbIDs[i] = entity.OwnerID
		}
	}

	if !o.LinkedTagsSubquery().IsEmpty() {
		o.saveModuleTagDbIDs()

		linkedTags, err := o.LinkedTagsSubquery().ModuleID(o.moduleTagDbIDs...).Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.LinkedTagsSubquery().moduleIDs = nil

		if err != nil {
			o.restoreModuleTagDbIDs()
			return nil, fmt.Errorf("linked tags: %w", err)
		}

		o.moduleTagDbIDs = make([]int64, len(linkedTags))
		for i, linedTag := range linkedTags {
			o.moduleTagDbIDs[i] = linedTag.ModuleID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreModuleTagDbIDs()
	return ret, err
}

func (o *ModuleTagQuery) IsEmpty() bool {
	return len(o.manifestIDs) == 0 &&
		o.ManifestCommonQuery.IsEmpty() &&
		o.ModuleTagCommonQuery.IsEmpty() &&
		o.LinkedTagsSubquery().IsEmpty() &&
		o.EntitiesSubquery().IsEmpty()
}

type EnvironmentQuery struct {
	modelQuery

	Exact bool

	classIDTypes []string
	classIDBytes [][]byte
	classIDs     []*classIDQueryEntry

	classSubqueries []*ClassSubquery

	vendors []string
	models  []string
	layers  []uint64
	indexes []uint64

	instanceTypes []string
	instanceBytes [][]byte
	instances     []*instanceQueryEntry

	groupTypes []string
	groupBytes [][]byte
	groups     []*groupQueryEntry
}

func NewEnvironmentQuery(exact bool) *EnvironmentQuery {
	return &EnvironmentQuery{Exact: exact}
}

func (o *EnvironmentQuery) ID(value ...int64) *EnvironmentQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *EnvironmentQuery) ClassIDType(value ...string) *EnvironmentQuery {
	o.classIDTypes = append(o.classIDTypes, value...)
	return o
}

func (o *EnvironmentQuery) ClassIDBytes(value ...[]byte) *EnvironmentQuery {
	o.classIDBytes = append(o.classIDBytes, value...)
	return o
}

func (o *EnvironmentQuery) ClassID(typ string, value []byte) *EnvironmentQuery {
	o.classIDs = append(o.classIDs, &classIDQueryEntry{typ, value})
	return o
}

func (o *EnvironmentQuery) Class(updater ...func(*ClassSubquery)) *EnvironmentQuery {
	for _, f := range updater {
		f(o.addClassSubquery())
	}
	return o
}

func (o *EnvironmentQuery) Vendor(value ...string) *EnvironmentQuery {
	o.vendors = append(o.vendors, value...)
	return o
}

func (o *EnvironmentQuery) Model(value ...string) *EnvironmentQuery {
	o.models = append(o.models, value...)
	return o
}

func (o *EnvironmentQuery) Layer(value ...uint64) *EnvironmentQuery {
	o.layers = append(o.layers, value...)
	return o
}

func (o *EnvironmentQuery) Index(value ...uint64) *EnvironmentQuery {
	o.indexes = append(o.indexes, value...)
	return o
}

func (o *EnvironmentQuery) InstanceType(value ...string) *EnvironmentQuery {
	o.instanceTypes = append(o.instanceTypes, value...)
	return o
}

func (o *EnvironmentQuery) InstanceBytes(value ...[]byte) *EnvironmentQuery {
	o.instanceBytes = append(o.instanceBytes, value...)
	return o
}

func (o *EnvironmentQuery) Instance(typ string, value []byte) *EnvironmentQuery {
	o.instances = append(o.instances, &instanceQueryEntry{typ, value})
	return o
}

func (o *EnvironmentQuery) GroupType(value ...string) *EnvironmentQuery {
	o.groupTypes = append(o.groupTypes, value...)
	return o
}

func (o *EnvironmentQuery) GroupBytes(value ...[]byte) *EnvironmentQuery {
	o.groupBytes = append(o.groupBytes, value...)
	return o
}

func (o *EnvironmentQuery) Group(typ string, value []byte) *EnvironmentQuery {
	o.groups = append(o.groups, &groupQueryEntry{typ, value})
	return o
}

func (o *EnvironmentQuery) UpdateFromModel(env *model.Environment) *EnvironmentQuery {
	if env.ClassType != nil {
		if env.ClassBytes != nil {
			o.ClassID(*env.ClassType, *env.ClassBytes)
		} else {
			o.ClassIDType(*env.ClassType)
		}
	} else if env.ClassBytes != nil {
		o.ClassIDBytes(*env.ClassBytes)
	}

	if env.Vendor != nil {
		o.Vendor(*env.Vendor)
	}

	if env.Model != nil {
		o.Model(*env.Model)
	}

	if env.Layer != nil {
		o.Layer(*env.Layer)
	}

	if env.Index != nil {
		o.Index(*env.Index)
	}

	if env.InstanceType != nil {
		if env.InstanceBytes != nil {
			o.Instance(*env.InstanceType, *env.InstanceBytes)
		} else {
			o.InstanceType(*env.InstanceType)
		}
	} else if env.InstanceBytes != nil {
		o.InstanceBytes(*env.InstanceBytes)
	}

	if env.GroupType != nil {
		if env.GroupBytes != nil {
			o.Group(*env.GroupType, *env.GroupBytes)
		} else {
			o.GroupType(*env.GroupType)
		}
	} else if env.GroupBytes != nil {
		o.GroupBytes(*env.GroupBytes)
	}

	return o
}

func (o *EnvironmentQuery) UpdateFromCoRIM(corimEnv *comid.Environment) error {
	env, err := model.NewEnvironmentFromCoRIM(corimEnv)
	if err != nil {
		// coverage:ignore
		return err
	}

	o.UpdateFromModel(env)

	return nil
}

func (o *EnvironmentQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		for _, sub := range o.classSubqueries {
			q.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				sub.UpdateSelectQuery(q, dialect, o.Exact)
				return q
			})
		}

		return q
	})

	// Override adding "<field> == NULL" for unspecified individual fields
	// to the query below when these fields are being handled by composite
	// entries (or, in case of class ID, sub-queries).
	exactClass := len(o.classSubqueries) == 0 && len(o.classIDs) == 0 && o.Exact
	exactInstance := len(o.instances) == 0 && o.Exact
	exactGroup := len(o.groups) == 0 && o.Exact

	addOrGroupWhereClause("class_type", o.classIDTypes, false, query, dialect)
	addOrGroupWhereClause("class_bytes", o.classIDBytes, exactClass, query, dialect)
	updateQueryWithEntries(o.classIDs, query, dialect)

	addOrGroupWhereClause("vendor", o.vendors, exactClass, query, dialect)
	addOrGroupWhereClause("model", o.models, exactClass, query, dialect)
	addOrGroupWhereClause("layer", o.layers, exactClass, query, dialect)
	addOrGroupWhereClause("index", o.indexes, exactClass, query, dialect)

	addOrGroupWhereClause("group_type", o.groupTypes, false, query, dialect)
	addOrGroupWhereClause("group_bytes", o.groupBytes, exactGroup, query, dialect)
	updateQueryWithEntries(o.groups, query, dialect)

	addOrGroupWhereClause("instance_type", o.instanceTypes, false, query, dialect)
	addOrGroupWhereClause("instance_bytes", o.instanceBytes, exactInstance, query, dialect)
	updateQueryWithEntries(o.instances, query, dialect)
}

func (o *EnvironmentQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Environment, error) {
	return runQuery(ctx, db, o)
}

func (o *EnvironmentQuery) IsEmpty() bool {
	return len(o.classIDTypes) == 0 &&
		len(o.classIDBytes) == 0 &&
		len(o.classIDs) == 0 &&
		len(o.classSubqueries) == 0 &&
		len(o.vendors) == 0 &&
		len(o.models) == 0 &&
		len(o.layers) == 0 &&
		len(o.indexes) == 0 &&
		len(o.instanceTypes) == 0 &&
		len(o.instanceBytes) == 0 &&
		len(o.instances) == 0 &&
		len(o.groupTypes) == 0 &&
		len(o.groupBytes) == 0 &&
		len(o.groups) == 0 &&
		o.modelQuery.IsEmpty()
}

func (o *EnvironmentQuery) addClassSubquery() *ClassSubquery {
	ret := &ClassSubquery{}
	o.classSubqueries = append(o.classSubqueries, ret)
	return ret
}

type ClassSubquery struct {
	classIDTypes []string
	classIDBytes [][]byte
	classIDs     []*classIDQueryEntry

	vendors []string
	models  []string
	layers  []uint64
	indexes []uint64
}

func (o *ClassSubquery) ClassIDType(value ...string) *ClassSubquery {
	o.classIDTypes = append(o.classIDTypes, value...)
	return o
}

func (o *ClassSubquery) ClassIDBytes(value ...[]byte) *ClassSubquery {
	o.classIDBytes = append(o.classIDBytes, value...)
	return o
}

func (o *ClassSubquery) ClassID(typ string, value []byte) *ClassSubquery {
	o.classIDs = append(o.classIDs, &classIDQueryEntry{typ, value})
	return o
}

func (o *ClassSubquery) Vendor(value ...string) *ClassSubquery {
	o.vendors = append(o.vendors, value...)
	return o
}

func (o *ClassSubquery) Model(value ...string) *ClassSubquery {
	o.models = append(o.models, value...)
	return o
}

func (o *ClassSubquery) Layer(value ...uint64) *ClassSubquery {
	o.layers = append(o.layers, value...)
	return o
}

func (o *ClassSubquery) Index(value ...uint64) *ClassSubquery {
	o.indexes = append(o.indexes, value...)
	return o
}

func (o *ClassSubquery) UpdateFromCoRIM(value *comid.Class) *ClassSubquery {
	if value.ClassID != nil {
		o.ClassID(value.ClassID.Type(), value.ClassID.Bytes())
	}

	if value.Vendor != nil {
		o.Vendor(*value.Vendor)
	}

	if value.Model != nil {
		o.Model(*value.Model)
	}

	if value.Layer != nil {
		o.Layer(*value.Layer)
	}

	if value.Index != nil {
		o.Index(*value.Index)
	}

	return o
}

func (o *ClassSubquery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect, exact bool) {
	addOrGroupWhereClause("class_type", o.classIDTypes, false, query, dialect)
	addOrGroupWhereClause("class_bytes", o.classIDBytes, exact, query, dialect)
	updateQueryWithEntries(o.classIDs, query, dialect)

	addOrGroupWhereClause("vendor", o.vendors, exact, query, dialect)
	addOrGroupWhereClause("model", o.models, exact, query, dialect)
	addOrGroupWhereClause("layer", o.layers, exact, query, dialect)
	addOrGroupWhereClause("index", o.indexes, exact, query, dialect)
}

func (o *ClassSubquery) IsEmpty() bool {
	return len(o.classIDTypes) == 0 &&
		len(o.classIDBytes) == 0 &&
		len(o.classIDs) == 0 &&
		len(o.vendors) == 0 &&
		len(o.models) == 0 &&
		len(o.layers) == 0 &&
		len(o.indexes) == 0
}

type KeyTripleQuery = TripleQuery[*model.KeyTripleEntry, model.KeyTripleType]

func NewKeyTripleQuery() *KeyTripleQuery {
	return NewTripleQuery[*model.KeyTripleEntry, model.KeyTripleType]()
}

type ValueTripleQuery = TripleQuery[*model.ValueTripleEntry, model.ValueTripleType]

func NewValueTripleQuery() *ValueTripleQuery {
	return NewTripleQuery[*model.ValueTripleEntry, model.ValueTripleType]()
}

type TripleQuery[T model.Model, TT any] struct {
	ManifestCommonQuery
	ModuleTagCommonQuery

	isActive *bool

	environmentQuery *EnvironmentQuery
	cryptoKeyQuery   *CryptoKeyQuery
	authByQuery      *CryptoKeyQuery

	measurementGroup *MeasurementQueryGroup

	tripleDbIDs    []int64
	environmentIDs []int64

	tripleTypes []TT

	savedTripleIDs []int64
	tripleSaved    bool
	savedEnvIDs    []int64
	envSaved       bool
}

func NewTripleQuery[T model.Model, TT any]() *TripleQuery[T, TT] {
	return &TripleQuery[T, TT]{}
}

func (o *TripleQuery[T, TT]) ID(value ...int64) *TripleQuery[T, TT] {
	o.TripleDbID(value...)
	return o
}

func (o *TripleQuery[T, TT]) TripleDbID(value ...int64) *TripleQuery[T, TT] {
	o.tripleDbIDs = append(o.tripleDbIDs, value...)
	return o
}

func (o *TripleQuery[T, TT]) ManifestDbID(value ...int64) *TripleQuery[T, TT] {
	o.manifestDbIDs = append(o.manifestDbIDs, value...)
	return o
}

func (o *TripleQuery[T, TT]) ModuleTagDbID(value ...int64) *TripleQuery[T, TT] {
	o.moduleTagDbIDs = append(o.moduleTagDbIDs, value...)
	return o
}

func (o *TripleQuery[T, TT]) EnvironmentID(value ...int64) *TripleQuery[T, TT] {
	o.environmentIDs = append(o.environmentIDs, value...)
	return o
}

func (o *TripleQuery[T, TT]) IsActive(value bool) *TripleQuery[T, TT] {
	o.isActive = &value
	return o
}

func (o *TripleQuery[T, TT]) TripleType(value ...TT) *TripleQuery[T, TT] {
	o.tripleTypes = append(o.tripleTypes, value...)
	return o
}

func (o *TripleQuery[T, TT]) ManifestIDType(value ...model.TagIDType) *TripleQuery[T, TT] {
	o.manifestIDTypes = append(o.manifestIDTypes, value...)
	return o
}

func (o *TripleQuery[T, TT]) ManifestIDValue(value ...string) *TripleQuery[T, TT] {
	o.manifestIDValues = append(o.manifestIDValues, value...)
	return o
}

func (o *TripleQuery[T, TT]) ManifestID(typ model.TagIDType, value string) *TripleQuery[T, TT] {
	o.manifestIDs = append(o.manifestIDs, &manifestIDQueryEntry{typ, value})
	return o
}

func (o *TripleQuery[T, TT]) ModuleTagIDType(value ...model.TagIDType) *TripleQuery[T, TT] {
	o.ModuleTagCommonQuery.ModuleTagIDType(value...)
	return o
}

func (o *TripleQuery[T, TT]) ModuleTagIDValue(value ...string) *TripleQuery[T, TT] {
	o.ModuleTagCommonQuery.ModuleTagIDValue(value...)
	return o
}

func (o *TripleQuery[T, TT]) ModuleTagID(typ model.TagIDType, value string) *TripleQuery[T, TT] {
	o.moduleTagIDs = append(o.moduleTagIDs, &moduleTagIDQueryEntry{typ, value})
	return o
}

func (o *TripleQuery[T, TT]) ModuleTagVersion(value ...uint) *TripleQuery[T, TT] {
	o.ModuleTagCommonQuery.ModuleTagVersion(value...)
	return o
}

func (o *TripleQuery[T, TT]) Language(value ...string) *TripleQuery[T, TT] {
	o.ModuleTagCommonQuery.Language(value...)
	return o
}

func (o *TripleQuery[T, TT]) Label(value ...string) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *TripleQuery[T, TT]) ProfileType(value ...model.ProfileType) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *TripleQuery[T, TT]) ProfileValue(value ...string) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *TripleQuery[T, TT]) Profile(typ model.ProfileType, value string) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *TripleQuery[T, TT]) ProfileFromEAT(value ...*eat.Profile) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *TripleQuery[T, TT]) AddedBefore(value time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *TripleQuery[T, TT]) AddedAfter(value time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *TripleQuery[T, TT]) AddedBetween(lower, upper time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *TripleQuery[T, TT]) ValidBefore(value time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *TripleQuery[T, TT]) ValidAfter(value time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *TripleQuery[T, TT]) ValidBetween(lower, upper time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *TripleQuery[T, TT]) ValidOn(value time.Time) *TripleQuery[T, TT] {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *TripleQuery[T, TT]) EnvironmentSubquery() *EnvironmentQuery {
	if o.environmentQuery == nil {
		o.environmentQuery = NewEnvironmentQuery(false)
	}

	return o.environmentQuery
}

func (o *TripleQuery[T, TT]) ExactEnvironment(value bool) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().Exact = value
	return o
}

func (o *TripleQuery[T, TT]) Environment(updater func(e *EnvironmentQuery)) *TripleQuery[T, TT] {
	updater(o.EnvironmentSubquery())
	return o
}

func (o *TripleQuery[T, TT]) MeasurementGroup() *MeasurementQueryGroup {
	if o.measurementGroup == nil {
		o.measurementGroup = NewMeasurementQueryGroup("value_triple")
	}

	return o.measurementGroup
}

func (o *TripleQuery[T, TT]) Measurement(updater func(e *MeasurementQuery)) *TripleQuery[T, TT] {
	mq := NewMeasurementQuery().OwnerType("value_triple")
	updater(mq)
	o.MeasurementGroup().Add(mq)
	return o
}

func (o *TripleQuery[T, TT]) CryptoKeysSubquery() *CryptoKeyQuery {
	if o.cryptoKeyQuery == nil {
		o.cryptoKeyQuery = NewCryptoKeyQuery()
	}

	return o.cryptoKeyQuery
}

func (o *TripleQuery[T, TT]) CryptoKey(updater func(e *CryptoKeyQuery)) *TripleQuery[T, TT] {
	updater(o.CryptoKeysSubquery())
	return o
}

func (o *TripleQuery[T, TT]) AuthorizedBySubquery() *CryptoKeyQuery {
	if o.authByQuery == nil {
		o.authByQuery = NewCryptoKeyQuery()
	}

	return o.authByQuery
}

func (o *TripleQuery[T, TT]) AuthorizedBy(updater func(e *CryptoKeyQuery)) *TripleQuery[T, TT] {
	updater(o.AuthorizedBySubquery())
	return o
}

func (o *TripleQuery[T, TT]) ClassType(value string) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().ClassIDType(value)
	return o
}

func (o *TripleQuery[T, TT]) ClassBytes(value ...[]byte) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().ClassIDBytes(value...)
	return o
}

func (o *TripleQuery[T, TT]) Vendor(value ...string) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().Vendor(value...)
	return o
}

func (o *TripleQuery[T, TT]) Model(value ...string) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().Model(value...)
	return o
}

func (o *TripleQuery[T, TT]) Layer(value ...uint64) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().Layer(value...)
	return o
}

func (o *TripleQuery[T, TT]) Index(value ...uint64) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().Index(value...)
	return o
}

func (o *TripleQuery[T, TT]) InstanceType(value string) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().InstanceType(value)
	return o
}

func (o *TripleQuery[T, TT]) InstanceBytes(value ...[]byte) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().InstanceBytes(value...)
	return o
}

func (o *TripleQuery[T, TT]) GroupType(value string) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().GroupType(value)
	return o
}

func (o *TripleQuery[T, TT]) GroupBytes(value ...[]byte) *TripleQuery[T, TT] {
	o.EnvironmentSubquery().GroupBytes(value...)
	return o
}

func (o *TripleQuery[T, TT]) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	o.ModuleTagCommonQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("environment_db_id", o.environmentIDs, false, query, dialect)
	addOrGroupWhereClause("triple_db_id", o.tripleDbIDs, false, query, dialect)
	addOrGroupWhereClause("triple_type", o.tripleTypes, false, query, dialect)

	if o.isActive != nil {
		query.Where("is_active = ?", *o.isActive)
	}
}

func (o *TripleQuery[T, TT]) Run(ctx context.Context, db bun.IDB) ([]T, error) {
	if !o.EnvironmentSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EnvironmentSubquery().ids)
		environments, err := o.EnvironmentSubquery().ID(o.environmentIDs...).Run(ctx, db)
		o.EnvironmentSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("environment: %w", err)
		}

		o.saveEnvIDs()
		o.environmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.environmentIDs[i] = env.ID
		}
	}

	if !o.CryptoKeysSubquery().IsEmpty() {
		o.saveTripleIDs()
		cryptoKeys, err := o.CryptoKeysSubquery().
			OwnerType("key_triple").
			OwnerID(o.tripleDbIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.CryptoKeysSubquery().ownerTypes = nil
		o.CryptoKeysSubquery().ownerIDs = nil

		if err != nil {
			o.restoreTripleIDs()
			o.restoreEnvIDs()
			return nil, fmt.Errorf("crypto keys: %w", err)
		}

		o.tripleDbIDs = make([]int64, len(cryptoKeys))
		for i, ck := range cryptoKeys {
			o.tripleDbIDs[i] = ck.OwnerID
		}
	}

	if !o.AuthorizedBySubquery().IsEmpty() {
		o.saveTripleIDs()
		cryptoKeys, err := o.AuthorizedBySubquery().
			OwnerType("key_triple_auth").
			OwnerID(o.tripleDbIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.AuthorizedBySubquery().ownerTypes = nil
		o.AuthorizedBySubquery().ownerIDs = nil

		if err != nil {
			o.restoreTripleIDs()
			o.restoreEnvIDs()
			return nil, fmt.Errorf("auth by: %w", err)
		}

		o.tripleDbIDs = make([]int64, len(cryptoKeys))
		for i, ck := range cryptoKeys {
			o.tripleDbIDs[i] = ck.OwnerID
		}
	}

	if !o.MeasurementGroup().IsEmpty() {
		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.tripleDbIDs...)
		})

		tripleIDs, err := o.MeasurementGroup().RunOwnerConjunction(ctx, db)

		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.ownerIDs = nil
		})

		if err != nil {
			o.restoreEnvIDs()
			o.restoreTripleIDs()

			return nil, fmt.Errorf("measurements: %w", err)
		}

		o.saveTripleIDs()
		o.tripleDbIDs = tripleIDs
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreTripleIDs()
	o.restoreEnvIDs()
	return ret, err
}

func (o *TripleQuery[T, TT]) IsEmpty() bool {
	return o.isActive == nil &&
		len(o.tripleDbIDs) == 0 &&
		len(o.manifestDbIDs) == 0 &&
		len(o.moduleTagDbIDs) == 0 &&
		len(o.environmentIDs) == 0 &&
		len(o.tripleTypes) == 0 &&
		len(o.manifestIDTypes) == 0 &&
		len(o.manifestIDValues) == 0 &&
		len(o.manifestIDs) == 0 &&
		o.ManifestCommonQuery.IsEmpty() &&
		o.ModuleTagCommonQuery.IsEmpty() &&
		o.CryptoKeysSubquery().IsEmpty() &&
		o.AuthorizedBySubquery().IsEmpty() &&
		o.MeasurementGroup().IsEmpty() &&
		o.EnvironmentSubquery().IsEmpty()
}

func (o *TripleQuery[T, TT]) saveTripleIDs() {
	if o.tripleSaved {
		return
	}

	o.savedTripleIDs = o.tripleDbIDs
	o.tripleSaved = true
}

func (o *TripleQuery[T, TT]) restoreTripleIDs() {
	if !o.tripleSaved {
		return
	}

	o.tripleDbIDs = o.savedTripleIDs
	o.tripleSaved = false
}

func (o *TripleQuery[T, TT]) saveEnvIDs() {
	if o.envSaved {
		// coverage:ignore
		return
	}

	o.savedEnvIDs = o.environmentIDs
	o.environmentIDs = slices.Clone(o.savedEnvIDs)
	o.envSaved = true
}

func (o *TripleQuery[T, TT]) restoreEnvIDs() {
	if !o.envSaved {
		return
	}

	o.environmentIDs = o.savedEnvIDs
	o.envSaved = false
}

type DomainEntryQuery struct {
	modelQuery
	ownedQuery

	environmentQuery *EnvironmentQuery

	environmentIDs []int64

	savedEnvIDs []int64
	envSaved    bool
}

func (o *DomainEntryQuery) ID(value ...int64) *DomainEntryQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *DomainEntryQuery) OwnerType(value ...string) *DomainEntryQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *DomainEntryQuery) OwnerID(value ...int64) *DomainEntryQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *DomainEntryQuery) Owner(typ string, id int64) *DomainEntryQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *DomainEntryQuery) OwnerFromModel(value ...*model.DomainEntry) *DomainEntryQuery {
	for _, entry := range value {
		o.Owner(entry.OwnerType, entry.OwnerID)
	}
	return o
}

func (o *DomainEntryQuery) EnvironmentSubquery() *EnvironmentQuery {
	if o.environmentQuery == nil {
		o.environmentQuery = NewEnvironmentQuery(false)
	}

	return o.environmentQuery
}

func (o *DomainEntryQuery) ExactEnvironment(value bool) *DomainEntryQuery {
	o.EnvironmentSubquery().Exact = value
	return o
}

func (o *DomainEntryQuery) Environment(updater func(e *EnvironmentQuery)) *DomainEntryQuery {
	updater(o.EnvironmentSubquery())
	return o
}

func (o *DomainEntryQuery) EnvironmentID(value ...int64) *DomainEntryQuery {
	o.environmentIDs = append(o.environmentIDs, value...)
	return o
}

func (o *DomainEntryQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("environment_id", o.environmentIDs, false, query, dialect)
}

func (o *DomainEntryQuery) Run(ctx context.Context, db bun.IDB) ([]*model.DomainEntry, error) {
	if !o.EnvironmentSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EnvironmentSubquery().ids)
		environments, err := o.EnvironmentSubquery().ID(o.environmentIDs...).Run(ctx, db)
		o.EnvironmentSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("environment: %w", err)
		}

		o.saveEnvIDs()
		o.environmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.environmentIDs[i] = env.ID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	o.restoreEnvIDs()
	return ret, err
}

func (o *DomainEntryQuery) IsEmpty() bool {
	return len(o.environmentIDs) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty() &&
		o.EnvironmentSubquery().IsEmpty()
}

func (o *DomainEntryQuery) saveEnvIDs() {
	if o.envSaved {
		// coverage:ignore
		return
	}

	o.savedEnvIDs = o.environmentIDs
	o.environmentIDs = slices.Clone(o.savedEnvIDs)
	o.envSaved = true
}

func (o *DomainEntryQuery) restoreEnvIDs() {
	if !o.envSaved {
		return
	}

	o.environmentIDs = o.savedEnvIDs
	o.envSaved = false
}

func NewDomainEntryQuery() *DomainEntryQuery {
	return &DomainEntryQuery{}
}

type DomainDependencyTripleQuery = DomainTripleQuery[*model.DomainDependencyTripleEntry]

func NewDomainDependencyTripleQuery() *DomainTripleQuery[*model.DomainDependencyTripleEntry] {
	return NewDomainTripleQuery[*model.DomainDependencyTripleEntry]("domain_dependency_triple")
}

type DomainMembershipTripleQuery = DomainTripleQuery[*model.DomainMembershipTripleEntry]

func NewDomainMembershipTripleQuery() *DomainTripleQuery[*model.DomainMembershipTripleEntry] {
	return NewDomainTripleQuery[*model.DomainMembershipTripleEntry]("domain_membership_triple")
}

type DomainTripleQuery[T model.Model] struct {
	ManifestCommonQuery
	ModuleTagCommonQuery

	isActive *bool

	domainIDQuery *EnvironmentQuery
	entryQuery    *DomainEntryQuery
	entryOwner    string

	tripleDbIDs            []int64
	domainIDEnvironmentIDs []int64
	entryIDs               []int64

	savedTripleIDs      []int64
	tripleSaved         bool
	savedDomainIDEnvIDs []int64
	domainIDEnvSaved    bool
}

func NewDomainTripleQuery[T model.Model](memberColumn string) *DomainTripleQuery[T] {
	return &DomainTripleQuery[T]{entryOwner: memberColumn}
}

func (o *DomainTripleQuery[T]) ID(value ...int64) *DomainTripleQuery[T] {
	o.TripleDbID(value...)
	return o
}

func (o *DomainTripleQuery[T]) TripleDbID(value ...int64) *DomainTripleQuery[T] {
	o.tripleDbIDs = append(o.tripleDbIDs, value...)
	return o
}

func (o *DomainTripleQuery[T]) ManifestDbID(value ...int64) *DomainTripleQuery[T] {
	o.manifestDbIDs = append(o.manifestDbIDs, value...)
	return o
}

func (o *DomainTripleQuery[T]) ModuleTagDbID(value ...int64) *DomainTripleQuery[T] {
	o.moduleTagDbIDs = append(o.moduleTagDbIDs, value...)
	return o
}

func (o *DomainTripleQuery[T]) IsActive(value bool) *DomainTripleQuery[T] {
	o.isActive = &value
	return o
}

func (o *DomainTripleQuery[T]) ManifestIDType(value ...model.TagIDType) *DomainTripleQuery[T] {
	o.manifestIDTypes = append(o.manifestIDTypes, value...)
	return o
}

func (o *DomainTripleQuery[T]) ManifestIDValue(value ...string) *DomainTripleQuery[T] {
	o.manifestIDValues = append(o.manifestIDValues, value...)
	return o
}

func (o *DomainTripleQuery[T]) ManifestID(typ model.TagIDType, value string) *DomainTripleQuery[T] {
	o.manifestIDs = append(o.manifestIDs, &manifestIDQueryEntry{typ, value})
	return o
}

func (o *DomainTripleQuery[T]) ModuleTagIDType(value ...model.TagIDType) *DomainTripleQuery[T] {
	o.ModuleTagCommonQuery.ModuleTagIDType(value...)
	return o
}

func (o *DomainTripleQuery[T]) ModuleTagIDValue(value ...string) *DomainTripleQuery[T] {
	o.ModuleTagCommonQuery.ModuleTagIDValue(value...)
	return o
}

func (o *DomainTripleQuery[T]) ModuleTagID(typ model.TagIDType, value string) *DomainTripleQuery[T] {
	o.moduleTagIDs = append(o.moduleTagIDs, &moduleTagIDQueryEntry{typ, value})
	return o
}

func (o *DomainTripleQuery[T]) ModuleTagVersion(value ...uint) *DomainTripleQuery[T] {
	o.ModuleTagCommonQuery.ModuleTagVersion(value...)
	return o
}

func (o *DomainTripleQuery[T]) Language(value ...string) *DomainTripleQuery[T] {
	o.ModuleTagCommonQuery.Language(value...)
	return o
}

func (o *DomainTripleQuery[T]) Label(value ...string) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *DomainTripleQuery[T]) ProfileType(value ...model.ProfileType) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *DomainTripleQuery[T]) ProfileValue(value ...string) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *DomainTripleQuery[T]) Profile(typ model.ProfileType, value string) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *DomainTripleQuery[T]) ProfileFromEAT(value ...*eat.Profile) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *DomainTripleQuery[T]) AddedBefore(value time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *DomainTripleQuery[T]) AddedAfter(value time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *DomainTripleQuery[T]) AddedBetween(lower, upper time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *DomainTripleQuery[T]) ValidBefore(value time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *DomainTripleQuery[T]) ValidAfter(value time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *DomainTripleQuery[T]) ValidBetween(lower, upper time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *DomainTripleQuery[T]) ValidOn(value time.Time) *DomainTripleQuery[T] {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *DomainTripleQuery[T]) DomainIDSubquery() *EnvironmentQuery {
	if o.domainIDQuery == nil {
		o.domainIDQuery = NewEnvironmentQuery(false)
	}

	return o.domainIDQuery
}

func (o *DomainTripleQuery[T]) ExactDomainID(value bool) *DomainTripleQuery[T] {
	o.DomainIDSubquery().Exact = value
	return o
}

func (o *DomainTripleQuery[T]) DomainID(updater func(e *EnvironmentQuery)) *DomainTripleQuery[T] {
	updater(o.DomainIDSubquery())
	return o
}

func (o *DomainTripleQuery[T]) DomainIDEnvironmentID(value ...int64) *DomainTripleQuery[T] {
	o.domainIDEnvironmentIDs = append(o.domainIDEnvironmentIDs, value...)
	return o
}

func (o *DomainTripleQuery[T]) EntrySubquery() *DomainEntryQuery {
	if o.entryQuery == nil {
		o.entryQuery = NewDomainEntryQuery()
	}

	return o.entryQuery
}

func (o *DomainTripleQuery[T]) EntryExactEnvironment(value bool) *DomainTripleQuery[T] {
	o.EntrySubquery().ExactEnvironment(value)
	return o
}

func (o *DomainTripleQuery[T]) EntryEnvironment(updater func(e *EnvironmentQuery)) *DomainTripleQuery[T] {
	o.EntrySubquery().Environment(updater)
	return o
}

func (o *DomainTripleQuery[T]) EntryEnvironmentID(value ...int64) *DomainTripleQuery[T] {
	o.EntrySubquery().EnvironmentID(value...)
	return o
}

func (o *DomainTripleQuery[T]) MemberExactEnvironment(value bool) *DomainTripleQuery[T] {
	o.EntrySubquery().ExactEnvironment(value)
	return o
}

func (o *DomainTripleQuery[T]) MemberEnvironment(updater func(e *EnvironmentQuery)) *DomainTripleQuery[T] {
	o.EntrySubquery().Environment(updater)
	return o
}

func (o *DomainTripleQuery[T]) MemberEnvironmentID(value ...int64) *DomainTripleQuery[T] {
	o.EntrySubquery().EnvironmentID(value...)
	return o
}

func (o *DomainTripleQuery[T]) TrusteeExactEnvironment(value bool) *DomainTripleQuery[T] {
	o.EntrySubquery().ExactEnvironment(value)
	return o
}

func (o *DomainTripleQuery[T]) TrusteeEnvironment(updater func(e *EnvironmentQuery)) *DomainTripleQuery[T] {
	o.EntrySubquery().Environment(updater)
	return o
}

func (o *DomainTripleQuery[T]) TrusteeEnvironmentID(value ...int64) *DomainTripleQuery[T] {
	o.EntrySubquery().EnvironmentID(value...)
	return o
}

func (o *DomainTripleQuery[T]) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	o.ModuleTagCommonQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("triple_db_id", o.tripleDbIDs, false, query, dialect)
	addOrGroupWhereClause("manifest_db_id", o.manifestDbIDs, false, query, dialect)
	addOrGroupWhereClause("module_tag_db_id", o.moduleTagDbIDs, false, query, dialect)
	addOrGroupWhereClause("environment_db_id", o.domainIDEnvironmentIDs, false, query, dialect)

	addOrGroupWhereClause("manifest_id_type", o.manifestIDTypes, false, query, dialect)
	addOrGroupWhereClause("manifest_id", o.manifestIDValues, false, query, dialect)
	updateQueryWithEntries(o.manifestIDs, query, dialect)

	if o.isActive != nil {
		query.Where("is_active = ?", *o.isActive)
	}
}

func (o *DomainTripleQuery[T]) Run(ctx context.Context, db bun.IDB) ([]T, error) {
	if !o.DomainIDSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.DomainIDSubquery().ids)
		environments, err := o.DomainIDSubquery().ID(o.domainIDEnvironmentIDs...).Run(ctx, db)
		o.DomainIDSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("domain ID: %w", err)
		}

		o.saveDomainIDEnvIDs()
		o.domainIDEnvironmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.domainIDEnvironmentIDs[i] = env.ID
		}
	}

	if !o.EntrySubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EntrySubquery().ids)
		entries, err := o.EntrySubquery().
			OwnerType(o.entryOwner).
			OwnerID(o.tripleDbIDs...).
			ID(o.entryIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.EntrySubquery().ids = savedQueryIDs
		o.EntrySubquery().ownerTypes = nil
		o.EntrySubquery().ownerIDs = nil

		if err != nil {
			o.restoreDomainIDEnvIDs()
			return nil, fmt.Errorf("entries: %w", err)
		}

		o.saveTripleIDs()
		for _, entry := range entries {
			o.tripleDbIDs = append(o.tripleDbIDs, entry.OwnerID)
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreTripleIDs()
	o.restoreDomainIDEnvIDs()
	return ret, err
}

func (o *DomainTripleQuery[T]) IsEmpty() bool {
	return o.isActive == nil &&
		len(o.tripleDbIDs) == 0 &&
		len(o.manifestDbIDs) == 0 &&
		len(o.moduleTagDbIDs) == 0 &&
		len(o.domainIDEnvironmentIDs) == 0 &&
		len(o.manifestIDTypes) == 0 &&
		len(o.manifestIDValues) == 0 &&
		len(o.manifestIDs) == 0 &&
		o.ManifestCommonQuery.IsEmpty() &&
		o.ModuleTagCommonQuery.IsEmpty() &&
		o.DomainIDSubquery().IsEmpty() &&
		o.EntrySubquery().IsEmpty()
}

func (o *DomainTripleQuery[T]) saveTripleIDs() {
	if o.tripleSaved {
		// coverage:ignore
		return
	}

	o.savedTripleIDs = o.tripleDbIDs
	o.tripleSaved = true
}

func (o *DomainTripleQuery[T]) restoreTripleIDs() {
	if !o.tripleSaved {
		return
	}

	o.tripleDbIDs = o.savedTripleIDs
	o.tripleSaved = false
}

func (o *DomainTripleQuery[T]) saveDomainIDEnvIDs() {
	if o.domainIDEnvSaved {
		// coverage:ignore
		return
	}

	o.savedDomainIDEnvIDs = o.domainIDEnvironmentIDs
	o.domainIDEnvironmentIDs = slices.Clone(o.savedDomainIDEnvIDs)
	o.domainIDEnvSaved = true
}

func (o *DomainTripleQuery[T]) restoreDomainIDEnvIDs() {
	if !o.domainIDEnvSaved {
		return
	}

	o.domainIDEnvironmentIDs = o.savedDomainIDEnvIDs
	o.domainIDEnvSaved = false
}

type StatefulEnvironmentQuery struct {
	modelQuery

	environemntQuery *EnvironmentQuery
	measurementGroup *MeasurementQueryGroup

	environmentIDs []int64
	tripleIDs      []int64

	savedEnvIDs []int64
	envSaved    bool
}

func NewStatefulEnvironmentQuery() *StatefulEnvironmentQuery {
	return &StatefulEnvironmentQuery{}
}

func (o *StatefulEnvironmentQuery) ID(value ...int64) *StatefulEnvironmentQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *StatefulEnvironmentQuery) TripleID(value ...int64) *StatefulEnvironmentQuery {
	o.tripleIDs = append(o.tripleIDs, value...)
	return o
}

func (o *StatefulEnvironmentQuery) EnvironmentSubquery() *EnvironmentQuery {
	if o.environemntQuery == nil {
		o.environemntQuery = NewEnvironmentQuery(false)
	}

	return o.environemntQuery
}

func (o *StatefulEnvironmentQuery) EnvironmentID(value ...int64) *StatefulEnvironmentQuery {
	o.environmentIDs = append(o.environmentIDs, value...)
	return o
}

func (o *StatefulEnvironmentQuery) ExactEnvironment(value bool) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Exact = value
	return o
}

func (o *StatefulEnvironmentQuery) ClassIDType(value ...string) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().ClassIDType(value...)
	return o
}

func (o *StatefulEnvironmentQuery) ClassIDBytes(value ...[]byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().ClassIDBytes(value...)
	return o
}

func (o *StatefulEnvironmentQuery) ClassID(typ string, value []byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().ClassID(typ, value)
	return o
}

func (o *StatefulEnvironmentQuery) Class(updater ...func(*ClassSubquery)) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Class(updater...)
	return o
}

func (o *StatefulEnvironmentQuery) Vendor(value ...string) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Vendor(value...)
	return o
}

func (o *StatefulEnvironmentQuery) Model(value ...string) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Model(value...)
	return o
}

func (o *StatefulEnvironmentQuery) Layer(value ...uint64) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Layer(value...)
	return o
}

func (o *StatefulEnvironmentQuery) Index(value ...uint64) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Index(value...)
	return o
}

func (o *StatefulEnvironmentQuery) InstanceType(value ...string) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().InstanceType(value...)
	return o
}

func (o *StatefulEnvironmentQuery) InstanceBytes(value ...[]byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().InstanceBytes(value...)
	return o
}

func (o *StatefulEnvironmentQuery) Instance(typ string, value []byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Instance(typ, value)
	return o
}

func (o *StatefulEnvironmentQuery) GroupType(value ...string) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().GroupType(value...)
	return o
}

func (o *StatefulEnvironmentQuery) GroupBytes(value ...[]byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().GroupBytes(value...)
	return o
}

func (o *StatefulEnvironmentQuery) Group(typ string, value []byte) *StatefulEnvironmentQuery {
	o.EnvironmentSubquery().Group(typ, value)
	return o
}

func (o *StatefulEnvironmentQuery) MeasurementGroup() *MeasurementQueryGroup {
	if o.measurementGroup == nil {
		o.measurementGroup = NewMeasurementQueryGroup("stateful_environment")
	}

	return o.measurementGroup
}

func (o *StatefulEnvironmentQuery) Measurement(updater func(e *MeasurementQuery)) *StatefulEnvironmentQuery {
	mq := NewMeasurementQuery().OwnerType("stateful_environment")
	updater(mq)
	o.MeasurementGroup().Add(mq)
	return o
}

func (o *StatefulEnvironmentQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("environment_id", o.environmentIDs, false, query, dialect)
	addOrGroupWhereClause("triple_id", o.tripleIDs, false, query, dialect)
}

func (o *StatefulEnvironmentQuery) Run(ctx context.Context, db bun.IDB) ([]*model.StatefulEnvironment, error) {
	if !o.EnvironmentSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EnvironmentSubquery().ids)
		environments, err := o.EnvironmentSubquery().ID(o.environmentIDs...).Run(ctx, db)
		o.EnvironmentSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("environment: %w", err)
		}

		o.saveEnvIDs()
		o.environmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.environmentIDs[i] = env.ID
		}
	}

	if !o.MeasurementGroup().IsEmpty() {
		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.ids...)
		})

		ids, err := o.MeasurementGroup().RunOwnerConjunction(ctx, db)

		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.ownerIDs = nil
		})

		if err != nil {
			o.restoreEnvIDs()

			return nil, fmt.Errorf("measurements: %w", err)
		}

		o.modelQuery.saveIDs()
		o.modelQuery.ids = ids
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreEnvIDs()
	o.modelQuery.restoreIDs()
	return ret, err
}

func (o *StatefulEnvironmentQuery) IsEmpty() bool {
	return o.modelQuery.IsEmpty() &&
		len(o.environmentIDs) == 0 &&
		len(o.tripleIDs) == 0 &&
		o.EnvironmentSubquery().IsEmpty() &&
		o.MeasurementGroup().IsEmpty()
}

func (o *StatefulEnvironmentQuery) saveEnvIDs() {
	if o.envSaved {
		// coverage:ignore
		return
	}

	o.savedEnvIDs = o.environmentIDs
	o.environmentIDs = slices.Clone(o.savedEnvIDs)
	o.envSaved = true
}

func (o *StatefulEnvironmentQuery) restoreEnvIDs() {
	if !o.envSaved {
		return
	}

	o.environmentIDs = o.savedEnvIDs
	o.envSaved = false
}

type EndorsementQuery struct {
	modelQuery
	ownedQuery

	environmentIDs []int64

	environmentQuery EnvironmentQuery
	measurements     *MeasurementQueryGroup

	savedEnvIDs []int64
	envSaved    bool
}

func NewEndorsementQuery() *EndorsementQuery {
	return &EndorsementQuery{}
}

func (o *EndorsementQuery) ID(value ...int64) *EndorsementQuery {
	o.modelQuery.ID(value...)
	return o
}
func (o *EndorsementQuery) OwnerType(value ...string) *EndorsementQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *EndorsementQuery) OwnerID(value ...int64) *EndorsementQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *EndorsementQuery) Owner(typ string, id int64) *EndorsementQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *EndorsementQuery) EnvironmentID(value ...int64) *EndorsementQuery {
	o.environmentIDs = append(o.environmentIDs, value...)
	return o
}

func (o *EndorsementQuery) Environment(updater func(*EnvironmentQuery)) *EndorsementQuery {
	updater(&o.environmentQuery)
	return o
}

func (o *EndorsementQuery) EnvironmentSubquery() *EnvironmentQuery {
	return &o.environmentQuery
}

func (o *EndorsementQuery) Measurement(updater func(*MeasurementQuery)) *EndorsementQuery {
	query := NewMeasurementQuery().OwnerType("value_triple")
	updater(query)
	o.MeasurementGroup().Add(query)
	return o
}

func (o *EndorsementQuery) MeasurementGroup() *MeasurementQueryGroup {
	if o.measurements == nil {
		o.measurements = NewMeasurementQueryGroup("value_triple")
	}

	return o.measurements
}

func (o *EndorsementQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("environment_id", o.environmentIDs, false, query, dialect)
}

func (o *EndorsementQuery) Run(ctx context.Context, db bun.IDB) ([]*model.ValueTriple, error) {
	if !o.EnvironmentSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EnvironmentSubquery().ids)
		environments, err := o.EnvironmentSubquery().ID(o.environmentIDs...).Run(ctx, db)
		o.EnvironmentSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("environment: %w", err)
		}

		o.saveEnvIDs()
		o.environmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.environmentIDs[i] = env.ID
		}
	}

	if !o.MeasurementGroup().IsEmpty() {

		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.modelQuery.ids...)
		})

		ids, err := o.MeasurementGroup().RunOwnerConjunction(ctx, db)

		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.ownedQuery.ownerIDs = nil
		})

		if err != nil {
			return nil, fmt.Errorf("measurements: %w", err)
		}

		o.modelQuery.saveIDs()
		o.modelQuery.ids = ids
	}

	ret, err := runQuery(ctx, db, o)
	o.modelQuery.restoreIDs()
	o.restoreEnvIDs()
	return ret, err
}

func (o *EndorsementQuery) IsEmpty() bool {
	return len(o.environmentIDs) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty() &&
		o.environmentQuery.IsEmpty() &&
		o.MeasurementGroup().IsEmpty()
}

func (o *EndorsementQuery) saveEnvIDs() {
	if o.envSaved {
		// coverage:ignore
		return
	}

	o.savedEnvIDs = o.environmentIDs
	o.environmentIDs = slices.Clone(o.savedEnvIDs)
	o.envSaved = true
}

func (o *EndorsementQuery) restoreEnvIDs() {
	if !o.envSaved {
		return
	}

	o.environmentIDs = o.savedEnvIDs
	o.envSaved = false
}

type ConditionalEndorsementTripleQuery struct {
	ManifestCommonQuery
	ModuleTagCommonQuery

	isActive *bool

	tripleDbIDs []int64

	conditions   *StatefulEnvironmentQueryGroup
	endorsements *EndorsementQueryGroup

	savedTripleIDs []int64
	tripleSaved    bool
}

func NewConditionalEndorsementTripleQuery() *ConditionalEndorsementTripleQuery {
	return &ConditionalEndorsementTripleQuery{}
}

func (o *ConditionalEndorsementTripleQuery) TripleDbID(value ...int64) *ConditionalEndorsementTripleQuery {
	o.tripleDbIDs = append(o.tripleDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ManifestDbID(value ...int64) *ConditionalEndorsementTripleQuery {
	o.manifestDbIDs = append(o.manifestDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ModuleTagDbID(value ...int64) *ConditionalEndorsementTripleQuery {
	o.moduleTagDbIDs = append(o.moduleTagDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) IsActive(value bool) *ConditionalEndorsementTripleQuery {
	o.isActive = &value
	return o
}

func (o *ConditionalEndorsementTripleQuery) ManifestIDType(
	value ...model.TagIDType,
) *ConditionalEndorsementTripleQuery {
	o.manifestIDTypes = append(o.manifestIDTypes, value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ManifestIDValue(value ...string) *ConditionalEndorsementTripleQuery {
	o.manifestIDValues = append(o.manifestIDValues, value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ManifestID(
	typ model.TagIDType, value string,
) *ConditionalEndorsementTripleQuery {
	o.manifestIDs = append(o.manifestIDs, &manifestIDQueryEntry{typ, value})
	return o
}

func (o *ConditionalEndorsementTripleQuery) ModuleTagIDType(
	value ...model.TagIDType,
) *ConditionalEndorsementTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagIDType(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ModuleTagIDValue(
	value ...string,
) *ConditionalEndorsementTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagIDValue(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ModuleTagID(
	typ model.TagIDType, value string,
) *ConditionalEndorsementTripleQuery {
	o.moduleTagIDs = append(o.moduleTagIDs, &moduleTagIDQueryEntry{typ, value})
	return o
}

func (o *ConditionalEndorsementTripleQuery) ModuleTagVersion(value ...uint) *ConditionalEndorsementTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagVersion(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) Language(value ...string) *ConditionalEndorsementTripleQuery {
	o.ModuleTagCommonQuery.Language(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) Label(value ...string) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ProfileType(
	value ...model.ProfileType,
) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ProfileValue(value ...string) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) Profile(
	typ model.ProfileType, value string,
) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ProfileFromEAT(
	value ...*eat.Profile,
) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *ConditionalEndorsementTripleQuery) AddedBefore(value time.Time) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) AddedAfter(value time.Time) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) AddedBetween(
	lower, upper time.Time,
) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ValidBefore(value time.Time) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ValidAfter(value time.Time) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ValidBetween(
	lower, upper time.Time,
) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ValidOn(value time.Time) *ConditionalEndorsementTripleQuery {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *ConditionalEndorsementTripleQuery) ConditionGroup() *StatefulEnvironmentQueryGroup {
	if o.conditions == nil {
		o.conditions = NewStatefulEnvironmentQueryGroup()
	}

	return o.conditions
}

func (o *ConditionalEndorsementTripleQuery) Condition(
	updater func(*StatefulEnvironmentQuery),
) *ConditionalEndorsementTripleQuery {
	query := NewStatefulEnvironmentQuery()
	updater(query)
	o.ConditionGroup().Add(query)
	return o
}

func (o *ConditionalEndorsementTripleQuery) EndorsementGroup() *EndorsementQueryGroup {
	if o.endorsements == nil {
		o.endorsements = NewEndorsementQueryGroup()
	}

	return o.endorsements
}

func (o *ConditionalEndorsementTripleQuery) Endorsement(
	updater func(*EndorsementQuery),
) *ConditionalEndorsementTripleQuery {
	query := NewEndorsementQuery().OwnerType("conditional_endorsement_triple")
	updater(query)
	o.EndorsementGroup().Add(query)
	return o
}

func (o *ConditionalEndorsementTripleQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	o.ModuleTagCommonQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("triple_db_id", o.tripleDbIDs, false, query, dialect)

	if o.isActive != nil {
		query.Where("is_active = ?", *o.isActive)
	}
}

func (o *ConditionalEndorsementTripleQuery) Run(
	ctx context.Context,
	db bun.IDB,
) ([]*model.ConditionalEndorsementTripleEntry, error) {
	if !o.ConditionGroup().IsEmpty() {
		o.ConditionGroup().ForEach(func(q *StatefulEnvironmentQuery) {
			q.TripleID(o.tripleDbIDs...)
		})

		tripleIDs, err := o.ConditionGroup().RunOwnerConjunction(ctx, db)

		o.ConditionGroup().ForEach(func(q *StatefulEnvironmentQuery) {
			q.tripleIDs = nil
		})

		if err != nil {
			return nil, fmt.Errorf("conditions: %w", err)
		}

		o.saveTripleIDs()
		o.tripleDbIDs = tripleIDs

	}

	if !o.EndorsementGroup().IsEmpty() {
		o.EndorsementGroup().ForEach(func(q *EndorsementQuery) {
			q.OwnerID(o.tripleDbIDs...)
		})

		tripleIDs, err := o.EndorsementGroup().RunOwnerConjunction(ctx, db)

		o.EndorsementGroup().ForEach(func(q *EndorsementQuery) {
			q.ownedQuery.ownerIDs = nil
		})

		if err != nil {
			o.restoreTripleIDs()
			return nil, fmt.Errorf("endorsements: %w", err)
		}

		o.saveTripleIDs()
		o.tripleDbIDs = tripleIDs

	}

	ret, err := runQuery(ctx, db, o)
	o.restoreTripleIDs()
	return ret, err
}

func (o *ConditionalEndorsementTripleQuery) IsEmpty() bool {
	return o.ManifestCommonQuery.IsEmpty() &&
		o.ModuleTagCommonQuery.IsEmpty() &&
		o.isActive == nil &&
		o.ConditionGroup().IsEmpty() &&
		o.EndorsementGroup().IsEmpty()
}

func (o *ConditionalEndorsementTripleQuery) saveTripleIDs() {
	if o.tripleSaved {
		return
	}

	o.savedTripleIDs = o.tripleDbIDs
	o.tripleSaved = true
}

func (o *ConditionalEndorsementTripleQuery) restoreTripleIDs() {
	if !o.tripleSaved {
		return
	}

	o.tripleDbIDs = o.savedTripleIDs
	o.tripleSaved = false
}

type ConditionalEndorsementSeriesRecordQuery struct {
	modelQuery

	tripleIDs []int64

	selections *MeasurementQueryGroup
	additions  *MeasurementQueryGroup
}

func NewConditionalEndorsementSeriesRecordQuery() *ConditionalEndorsementSeriesRecordQuery {
	return &ConditionalEndorsementSeriesRecordQuery{}
}

func (o *ConditionalEndorsementSeriesRecordQuery) ID(value ...int64) *ConditionalEndorsementSeriesRecordQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *ConditionalEndorsementSeriesRecordQuery) TripleID(
	value ...int64,
) *ConditionalEndorsementSeriesRecordQuery {
	o.tripleIDs = append(o.tripleIDs, value...)
	return o
}

func (o *ConditionalEndorsementSeriesRecordQuery) SelectionGroup() *MeasurementQueryGroup {
	if o.selections == nil {
		o.selections = NewMeasurementQueryGroup("ces_record_selection")
	}

	return o.selections
}

func (o *ConditionalEndorsementSeriesRecordQuery) Selection(
	updater func(*MeasurementQuery),
) *ConditionalEndorsementSeriesRecordQuery {
	query := NewMeasurementQuery()
	updater(query)
	o.SelectionGroup().Add(query)
	return o
}

func (o *ConditionalEndorsementSeriesRecordQuery) AdditionGroup() *MeasurementQueryGroup {
	if o.additions == nil {
		o.additions = NewMeasurementQueryGroup("ces_record_addition")
	}

	return o.additions
}

func (o *ConditionalEndorsementSeriesRecordQuery) Addition(
	updater func(*MeasurementQuery),
) *ConditionalEndorsementSeriesRecordQuery {
	query := NewMeasurementQuery()
	updater(query)
	o.AdditionGroup().Add(query)
	return o
}

func (o *ConditionalEndorsementSeriesRecordQuery) UpdateSelectQuery(
	query *bun.SelectQuery,
	dialect schema.Dialect,
) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	addOrGroupWhereClause("triple_id", o.tripleIDs, false, query, dialect)
}

func (o *ConditionalEndorsementSeriesRecordQuery) Run(
	ctx context.Context,
	db bun.IDB,
) ([]*model.ConditionalEndorsementSeriesRecord, error) {
	if !o.SelectionGroup().IsEmpty() {
		o.SelectionGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.ids...)
		})

		ids, err := o.SelectionGroup().RunOwnerConjunction(ctx, db)

		o.SelectionGroup().ForEach(func(q *MeasurementQuery) {
			q.ownedQuery.ownerIDs = nil
		})

		if err != nil {
			return nil, fmt.Errorf("selection: %w", err)
		}

		o.modelQuery.saveIDs()
		o.modelQuery.ids = ids

	}

	if !o.AdditionGroup().IsEmpty() {
		o.AdditionGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.ids...)
		})

		ids, err := o.AdditionGroup().RunOwnerConjunction(ctx, db)

		o.AdditionGroup().ForEach(func(q *MeasurementQuery) {
			q.ownedQuery.ownerIDs = nil
		})

		if err != nil {
			o.modelQuery.restoreIDs()
			return nil, fmt.Errorf("addition: %w", err)
		}

		o.modelQuery.saveIDs()
		o.modelQuery.ids = ids

	}

	ret, err := runQuery(ctx, db, o)
	o.modelQuery.restoreIDs()
	return ret, err
}

func (o *ConditionalEndorsementSeriesRecordQuery) IsEmpty() bool {
	return o.modelQuery.IsEmpty() &&
		len(o.tripleIDs) == 0 &&
		o.SelectionGroup().IsEmpty() &&
		o.AdditionGroup().IsEmpty()
}

type ConditionalEndorsementSeriesTripleQuery struct {
	ManifestCommonQuery
	ModuleTagCommonQuery

	isActive *bool

	tripleDbIDs    []int64
	environmentIDs []int64

	environmentQuery *EnvironmentQuery
	authByQuery      *CryptoKeyQuery
	measurementGroup *MeasurementQueryGroup
	records          *ConditionalEndorsementSeriesRecordQueryGroup

	savedTripleIDs []int64
	tripleSaved    bool
	savedEnvIDs    []int64
	envSaved       bool
}

func NewConditionalEndorsementSeriesTripleQuery() *ConditionalEndorsementSeriesTripleQuery {
	return &ConditionalEndorsementSeriesTripleQuery{}
}

func (o *ConditionalEndorsementSeriesTripleQuery) TripleDbID(
	value ...int64,
) *ConditionalEndorsementSeriesTripleQuery {
	o.tripleDbIDs = append(o.tripleDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ManifestDbID(
	value ...int64,
) *ConditionalEndorsementSeriesTripleQuery {
	o.manifestDbIDs = append(o.manifestDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ModuleTagDbID(
	value ...int64,
) *ConditionalEndorsementSeriesTripleQuery {
	o.moduleTagDbIDs = append(o.moduleTagDbIDs, value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) IsActive(
	value bool,
) *ConditionalEndorsementSeriesTripleQuery {
	o.isActive = &value
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ManifestIDType(
	value ...model.TagIDType,
) *ConditionalEndorsementSeriesTripleQuery {
	o.manifestIDTypes = append(o.manifestIDTypes, value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ManifestIDValue(
	value ...string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.manifestIDValues = append(o.manifestIDValues, value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ManifestID(
	typ model.TagIDType, value string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.manifestIDs = append(o.manifestIDs, &manifestIDQueryEntry{typ, value})
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ModuleTagIDType(
	value ...model.TagIDType,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagIDType(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ModuleTagIDValue(
	value ...string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagIDValue(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ModuleTagID(
	typ model.TagIDType, value string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.moduleTagIDs = append(o.moduleTagIDs, &moduleTagIDQueryEntry{typ, value})
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ModuleTagVersion(
	value ...uint,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ModuleTagCommonQuery.ModuleTagVersion(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) Language(
	value ...string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ModuleTagCommonQuery.Language(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) Label(
	value ...string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.Label(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ProfileType(
	value ...model.ProfileType,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ProfileType(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ProfileValue(
	value ...string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ProfileValue(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) Profile(
	typ model.ProfileType, value string,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.Profile(typ, value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ProfileFromEAT(
	value ...*eat.Profile,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ProfileFromEAT(value...)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) AddedBefore(
	value time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.AddedBefore(value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) AddedAfter(
	value time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.AddedAfter(value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) AddedBetween(
	lower, upper time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.AddedBetween(lower, upper)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ValidBefore(
	value time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ValidBefore(value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ValidAfter(
	value time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ValidAfter(value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ValidBetween(
	lower, upper time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ValidBetween(lower, upper)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) ValidOn(
	value time.Time,
) *ConditionalEndorsementSeriesTripleQuery {
	o.ManifestCommonQuery.ValidOn(value)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) EnvironmentSubquery() *EnvironmentQuery {
	if o.environmentQuery == nil {
		o.environmentQuery = NewEnvironmentQuery(false)
	}

	return o.environmentQuery
}

func (o *ConditionalEndorsementSeriesTripleQuery) ExactEnvironment(
	value bool,
) *ConditionalEndorsementSeriesTripleQuery {
	o.EnvironmentSubquery().Exact = value
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) Environment(
	updater func(*EnvironmentQuery),
) *ConditionalEndorsementSeriesTripleQuery {
	updater(o.EnvironmentSubquery())
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) RecordGroup() *ConditionalEndorsementSeriesRecordQueryGroup {
	if o.records == nil {
		o.records = NewConditionalEndorsementSeriesRecordQueryGroup()
	}

	return o.records
}

func (o *ConditionalEndorsementSeriesTripleQuery) Record(
	updater func(*ConditionalEndorsementSeriesRecordQuery),
) *ConditionalEndorsementSeriesTripleQuery {
	query := NewConditionalEndorsementSeriesRecordQuery()
	updater(query)
	o.RecordGroup().Add(query)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) MeasurementGroup() *MeasurementQueryGroup {
	if o.measurementGroup == nil {
		o.measurementGroup = NewMeasurementQueryGroup("ces_condition")
	}

	return o.measurementGroup
}

func (o *ConditionalEndorsementSeriesTripleQuery) Measurement(
	updater func(e *MeasurementQuery),
) *ConditionalEndorsementSeriesTripleQuery {
	mq := NewMeasurementQuery().OwnerType("ces_condition")
	updater(mq)
	o.MeasurementGroup().Add(mq)
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) AuthorizedBySubquery() *CryptoKeyQuery {
	if o.authByQuery == nil {
		o.authByQuery = NewCryptoKeyQuery()
	}

	return o.authByQuery
}

func (o *ConditionalEndorsementSeriesTripleQuery) AuthorizedBy(
	updater func(e *CryptoKeyQuery),
) *ConditionalEndorsementSeriesTripleQuery {
	updater(o.AuthorizedBySubquery())
	return o
}

func (o *ConditionalEndorsementSeriesTripleQuery) UpdateSelectQuery(
	query *bun.SelectQuery,
	dialect schema.Dialect,
) {
	o.ManifestCommonQuery.UpdateSelectQuery(query, dialect)
	o.ModuleTagCommonQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("environment_db_id", o.environmentIDs, false, query, dialect)
	addOrGroupWhereClause("triple_db_id", o.tripleDbIDs, false, query, dialect)

	if o.isActive != nil {
		query.Where("is_active = ?", *o.isActive)
	}
}

func (o *ConditionalEndorsementSeriesTripleQuery) Run(
	ctx context.Context,
	db bun.IDB,
) ([]*model.ConditionalEndorsementSeriesTripleEntry, error) {
	if !o.EnvironmentSubquery().IsEmpty() {
		savedQueryIDs := slices.Clone(o.EnvironmentSubquery().ids)
		environments, err := o.EnvironmentSubquery().ID(o.environmentIDs...).Run(ctx, db)
		o.EnvironmentSubquery().ids = savedQueryIDs
		if err != nil {
			return nil, fmt.Errorf("condition environment: %w", err)
		}

		o.saveEnvIDs()
		o.environmentIDs = make([]int64, len(environments))
		for i, env := range environments {
			o.environmentIDs[i] = env.ID
		}
	}

	if !o.AuthorizedBySubquery().IsEmpty() {
		o.saveTripleIDs()
		cryptoKeys, err := o.AuthorizedBySubquery().
			OwnerType("ces_condition").
			OwnerID(o.tripleDbIDs...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.AuthorizedBySubquery().ownerTypes = nil
		o.AuthorizedBySubquery().ownerIDs = nil

		if err != nil {
			o.restoreEnvIDs()
			o.restoreTripleIDs()

			return nil, fmt.Errorf("condition auth by: %w", err)
		}

		o.tripleDbIDs = make([]int64, len(cryptoKeys))
		for i, ck := range cryptoKeys {
			o.tripleDbIDs[i] = ck.OwnerID
		}
	}

	if !o.MeasurementGroup().IsEmpty() {
		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.OwnerID(o.tripleDbIDs...)
		})

		tripleIDs, err := o.MeasurementGroup().RunOwnerConjunction(ctx, db)

		o.MeasurementGroup().ForEach(func(q *MeasurementQuery) {
			q.ownerIDs = nil
		})

		if err != nil {
			o.restoreEnvIDs()
			o.restoreTripleIDs()

			return nil, fmt.Errorf("condition measurements: %w", err)
		}

		o.saveTripleIDs()
		o.tripleDbIDs = tripleIDs
	}

	if !o.RecordGroup().IsEmpty() {
		o.RecordGroup().ForEach(func(q *ConditionalEndorsementSeriesRecordQuery) {
			q.TripleID(o.tripleDbIDs...)
		})

		tripleIDs, err := o.RecordGroup().RunOwnerConjunction(ctx, db)

		o.RecordGroup().ForEach(func(q *ConditionalEndorsementSeriesRecordQuery) {
			q.tripleIDs = nil
		})

		if err != nil {
			o.restoreEnvIDs()
			o.restoreTripleIDs()

			return nil, fmt.Errorf("records: %w", err)
		}

		o.saveTripleIDs()
		o.tripleDbIDs = tripleIDs
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreTripleIDs()
	o.restoreEnvIDs()
	return ret, err
}

func (o *ConditionalEndorsementSeriesTripleQuery) IsEmpty() bool {
	return o.ManifestCommonQuery.IsEmpty() &&
		o.ModuleTagCommonQuery.IsEmpty() &&
		o.isActive == nil &&
		o.EnvironmentSubquery().IsEmpty() &&
		o.MeasurementGroup().IsEmpty() &&
		o.AuthorizedBySubquery().IsEmpty()
}

func (o *ConditionalEndorsementSeriesTripleQuery) saveTripleIDs() {
	if o.tripleSaved {
		return
	}

	o.savedTripleIDs = o.tripleDbIDs
	o.tripleSaved = true
}

func (o *ConditionalEndorsementSeriesTripleQuery) restoreTripleIDs() {
	if !o.tripleSaved {
		return
	}

	o.tripleDbIDs = o.savedTripleIDs
	o.tripleSaved = false
}

func (o *ConditionalEndorsementSeriesTripleQuery) saveEnvIDs() {
	if o.envSaved {
		return
	}

	o.savedEnvIDs = o.environmentIDs
	o.environmentIDs = slices.Clone(o.savedEnvIDs)
	o.envSaved = true
}

func (o *ConditionalEndorsementSeriesTripleQuery) restoreEnvIDs() {
	if !o.envSaved {
		return
	}

	o.environmentIDs = o.savedEnvIDs
	o.envSaved = false
}

type CryptoKeyQuery struct {
	modelQuery
	ownedQuery

	keyTypes []string
	keyBytes [][]byte
	keys     []*keyQueryEntry
}

func NewCryptoKeyQuery() *CryptoKeyQuery {
	return &CryptoKeyQuery{}
}

func (o *CryptoKeyQuery) ID(value ...int64) *CryptoKeyQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *CryptoKeyQuery) OwnerType(value ...string) *CryptoKeyQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *CryptoKeyQuery) OwnerID(value ...int64) *CryptoKeyQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *CryptoKeyQuery) Owner(typ string, id int64) *CryptoKeyQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *CryptoKeyQuery) OwnerFromModel(value ...*model.CryptoKey) *CryptoKeyQuery {
	for _, ck := range value {
		o.Owner(ck.OwnerType, ck.OwnerID)
	}
	return o
}

func (o *CryptoKeyQuery) KeyType(value ...string) *CryptoKeyQuery {
	o.keyTypes = append(o.keyTypes, value...)
	return o
}

func (o *CryptoKeyQuery) KeyBytes(value ...[]byte) *CryptoKeyQuery {
	o.keyBytes = append(o.keyBytes, value...)
	return o
}

func (o *CryptoKeyQuery) Key(typ string, value []byte) *CryptoKeyQuery {
	o.keys = append(o.keys, &keyQueryEntry{typ, value})
	return o
}

func (o *CryptoKeyQuery) KeyFromModel(value ...*model.CryptoKey) *CryptoKeyQuery {
	for _, ck := range value {
		o.Key(ck.KeyType, ck.KeyBytes)
	}
	return o
}

func (o *CryptoKeyQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("key_type", o.keyTypes, false, query, dialect)
	addOrGroupWhereClause("key_bytes", o.keyBytes, false, query, dialect)
	updateQueryWithEntries(o.keys, query, dialect)
}

func (o *CryptoKeyQuery) Run(ctx context.Context, db bun.IDB) ([]*model.CryptoKey, error) {
	return runQuery(ctx, db, o)
}

func (o *CryptoKeyQuery) IsEmpty() bool {
	return len(o.keyTypes) == 0 &&
		len(o.keyBytes) == 0 &&
		len(o.keys) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty()
}

type DigestQuery struct {
	modelQuery
	ownedQuery

	TextAlgIDs []string
	IntAlgIDs  []int64
	values     [][]byte
	digests    []*digestQueryEntry
}

func NewDigestQuery() *DigestQuery {
	return &DigestQuery{}
}

func (o *DigestQuery) ID(value ...int64) *DigestQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *DigestQuery) OwnerType(value ...string) *DigestQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *DigestQuery) OwnerID(value ...int64) *DigestQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *DigestQuery) Owner(typ string, id int64) *DigestQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *DigestQuery) OwnerFromModel(value ...*model.Digest) *DigestQuery {
	for _, digest := range value {
		o.Owner(digest.OwnerType, digest.OwnerID)
	}
	return o
}

func (o *DigestQuery) IntAlgID(value ...int64) *DigestQuery {
	o.IntAlgIDs = append(o.IntAlgIDs, value...)
	return o
}

func (o *DigestQuery) TextAlgID(value ...string) *DigestQuery {
	o.TextAlgIDs = append(o.TextAlgIDs, value...)
	return o
}

func (o *DigestQuery) Value(value ...[]byte) *DigestQuery {
	o.values = append(o.values, value...)
	return o
}

func (o *DigestQuery) Digest(algID any, value []byte) *DigestQuery {
	switch t := algID.(type) {
	case int:
		return o.DigestIntAlg(int64(t), value)
	case int64:
		return o.DigestIntAlg(t, value)
	case uint64:
		if t <= math.MaxInt64 {
			return o.DigestIntAlg(int64(t), value)
		}

		return o
	case string:
		return o.DigestTextAlg(t, value)
	default:
		return o
	}
}

func (o *DigestQuery) DigestIntAlg(algID int64, value []byte) *DigestQuery {
	o.digests = append(o.digests, &digestQueryEntry{algID, "", value})
	return o
}

func (o *DigestQuery) DigestTextAlg(algID string, value []byte) *DigestQuery {
	o.digests = append(o.digests, &digestQueryEntry{0, algID, value})
	return o
}

func (o *DigestQuery) DigestFromModel(value ...*model.Digest) *DigestQuery {
	for _, v := range value {
		if v.AlgIDText == "" {
			o.DigestIntAlg(v.AlgIDInt, v.Value)
		} else {
			o.DigestTextAlg(v.AlgIDText, v.Value)
		}
	}
	return o
}

func (o *DigestQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)

	if len(o.IntAlgIDs) != 0 || len(o.TextAlgIDs) != 0 {
		query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			whereText := fmt.Sprintf(`%s = ?`, identQuote("alg_id_int", dialect))
			for _, value := range o.IntAlgIDs {
				q.WhereOr(whereText, value)
			}

			whereText = fmt.Sprintf(`%s = ?`, identQuote("alg_id_text", dialect))
			for _, value := range o.TextAlgIDs {
				q.WhereOr(whereText, value)
			}

			return q
		})
	}

	addOrGroupWhereClause("value", o.values, false, query, dialect)
	updateQueryWithEntries(o.digests, query, dialect)
}

func (o *DigestQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Digest, error) {
	return runQuery(ctx, db, o)
}

func (o *DigestQuery) IsEmpty() bool {
	return len(o.IntAlgIDs) == 0 &&
		len(o.TextAlgIDs) == 0 &&
		len(o.values) == 0 &&
		len(o.digests) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty()
}

type IntegrityRegisterQuery struct {
	modelQuery

	indexUints     []uint64
	indexTexts     []string
	measurementIDs []int64

	digestsQuery *DigestQuery
}

func NewIntegrityRegisterQuery() *IntegrityRegisterQuery {
	return &IntegrityRegisterQuery{}
}

func (o *IntegrityRegisterQuery) ID(value ...int64) *IntegrityRegisterQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *IntegrityRegisterQuery) IndexUint(value ...uint64) *IntegrityRegisterQuery {
	o.indexUints = append(o.indexUints, value...)
	return o
}

func (o *IntegrityRegisterQuery) IndexText(value ...string) *IntegrityRegisterQuery {
	o.indexTexts = append(o.indexTexts, value...)
	return o
}

func (o *IntegrityRegisterQuery) MeasurementID(value ...int64) *IntegrityRegisterQuery {
	o.measurementIDs = append(o.measurementIDs, value...)
	return o
}

func (o *IntegrityRegisterQuery) DigestsSubquery() *DigestQuery {
	if o.digestsQuery == nil {
		o.digestsQuery = NewDigestQuery()
	}

	return o.digestsQuery
}

func (o *IntegrityRegisterQuery) DigestID(value ...int64) *IntegrityRegisterQuery {
	o.DigestsSubquery().ID(value...)
	return o
}

func (o *IntegrityRegisterQuery) DigestIntAlgID(value ...int64) *IntegrityRegisterQuery {
	o.DigestsSubquery().IntAlgID(value...)
	return o
}

func (o *IntegrityRegisterQuery) DigestTextAlgID(value ...string) *IntegrityRegisterQuery {
	o.DigestsSubquery().TextAlgID(value...)
	return o
}

func (o *IntegrityRegisterQuery) DigestValue(algID uint64, value []byte) *IntegrityRegisterQuery {
	o.DigestsSubquery().Digest(algID, value)
	return o
}

func (o *IntegrityRegisterQuery) DigestFromModel(value ...*model.Digest) *IntegrityRegisterQuery {
	o.DigestsSubquery().DigestFromModel(value...)
	return o
}

func (o *IntegrityRegisterQuery) UpdateFromModel(value *model.IntegrityRegister) *IntegrityRegisterQuery {
	if value.IndexUint != nil {
		o.IndexUint(*value.IndexUint)
	} else if value.IndexText != nil {
		o.IndexText(*value.IndexText)
	}

	for _, digestModel := range value.Digests {
		o.DigestsSubquery().DigestFromModel(digestModel)
	}

	return o
}

func (o *IntegrityRegisterQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("index_uint", o.indexUints, false, query, dialect)
	addOrGroupWhereClause("index_text", o.indexTexts, false, query, dialect)
	addOrGroupWhereClause("measurement_id", o.measurementIDs, false, query, dialect)
}

func (o *IntegrityRegisterQuery) Run(ctx context.Context, db bun.IDB) ([]*model.IntegrityRegister, error) {
	if !o.DigestsSubquery().IsEmpty() {
		o.saveIDs()

		digests, err := o.DigestsSubquery().
			OwnerType("integrity_register").
			OwnerID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.DigestsSubquery().ownerTypes = nil
		o.DigestsSubquery().ownerIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("digests: %w", err)
		}

		o.ids = make([]int64, len(digests))
		for i, digest := range digests {
			o.ids[i] = digest.OwnerID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	return ret, err
}

func (o *IntegrityRegisterQuery) IsEmpty() bool {
	return len(o.indexUints) == 0 &&
		len(o.indexTexts) == 0 &&
		len(o.measurementIDs) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.DigestsSubquery().IsEmpty()
}

type FlagQuery struct {
	modelQuery

	codePoints     []int64
	value          *bool
	entries        []*flagQueryEntry
	measurementIDs []int64
}

func NewFlagQuery() *FlagQuery {
	return &FlagQuery{}
}

func (o *FlagQuery) ID(value ...int64) *FlagQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *FlagQuery) CodePoint(value ...int64) *FlagQuery {
	o.codePoints = append(o.codePoints, value...)
	return o
}

func (o *FlagQuery) Value(value bool) *FlagQuery {
	o.value = &value
	return o
}

func (o *FlagQuery) Flag(codePoint int64, value bool) *FlagQuery {
	o.entries = append(o.entries, &flagQueryEntry{codePoint, value})
	return o
}

func (o *FlagQuery) MeasurementID(value ...int64) *FlagQuery {
	o.measurementIDs = append(o.measurementIDs, value...)
	return o
}

func (o *FlagQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("code_point", o.codePoints, false, query, dialect)

	if o.value != nil {
		query.Where("value = ?", *o.value)
	}

	addOrGroupWhereClause("measurement_id", o.measurementIDs, false, query, dialect)
	updateQueryWithEntries(o.entries, query, dialect)
}

func (o *FlagQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Flag, error) {
	return runQuery(ctx, db, o)
}

func (o *FlagQuery) IsEmpty() bool {
	return len(o.codePoints) == 0 &&
		o.value == nil &&
		len(o.entries) == 0 &&
		len(o.measurementIDs) == 0 &&
		o.modelQuery.IsEmpty()
}

type MeasurementValueQuery struct {
	modelQuery

	codePoints     []int64
	valueTypes     []string
	valueBytes     [][]byte
	valueTexts     []string
	valueInts      []int64
	values         []*valueQueryEntry
	measurementIDs []int64
}

func NewMeasurementValueQuery() *MeasurementValueQuery {
	return &MeasurementValueQuery{}
}

func (o *MeasurementValueQuery) ID(value ...int64) *MeasurementValueQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *MeasurementValueQuery) CodePoint(value ...int64) *MeasurementValueQuery {
	o.codePoints = append(o.codePoints, value...)
	return o
}

func (o *MeasurementValueQuery) ValueType(value ...string) *MeasurementValueQuery {
	o.valueTypes = append(o.valueTypes, value...)
	return o
}

func (o *MeasurementValueQuery) ValueBytes(value ...[]byte) *MeasurementValueQuery {
	o.valueBytes = append(o.valueBytes, value...)
	return o
}

func (o *MeasurementValueQuery) ValueText(value ...string) *MeasurementValueQuery {
	o.valueTexts = append(o.valueTexts, value...)
	return o
}

func (o *MeasurementValueQuery) ValueInt(value ...int64) *MeasurementValueQuery {
	o.valueInts = append(o.valueInts, value...)
	return o
}

func (o *MeasurementValueQuery) Value(typ string, value any) *MeasurementValueQuery {
	qv, err := newValueQueryEntry(typ, value)
	if err != nil {
		// Since this method is meant to be chained, we cannot
		// propagate the error here; so instead, we add a "type"
		// that contains the err and is guaranteed not to match
		// anything in the database, thus ensuring that  the query
		// will return ErrNoMatch when run, and the actual error
		// is visible in the SQL. This is not ideal, but is better
		// than ignoring the error entirely.
		o.valueTypes = append(o.valueTypes, fmt.Sprintf("@ERROR: %s@", err.Error()))
		return o

	}

	o.values = append(o.values, qv)
	return o
}

// AddValue is a non-chaining version of Value() that returns an error if the
// provided value has an unexpected type.
func (o *MeasurementValueQuery) AddValue(typ string, value any) error {
	qv, err := newValueQueryEntry(typ, value)
	if err != nil {
		return err

	}

	o.values = append(o.values, qv)
	return nil
}

func (o *MeasurementValueQuery) UpdateFromModel(entry *model.MeasurementValueEntry) *MeasurementValueQuery {
	o.CodePoint(entry.CodePoint)
	o.ValueType(entry.ValueType)

	if entry.ValueBytes != nil { // nolint:gocritic
		o.ValueBytes(*entry.ValueBytes)
	} else if entry.ValueText != nil {
		o.ValueText(*entry.ValueText)
	} else {
		o.ValueInt(*entry.ValueInt)
	}

	return o
}

func (o *MeasurementValueQuery) MeasurementID(value ...int64) *MeasurementValueQuery {
	o.measurementIDs = append(o.measurementIDs, value...)
	return o
}

func (o *MeasurementValueQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("code_point", o.codePoints, false, query, dialect)
	addOrGroupWhereClause("value_type", o.valueTypes, false, query, dialect)
	addOrGroupWhereClause("value_text", o.valueTexts, false, query, dialect)
	addOrGroupWhereClause("value_bytes", o.valueBytes, false, query, dialect)
	addOrGroupWhereClause("value_int", o.valueInts, false, query, dialect)
	updateQueryWithEntries(o.values, query, dialect)

	addOrGroupWhereClause("measurement_id", o.measurementIDs, false, query, dialect)
}

func (o *MeasurementValueQuery) Run(ctx context.Context, db bun.IDB) ([]*model.MeasurementValueEntry, error) {
	return runQuery(ctx, db, o)
}

func (o *MeasurementValueQuery) IsEmpty() bool {
	return len(o.codePoints) == 0 &&
		len(o.valueTypes) == 0 &&
		len(o.valueBytes) == 0 &&
		len(o.valueTexts) == 0 &&
		len(o.valueInts) == 0 &&
		len(o.values) == 0 &&
		len(o.measurementIDs) == 0 &&
		o.modelQuery.IsEmpty()
}

type MeasurementQuery struct {
	modelQuery
	ownedQuery

	mkeyTypes []string
	mkeyBytes [][]byte
	mkeys     []*keyQueryEntry

	authByQuery            *CryptoKeyQuery
	mvalQuery              *MeasurementValueQuery
	digestQuery            *DigestQuery
	integrityRegisterQuery *IntegrityRegisterQuery
	flagQuery              *FlagQuery
}

func NewMeasurementQuery() *MeasurementQuery {
	return &MeasurementQuery{}
}

func (o *MeasurementQuery) ID(value ...int64) *MeasurementQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *MeasurementQuery) OwnerType(value ...string) *MeasurementQuery {
	o.ownedQuery.OwnerType(value...)
	return o
}

func (o *MeasurementQuery) OwnerID(value ...int64) *MeasurementQuery {
	o.ownedQuery.OwnerID(value...)
	return o
}

func (o *MeasurementQuery) Owner(typ string, id int64) *MeasurementQuery {
	o.ownedQuery.Owner(typ, id)
	return o
}

func (o *MeasurementQuery) MkeyType(value ...string) *MeasurementQuery {
	o.mkeyTypes = append(o.mkeyTypes, value...)
	return o
}

func (o *MeasurementQuery) MkeyBytes(value ...[]byte) *MeasurementQuery {
	o.mkeyBytes = append(o.mkeyBytes, value...)
	return o
}

func (o *MeasurementQuery) Mkey(typ string, value []byte) *MeasurementQuery {
	o.mkeys = append(o.mkeys, &keyQueryEntry{typ, value})
	return o
}

func (o *MeasurementQuery) AuthorizedBySubquery() *CryptoKeyQuery {
	if o.authByQuery == nil {
		o.authByQuery = NewCryptoKeyQuery()
	}

	return o.authByQuery
}

func (o *MeasurementQuery) AuthorizedByType(value ...string) *MeasurementQuery {
	o.AuthorizedBySubquery().KeyType(value...)
	return o
}

func (o *MeasurementQuery) AuthorizedByBytes(value ...[]byte) *MeasurementQuery {
	o.AuthorizedBySubquery().KeyBytes(value...)
	return o
}

func (o *MeasurementQuery) AuthorizedBy(typ string, value []byte) *MeasurementQuery {
	o.AuthorizedBySubquery().Key(typ, value)
	return o
}

func (o *MeasurementQuery) AuthorizedByFromModel(value ...*model.CryptoKey) *MeasurementQuery {
	o.AuthorizedBySubquery().KeyFromModel(value...)
	return o
}

func (o *MeasurementQuery) ValueSubquery() *MeasurementValueQuery {
	if o.mvalQuery == nil {
		o.mvalQuery = NewMeasurementValueQuery()
	}

	return o.mvalQuery
}

func (o *MeasurementQuery) MVal(updater func(*MeasurementValueQuery)) *MeasurementQuery {
	updater(o.ValueSubquery())
	return o
}

func (o *MeasurementQuery) DigestsSubquery() *DigestQuery {
	if o.digestQuery == nil {
		o.digestQuery = NewDigestQuery()
	}

	return o.digestQuery
}

func (o *MeasurementQuery) DigestIntAlgID(value ...int64) *MeasurementQuery {
	o.DigestsSubquery().IntAlgID(value...)
	return o
}

func (o *MeasurementQuery) DigestTextAlgID(value ...string) *MeasurementQuery {
	o.DigestsSubquery().TextAlgID(value...)
	return o
}

func (o *MeasurementQuery) DigestValue(value ...[]byte) *MeasurementQuery {
	o.DigestsSubquery().Value(value...)
	return o
}

func (o *MeasurementQuery) Digest(algID uint64, value []byte) *MeasurementQuery {
	o.DigestsSubquery().Digest(algID, value)
	return o
}

func (o *MeasurementQuery) DigestFromModel(value ...*model.Digest) *MeasurementQuery {
	o.DigestsSubquery().DigestFromModel(value...)
	return o
}

func (o *MeasurementQuery) IntegrityRegistersSubquery() *IntegrityRegisterQuery {
	if o.integrityRegisterQuery == nil {
		o.integrityRegisterQuery = NewIntegrityRegisterQuery()
	}

	return o.integrityRegisterQuery
}

func (o *MeasurementQuery) IntegrityRegister(updater func(*IntegrityRegisterQuery)) *MeasurementQuery {
	updater(o.IntegrityRegistersSubquery())
	return o
}

func (o *MeasurementQuery) FlagsSubquery() *FlagQuery {
	if o.flagQuery == nil {
		o.flagQuery = NewFlagQuery()
	}

	return o.flagQuery
}

func (o *MeasurementQuery) Flag(updater func(*FlagQuery)) *MeasurementQuery {
	updater(o.FlagsSubquery())
	return o
}

func (o *MeasurementQuery) UpdateFromModel(value *model.Measurement) *MeasurementQuery {
	if value.KeyType != nil && value.KeyBytes != nil {
		o.Mkey(*value.KeyType, *value.KeyBytes)
	}

	for _, digest := range value.Digests {
		o.DigestFromModel(digest)
	}

	for _, flag := range value.Flags {
		o.Flag(func(fq *FlagQuery) {
			fq.Flag(flag.CodePoint, flag.Value)
		})
	}

	for _, ireg := range value.IntegrityRegisters {
		o.IntegrityRegister(func(irq *IntegrityRegisterQuery) {
			irq.UpdateFromModel(ireg)
		})
	}

	o.AuthorizedByFromModel(value.AuthorizedBy...)

	for _, entry := range value.ValueEntries {
		o.ValueSubquery().UpdateFromModel(entry)
	}

	return o
}

func (o *MeasurementQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)
	o.ownedQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("key_type", o.mkeyTypes, false, query, dialect)
	addOrGroupWhereClause("key_bytes", o.mkeyBytes, false, query, dialect)
	updateQueryWithEntries(o.mkeys, query, dialect)
}

func (o *MeasurementQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Measurement, error) {
	if !o.AuthorizedBySubquery().IsEmpty() {
		o.saveIDs()

		cks, err := o.AuthorizedBySubquery().
			OwnerType("measurement").
			OwnerID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.AuthorizedBySubquery().ownerTypes = nil
		o.AuthorizedBySubquery().ownerIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("auth_by: %w", err)
		}

		o.ids = make([]int64, len(cks))
		for i, ck := range cks {
			o.ids[i] = ck.OwnerID
		}
	}

	if !o.DigestsSubquery().IsEmpty() {
		o.saveIDs()

		digests, err := o.DigestsSubquery().
			OwnerType("measurement").
			OwnerID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.DigestsSubquery().ownerTypes = nil
		o.DigestsSubquery().ownerIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("digests: %w", err)
		}

		o.ids = make([]int64, len(digests))
		for i, digest := range digests {
			o.ids[i] = digest.OwnerID
		}
	}

	if !o.ValueSubquery().IsEmpty() {
		o.saveIDs()

		mvals, err := o.ValueSubquery().MeasurementID(o.ids...).Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.ValueSubquery().measurementIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("mval: %w", err)
		}

		o.ids = make([]int64, len(mvals))
		for i, mval := range mvals {
			o.ids[i] = mval.MeasurementID
		}
	}

	if !o.IntegrityRegistersSubquery().IsEmpty() {
		o.saveIDs()

		regs, err := o.IntegrityRegistersSubquery().MeasurementID(o.ids...).Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.IntegrityRegistersSubquery().measurementIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("integrity registers: %w", err)
		}

		o.ids = make([]int64, len(regs))
		for i, reg := range regs {
			o.ids[i] = reg.MeasurementID
		}
	}

	if !o.FlagsSubquery().IsEmpty() {
		o.saveIDs()

		flags, err := o.FlagsSubquery().MeasurementID(o.ids...).Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.FlagsSubquery().measurementIDs = nil

		if err != nil {
			o.restoreIDs()
			return nil, fmt.Errorf("flags: %w", err)
		}

		o.ids = make([]int64, len(flags))
		for i, flag := range flags {
			o.ids[i] = flag.MeasurementID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	return ret, err
}

func (o *MeasurementQuery) IsEmpty() bool {
	return len(o.mkeyTypes) == 0 &&
		len(o.mkeyBytes) == 0 &&
		len(o.mkeys) == 0 &&
		o.modelQuery.IsEmpty() &&
		o.ownedQuery.IsEmpty() &&
		o.AuthorizedBySubquery().IsEmpty() &&
		o.ValueSubquery().IsEmpty() &&
		o.DigestsSubquery().IsEmpty() &&
		o.IntegrityRegistersSubquery().IsEmpty() &&
		o.FlagsSubquery().IsEmpty()
}

type TokenQuery struct {
	modelQuery

	manifestIDs []string
	isSigned    []bool
	data        [][]byte

	authoritySubquery *CryptoKeyQuery
}

func NewTokenQuery() *TokenQuery {
	return &TokenQuery{}
}

func (o *TokenQuery) ID(value ...int64) *TokenQuery {
	o.modelQuery.ID(value...)
	return o
}

func (o *TokenQuery) ManifestID(value ...string) *TokenQuery {
	o.manifestIDs = append(o.manifestIDs, value...)
	return o
}

func (o *TokenQuery) IsSigned(value ...bool) *TokenQuery {
	o.isSigned = append(o.isSigned, value...)
	return o
}

func (o *TokenQuery) Data(value ...[]byte) *TokenQuery {
	o.data = append(o.data, value...)
	return o
}

func (o *TokenQuery) AuthoritySubquery() *CryptoKeyQuery {
	if o.authoritySubquery == nil {
		o.authoritySubquery = NewCryptoKeyQuery()
	}

	return o.authoritySubquery
}

func (o *TokenQuery) Authority(updater func(*CryptoKeyQuery)) *TokenQuery {
	updater(o.AuthoritySubquery())
	return o
}

func (o *TokenQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	o.modelQuery.UpdateSelectQuery(query, dialect)

	addOrGroupWhereClause("manifest_id", o.manifestIDs, false, query, dialect)
	addOrGroupWhereClause("is_signed", o.isSigned, false, query, dialect)
	addOrGroupWhereClause("data", o.data, false, query, dialect)
}

func (o *TokenQuery) Run(ctx context.Context, db bun.IDB) ([]*model.Token, error) {
	if !o.AuthoritySubquery().IsEmpty() {
		o.saveIDs()

		cks, err := o.AuthoritySubquery().
			OwnerType("token").
			OwnerID(o.ids...).
			Run(ctx, db)

		// reset so that it doesn't affect the IsEmpty() test if
		// the query is repeated.
		o.AuthoritySubquery().ownerTypes = nil
		o.AuthoritySubquery().ownerIDs = nil

		if err != nil {
			// coverage:ignore
			o.restoreIDs()
			return nil, fmt.Errorf("authority: %w", err)
		}

		o.ids = make([]int64, len(cks))
		for i, ck := range cks {
			o.ids[i] = ck.OwnerID
		}
	}

	ret, err := runQuery(ctx, db, o)
	o.restoreIDs()
	return ret, err
}

func (o *TokenQuery) IsEmpty() bool {
	return o.modelQuery.IsEmpty() &&
		len(o.manifestIDs) == 0 &&
		len(o.isSigned) == 0 &&
		len(o.data) == 0 &&
		o.AuthoritySubquery().IsEmpty()
}

type modelQuery struct {
	ids []int64

	savedIDs []int64
	saved    bool
}

func (o *modelQuery) ID(value ...int64) *modelQuery {
	o.ids = append(o.ids, value...)
	return o
}

func (o *modelQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	addOrGroupWhereClause("id", o.ids, false, query, dialect)
}

func (o *modelQuery) IsEmpty() bool {
	return len(o.ids) == 0
}

func (o *modelQuery) saveIDs() {
	if o.saved {
		return
	}

	o.savedIDs = o.ids
	o.saved = true
}

func (o *modelQuery) restoreIDs() {
	if !o.saved {
		return
	}

	o.ids = o.savedIDs
	o.saved = false
}

type ownedQuery struct {
	ownerIDs   []int64
	ownerTypes []string
	owners     []*ownerQueryEntry
}

func (o *ownedQuery) OwnerType(value ...string) *ownedQuery {
	o.ownerTypes = append(o.ownerTypes, value...)
	return o
}

func (o *ownedQuery) OwnerID(value ...int64) *ownedQuery {
	o.ownerIDs = append(o.ownerIDs, value...)
	return o
}

func (o *ownedQuery) Owner(typ string, id int64) *ownedQuery {
	o.owners = append(o.owners, &ownerQueryEntry{typ, id})
	return o
}

func (o *ownedQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	addOrGroupWhereClause("owner_id", o.ownerIDs, false, query, dialect)
	addOrGroupWhereClause("owner_type", o.ownerTypes, false, query, dialect)
	updateQueryWithEntries(o.owners, query, dialect)
}

func (o *ownedQuery) IsEmpty() bool {
	return len(o.ownerIDs) == 0 &&
		len(o.ownerTypes) == 0 &&
		len(o.owners) == 0
}

type ManifestCommonQuery struct {
	labels []string

	manifestDbIDs []int64

	manifestIDTypes  []model.TagIDType
	manifestIDValues []string
	manifestIDs      []*manifestIDQueryEntry

	profileTypes  []model.ProfileType
	profileValues []string
	profiles      []*profileQueryEntry

	timeAdded []*timePointQueryEntry
	validity  []*timePeriodQueryEntry

	savedIDs []int64
	saved    bool
}

func (o *ManifestCommonQuery) ManifestDbID(value ...int64) *ManifestCommonQuery {
	o.manifestDbIDs = append(o.manifestDbIDs, value...)
	return o
}

func (o *ManifestCommonQuery) ManifestIDType(value ...model.TagIDType) *ManifestCommonQuery {
	o.manifestIDTypes = append(o.manifestIDTypes, value...)
	return o
}

func (o *ManifestCommonQuery) ManifestIDValue(value ...string) *ManifestCommonQuery {
	o.manifestIDValues = append(o.manifestIDValues, value...)
	return o
}

func (o *ManifestCommonQuery) ManifestIDFromSWID(tag swid.TagID) *ManifestCommonQuery {
	typ, value := model.ParseSWIDTagID(tag)
	return o.ManifestID(typ, value)
}

func (o *ManifestCommonQuery) ManifestID(typ model.TagIDType, value string) *ManifestCommonQuery {
	o.manifestIDs = append(o.manifestIDs, &manifestIDQueryEntry{typ, value})
	return o
}

func (o *ManifestCommonQuery) Label(value ...string) *ManifestCommonQuery {
	o.labels = append(o.labels, value...)
	return o
}

func (o *ManifestCommonQuery) ProfileType(value ...model.ProfileType) *ManifestCommonQuery {
	o.profileTypes = append(o.profileTypes, value...)
	return o
}

func (o *ManifestCommonQuery) ProfileValue(value ...string) *ManifestCommonQuery {
	o.profileValues = append(o.profileValues, value...)
	return o
}

func (o *ManifestCommonQuery) Profile(typ model.ProfileType, value string) *ManifestCommonQuery {
	o.profiles = append(o.profiles, &profileQueryEntry{typ, value})
	return o
}

func (o *ManifestCommonQuery) ProfileFromEAT(values ...*eat.Profile) *ManifestCommonQuery {
	for _, profile := range values {
		if profile == nil || (!profile.IsOID() && !profile.IsURI()) {
			continue
		}

		value, err := profile.Get()
		if err != nil {
			panic(err)
		}

		typ := model.URIProfile
		if profile.IsOID() {
			typ = model.OIDProfile
		}

		o.profiles = append(o.profiles, &profileQueryEntry{typ, value})
	}

	return o
}

func (o *ManifestCommonQuery) AddedBefore(value time.Time) *ManifestCommonQuery {
	o.timeAdded = append(o.timeAdded, &timePointQueryEntry{
		field: "time_added",
		upper: &value,
	})
	return o
}

func (o *ManifestCommonQuery) AddedAfter(value time.Time) *ManifestCommonQuery {
	o.timeAdded = append(o.timeAdded, &timePointQueryEntry{
		field: "time_added",
		lower: &value,
	})
	return o
}

func (o *ManifestCommonQuery) AddedBetween(lower, upper time.Time) *ManifestCommonQuery {
	o.timeAdded = append(o.timeAdded, &timePointQueryEntry{
		field: "time_added",
		lower: &lower,
		upper: &upper,
	})
	return o
}

func (o *ManifestCommonQuery) ValidBefore(value time.Time) *ManifestCommonQuery {
	o.validity = append(o.validity, &timePeriodQueryEntry{
		lowerField: "not_before",
		upperField: "not_after",
		lower:      &value,
		optional:   true,
	})
	return o
}

func (o *ManifestCommonQuery) ValidAfter(value time.Time) *ManifestCommonQuery {
	o.validity = append(o.validity, &timePeriodQueryEntry{
		lowerField: "not_before",
		upperField: "not_after",
		upper:      &value,
		optional:   true,
	})
	return o
}

func (o *ManifestCommonQuery) ValidBetween(lower, upper time.Time) *ManifestCommonQuery {
	o.validity = append(o.validity, &timePeriodQueryEntry{
		lowerField: "not_before",
		upperField: "not_after",
		lower:      &lower,
		upper:      &upper,
		optional:   true,
	})
	return o
}

func (o *ManifestCommonQuery) ValidOn(value time.Time) *ManifestCommonQuery {
	o.validity = append(o.validity, &timePeriodQueryEntry{
		lowerField: "not_before",
		upperField: "not_after",
		lower:      &value,
		upper:      &value,
		optional:   true,
	})
	return o
}

func (o *ManifestCommonQuery) saveManifestDbIDs() {
	if o.saved {
		return
	}

	o.savedIDs = o.manifestDbIDs
	o.saved = true
}

func (o *ManifestCommonQuery) restoreManifestDbIDs() {
	if !o.saved {
		return
	}

	o.manifestDbIDs = o.savedIDs
	o.saved = false
}

func (o *ManifestCommonQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	addOrGroupWhereClause("manifest_db_id", o.manifestDbIDs, false, query, dialect)
	addOrGroupWhereClause("label", o.labels, false, query, dialect)

	addOrGroupWhereClause("manifest_id_type", o.manifestIDTypes, false, query, dialect)
	addOrGroupWhereClause("manifest_id", o.manifestIDValues, false, query, dialect)
	updateQueryWithEntries(o.manifestIDs, query, dialect)

	addOrGroupWhereClause("profile_type", o.profileTypes, false, query, dialect)
	addOrGroupWhereClause("profile", o.profileValues, false, query, dialect)
	updateQueryWithEntries(o.profiles, query, dialect)

	updateQueryWithEntries(o.timeAdded, query, dialect)
	updateQueryWithEntries(o.validity, query, dialect)
}

func (o *ManifestCommonQuery) IsEmpty() bool {
	return len(o.manifestDbIDs) == 0 &&
		len(o.labels) == 0 &&
		len(o.profileTypes) == 0 &&
		len(o.profileValues) == 0 &&
		len(o.profiles) == 0 &&
		len(o.timeAdded) == 0 &&
		len(o.validity) == 0
}

type ModuleTagCommonQuery struct {
	moduleTagDbIDs []int64

	moduleTagIDTypes  []model.TagIDType
	moduleTagIDValues []string
	moduleTagIDs      []*moduleTagIDQueryEntry

	languages         []string
	moduleTagVersions []uint

	savedIDs []int64
	saved    bool
}

func (o *ModuleTagCommonQuery) ModuleTagDbID(value ...int64) *ModuleTagCommonQuery {
	o.moduleTagDbIDs = append(o.moduleTagDbIDs, value...)
	return o
}

func (o *ModuleTagCommonQuery) ModuleTagIDType(value ...model.TagIDType) *ModuleTagCommonQuery {
	o.moduleTagIDTypes = append(o.moduleTagIDTypes, value...)
	return o
}

func (o *ModuleTagCommonQuery) ModuleTagIDValue(value ...string) *ModuleTagCommonQuery {
	o.moduleTagIDValues = append(o.moduleTagIDValues, value...)
	return o
}

func (o *ModuleTagCommonQuery) ModuleTagID(typ model.TagIDType, value string) *ModuleTagCommonQuery {
	o.moduleTagIDs = append(o.moduleTagIDs, &moduleTagIDQueryEntry{typ, value})
	return o
}

func (o *ModuleTagCommonQuery) Language(value ...string) *ModuleTagCommonQuery {
	o.languages = append(o.languages, value...)
	return o
}

func (o *ModuleTagCommonQuery) ModuleTagVersion(value ...uint) *ModuleTagCommonQuery {
	o.moduleTagVersions = append(o.moduleTagVersions, value...)
	return o
}

func (o *ModuleTagCommonQuery) UpdateSelectQuery(query *bun.SelectQuery, dialect schema.Dialect) {
	addOrGroupWhereClause("module_tag_db_id", o.moduleTagDbIDs, false, query, dialect)

	addOrGroupWhereClause("module_tag_id_type", o.moduleTagIDTypes, false, query, dialect)
	addOrGroupWhereClause("module_tag_id", o.moduleTagIDValues, false, query, dialect)
	updateQueryWithEntries(o.moduleTagIDs, query, dialect)

	addOrGroupWhereClause("module_tag_version", o.moduleTagVersions, false, query, dialect)
	addOrGroupWhereClause("language", o.languages, false, query, dialect)
}

func (o *ModuleTagCommonQuery) saveModuleTagDbIDs() {
	if o.saved {
		return
	}

	o.savedIDs = o.moduleTagDbIDs
	o.moduleTagDbIDs = slices.Clone(o.savedIDs)
	o.saved = true
}

func (o *ModuleTagCommonQuery) restoreModuleTagDbIDs() {
	if !o.saved {
		return
	}

	o.moduleTagDbIDs = o.savedIDs
	o.saved = false
}

func (o *ModuleTagCommonQuery) IsEmpty() bool {
	return len(o.moduleTagDbIDs) == 0 &&
		len(o.moduleTagIDTypes) == 0 &&
		len(o.moduleTagIDValues) == 0 &&
		len(o.moduleTagIDs) == 0 &&
		len(o.languages) == 0 &&
		len(o.moduleTagVersions) == 0
}

type whereFunc func(query string, args ...any) *bun.SelectQuery

type timePeriodQueryEntry struct {
	lowerField string
	upperField string
	lower      *time.Time
	upper      *time.Time
	optional   bool
}

func quote(column string, dialect schema.Dialect) string {
	return fmt.Sprintf("%c%s%c", dialect.IdentQuote(), column, dialect.IdentQuote())
}

func (o *timePeriodQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) { // nolint:dupl
	var queryText string
	if o.lower != nil && o.upper != nil { // nolint:gocritic
		if o.optional {
			queryText = fmt.Sprintf(`(%s IS NULL OR %s < ? ) AND (%s IS NULL OR %s > ?)`,
				quote(o.lowerField, dialect), quote(o.lowerField, dialect),
				quote(o.upperField, dialect), quote(o.upperField, dialect),
			)
		} else {
			queryText = fmt.Sprintf(`%s < ? AND %s > ?`,
				quote(o.lowerField, dialect), quote(o.upperField, dialect),
			)
		}

		whereFunc(queryText, *o.lower, *o.upper)
	} else if o.lower != nil {
		if o.optional {
			queryText = fmt.Sprintf(`%s IS NULL OR %s < ?`,
				quote(o.lowerField, dialect), quote(o.lowerField, dialect),
			)
		} else {
			queryText = fmt.Sprintf(`%s < ?`, quote(o.lowerField, dialect))
		}

		whereFunc(queryText, *o.lower)
	} else if o.upper != nil {
		if o.optional {
			queryText = fmt.Sprintf(`%s IS NULL OR %s > ?`,
				quote(o.upperField, dialect), quote(o.upperField, dialect))
		} else {
			queryText = fmt.Sprintf(`%s > ?`, quote(o.upperField, dialect))
		}

		whereFunc(queryText, *o.upper)
	}
}

type timePointQueryEntry struct {
	field    string
	lower    *time.Time
	upper    *time.Time
	optional bool
}

func (o *timePointQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) { // nolint:dupl
	var queryText string
	if o.lower != nil && o.upper != nil { // nolint:gocritic
		if o.optional {
			queryText = fmt.Sprintf(`(%s IS NULL OR %s > ? ) OR (%s IS NULL OR %s < ?)`,
				quote(o.field, dialect), quote(o.field, dialect),
				quote(o.field, dialect), quote(o.field, dialect),
			)
		} else {
			queryText = fmt.Sprintf(`%s > ? AND %s < ?`,
				quote(o.field, dialect), quote(o.field, dialect))
		}

		whereFunc(queryText, *o.lower, *o.upper)
	} else if o.lower != nil {
		if o.optional {
			queryText = fmt.Sprintf(`%s IS NULL OR %s > ?`,
				quote(o.field, dialect), quote(o.field, dialect))
		} else {
			queryText = fmt.Sprintf(`%s > ?`, quote(o.field, dialect))
		}

		whereFunc(queryText, *o.lower)
	} else if o.upper != nil {
		if o.optional {
			queryText = fmt.Sprintf(`%s IS NULL OR %s < ?`,
				quote(o.field, dialect), quote(o.field, dialect))
		} else {
			queryText = fmt.Sprintf(`%s < ?`, quote(o.field, dialect))
		}

		whereFunc(queryText, *o.upper)
	}
}

type nameQueryEntry struct {
	typ   string
	value string
}

func (o *nameQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("name_type = ? AND name = ?", o.typ, o.value)
}

type manifestIDQueryEntry struct {
	typ model.TagIDType
	id  string
}

func (o *manifestIDQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("manifest_id_type = ? AND manifest_id = ?", o.typ, o.id)
}

type moduleTagIDQueryEntry struct {
	typ model.TagIDType
	id  string
}

func (o *moduleTagIDQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("module_tag_id_type = ? AND module_tag_id = ?", o.typ, o.id)
}

type linkedTagIDQueryEntry struct {
	typ model.TagIDType
	id  string
}

func (o *linkedTagIDQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("linked_tag_id_type = ? AND linked_tag_id = ?", o.typ, o.id)
}

type profileQueryEntry struct {
	typ   model.ProfileType
	value string
}

func (o *profileQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("profile_type = ? AND profile = ?", o.typ, o.value)
}

type ownerQueryEntry struct {
	typ string
	id  int64
}

func (o *ownerQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("owner_type = ? AND owner_id = ?", o.typ, o.id)
}

type keyQueryEntry struct {
	typ   string
	bytes []byte
}

func (o *keyQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("key_type = ? AND key_bytes = ?", o.typ, o.bytes)
}

type digestQueryEntry struct {
	algIDInt  int64
	algIDText string
	value     []byte
}

func (o *digestQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	if o.algIDText == "" {
		whereFunc("alg_id_int = ? AND value = ?", o.algIDInt, o.value)
	} else {
		whereFunc("alg_id_text = ? AND value = ?", o.algIDText, o.value)
	}
}

type classIDQueryEntry struct {
	typ   string
	value []byte
}

func (o *classIDQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("class_type = ? AND class_bytes = ?", o.typ, o.value)
}

type instanceQueryEntry struct {
	typ   string
	value []byte
}

func (o *instanceQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("instance_type = ? AND instance_bytes = ?", o.typ, o.value)
}

type groupQueryEntry struct {
	typ   string
	value []byte
}

func (o *groupQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("group_type = ? AND group_bytes = ?", o.typ, o.value)
}

type valueQueryEntry struct {
	valueType  string
	valueBytes *[]byte
	valueText  *string
	valueInt   *int64
}

func newValueQueryEntry(typ string, value any) (*valueQueryEntry, error) {
	ret := &valueQueryEntry{}
	var i int64

	switch t := value.(type) {
	case []byte:
		ret.valueBytes = &t
	case *[]byte:
		ret.valueBytes = t
	case string:
		ret.valueText = &t
	case *string:
		ret.valueText = t
	case int64:
		ret.valueInt = &t
	case *int64:
		ret.valueInt = t
	case int:
		i = int64(t)
		ret.valueInt = &i
	case *int:
		i = int64(*t)
		ret.valueInt = &i

	default:
		return nil, fmt.Errorf("unexpected value: %v (%T)", value, value)
	}

	ret.valueType = typ

	return ret, nil
}

func (o *valueQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	if o.valueBytes != nil { // nolint:gocritic
		whereFunc("value_type = ? AND value_bytes = ?", o.valueType, *o.valueBytes)
	} else if o.valueText != nil {
		whereFunc("value_type = ? AND value_text = ?", o.valueType, *o.valueText)
	} else {
		whereFunc("value_type = ? AND value_int = ?", o.valueType, *o.valueInt)
	}
}

type flagQueryEntry struct {
	codePoint int64
	value     bool
}

func (o *flagQueryEntry) UpdateQuery(whereFunc whereFunc, dialect schema.Dialect) {
	whereFunc("code_point = ? AND value = ?", o.codePoint, o.value)
}

type queryEntry interface {
	UpdateQuery(whereFunc whereFunc, dialect schema.Dialect)
}

func updateQueryWithEntries[T queryEntry](entries []T, query *bun.SelectQuery, dialect schema.Dialect) {
	if len(entries) != 0 {
		query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, value := range entries {
				value.UpdateQuery(q.WhereOr, dialect)
			}

			return q
		})
	}
}

func runQuery[T model.Model](ctx context.Context, db bun.IDB, query Query[T]) ([]T, error) {
	var ret []T
	bunQuery := db.NewSelect().Model(&ret)
	query.UpdateSelectQuery(bunQuery, db.Dialect())

	if err := bunQuery.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoMatch
		}

		return nil, err
	}

	if len(ret) == 0 {
		return ret, ErrNoMatch
	}

	return ret, nil
}

func identQuote(column string, dialect schema.Dialect) string {
	qc := dialect.IdentQuote()
	return fmt.Sprintf("%c%s%c", qc, column, qc)
}

func addOrGroupWhereClause[T any](
	column string,
	values []T,
	exact bool,
	query *bun.SelectQuery,
	dialect schema.Dialect,
) {
	switch len(values) {
	case 0:
		if exact {
			query.Where(fmt.Sprintf(`%s IS NULL`, identQuote(column, dialect)))
		}
	case 1:
		query.Where(fmt.Sprintf(`%s = ?`, identQuote(column, dialect)), values[0])
	default:
		query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			whereText := fmt.Sprintf(`%s = ?`, identQuote(column, dialect))

			for _, value := range values {
				q.WhereOr(whereText, value)
			}

			return q
		})
	}
}
