package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func DomainEntriesFromCoRIM(origin []comid.Environment) ([]*DomainEntry, error) {
	ret := make([]*DomainEntry, 0, len(origin))

	for i, origEnv := range origin {
		env, err := NewEnvironmentFromCoRIM(&origEnv)
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("entry[%d]: %w", i, err)
		}

		ret = append(ret, &DomainEntry{Environment: env})
	}

	return ret, nil
}

func DomainEntriesToCoRIM(origin []*DomainEntry) ([]comid.Environment, error) {
	ret := make([]comid.Environment, 0, len(origin))

	for i, entry := range origin {
		env, err := entry.Environment.ToCoRIM()
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("entry[%d]: %w", i, err)
		}

		ret = append(ret, *env)
	}

	return ret, nil
}

type DomainEntry struct {
	bun.BaseModel `bun:"table:domain_entries,alias:de"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	Environment   *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

func (o *DomainEntry) DbID() int64 {
	return o.ID
}

func (o *DomainEntry) OwnerDbID() int64 {
	return o.OwnerID
}

func (o DomainEntry) OwnerName() string {
	return o.OwnerType
}

func (o *DomainEntry) TableName() string {
	return "domain_entries"
}

func (o *DomainEntry) IsTable() bool {
	return true
}

func (o *DomainEntry) Validate() error {
	if o.Environment == nil {
		return errors.New("no environment")
	}

	return o.Environment.Validate()
}

func (o *DomainEntry) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if err := o.Environment.Insert(ctx, db); err != nil {
		return err
	}
	o.EnvironmentID = o.Environment.ID

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (o *DomainEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().
		Model(o).
		Relation("Environment").
		Where("de.id = ?", o.ID).
		Scan(ctx)
}

func (o *DomainEntry) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		return err
	}

	if o.Environment != nil {
		if err := o.Environment.DeleteIfOrphaned(ctx, db); err != nil {
			return fmt.Errorf("environment: %w", err)
		}
	}

	return nil
}

func DomainMembershipTriplesFromCoRIM(origin *comid.DomainMembershipTriples) ([]*DomainMembershipTriple, error) {
	if origin == nil || len(*origin) == 0 {
		return nil, nil
	}

	ret := make([]*DomainMembershipTriple, 0, len(*origin))

	for i, originTriple := range *origin {
		triple, err := NewDomainMembershipTripleFromCoRIM(&originTriple)
		if err != nil {
			return nil, fmt.Errorf("domain membership triple[%d]: %w", i, err)
		}

		ret = append(ret, triple)
	}

	return ret, nil
}

func DomainMembershipTriplesToCoRIM(origin []*DomainMembershipTriple) (*comid.DomainMembershipTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewDomainMebershipTriples()

	for i, originTriple := range origin {
		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("domain membership triple[%d]: %w", i, err)
		}

		ret.Add(*triple)
	}

	return ret, nil
}

type DomainMembershipTriple struct {
	bun.BaseModel `bun:"table:domain_membership_triples,alias:dmt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	DomainID      *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Members []*DomainEntry `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:domain_membership_triple"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

func NewDomainMembershipTripleFromCoRIM(origin *comid.DomainMembershipTriple) (*DomainMembershipTriple, error) {
	var ret DomainMembershipTriple

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectDomainMembershipTriple(ctx context.Context, db bun.IDB, id int64) (*DomainMembershipTriple, error) {
	ret := DomainMembershipTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *DomainMembershipTriple) DbID() int64 {
	return o.ID
}

func (o *DomainMembershipTriple) TableName() string {
	return "domain_membership_triples"
}

func (o *DomainMembershipTriple) IsTable() bool {
	return true
}

func (o *DomainMembershipTriple) OwnerDbID() int64 {
	return o.ModuleID
}

func (o DomainMembershipTriple) OwnerName() string {
	return "module_tag"
}

func (o *DomainMembershipTriple) FromCoRIM(origin *comid.DomainMembershipTriple) error {
	env, err := NewEnvironmentFromCoRIM(&origin.DomainID)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("domain ID: %w", err)
	}

	entries, err := DomainEntriesFromCoRIM(origin.Members)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("members: %w", err)
	}

	o.DomainID = env
	o.Members = entries

	return nil
}

func (o *DomainMembershipTriple) ToCoRIM() (*comid.DomainMembershipTriple, error) {
	var ret comid.DomainMembershipTriple

	env, err := o.DomainID.ToCoRIM()
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("environment: %w", err)
	}
	ret.DomainID = *env

	ret.Members, err = DomainEntriesToCoRIM(o.Members)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("members: %w", err)
	}

	return &ret, nil
}

func (o *DomainMembershipTriple) Validate() error {
	if o.DomainID == nil {
		return errors.New("domain ID not set")
	}

	if err := o.DomainID.Validate(); err != nil {
		return fmt.Errorf("domain ID: %w", err)
	}

	if len(o.Members) == 0 {
		return errors.New("no members")
	}

	for i, entry := range o.Members {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("member[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainMembershipTriple) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if err := o.DomainID.Insert(ctx, db); err != nil {
		// coverage:ignore
		return err
	}
	o.EnvironmentID = o.DomainID.ID

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	for i, entry := range o.Members {
		entry.OwnerID = o.ID
		entry.OwnerType = "domain_membership_triple"

		if err := entry.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("member[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainMembershipTriple) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("DomainID").
		Relation("Members").
		Where("dmt.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, entry := range o.Members {
		if err := entry.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("member[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainMembershipTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, entry := range o.Members {
		if err := entry.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("member[%d]: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	if o.DomainID != nil {
		if err := o.DomainID.DeleteIfOrphaned(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("domain ID: %w", err)
		}
	}

	return nil
}

func DomainDependencyTriplesFromCoRIM(origin *comid.DomainDependencyTriples) ([]*DomainDependencyTriple, error) {
	if origin == nil || len(*origin) == 0 {
		return nil, nil
	}

	ret := make([]*DomainDependencyTriple, 0, len(*origin))

	for i, originTriple := range *origin {
		triple, err := NewDomainDependencyTripleFromCoRIM(&originTriple)
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("domain dependency triple[%d]: %w", i, err)
		}

		ret = append(ret, triple)
	}

	return ret, nil
}

func DomainDependencyTriplesToCoRIM(origin []*DomainDependencyTriple) (*comid.DomainDependencyTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	var ret comid.DomainDependencyTriples

	for i, originTriple := range origin {
		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("domain dependency triple[%d]: %w", i, err)
		}

		ret = append(ret, *triple)
	}

	return &ret, nil
}

type DomainDependencyTriple struct {
	bun.BaseModel `bun:"table:domain_dependency_triples,alias:ddt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	DomainID      *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Trustees []*DomainEntry `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:domain_dependency_triple"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

func NewDomainDependencyTripleFromCoRIM(origin *comid.DomainDependencyTriple) (*DomainDependencyTriple, error) {
	var ret DomainDependencyTriple

	if err := ret.FromCoRIM(origin); err != nil {
		// coverage:ignore
		return nil, err
	}

	return &ret, nil
}

func SelectDomainDependencyTriple(ctx context.Context, db bun.IDB, id int64) (*DomainDependencyTriple, error) {
	ret := DomainDependencyTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *DomainDependencyTriple) DbID() int64 {
	return o.ID
}

func (o *DomainDependencyTriple) TableName() string {
	return "domain_dependency_triples"
}

func (o *DomainDependencyTriple) IsTable() bool {
	return true
}

func (o *DomainDependencyTriple) OwnerDbID() int64 {
	return o.ModuleID
}

func (o DomainDependencyTriple) OwnerName() string {
	return "module_tag"
}

func (o *DomainDependencyTriple) FromCoRIM(origin *comid.DomainDependencyTriple) error {
	env, err := NewEnvironmentFromCoRIM(&origin.DomainID)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("domain ID: %w", err)
	}

	entries, err := DomainEntriesFromCoRIM(origin.Trustees)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("trustees: %w", err)
	}

	o.DomainID = env
	o.Trustees = entries

	return nil
}

func (o *DomainDependencyTriple) ToCoRIM() (*comid.DomainDependencyTriple, error) {
	var ret comid.DomainDependencyTriple

	env, err := o.DomainID.ToCoRIM()
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("environment: %w", err)
	}
	ret.DomainID = *env

	ret.Trustees, err = DomainEntriesToCoRIM(o.Trustees)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("trustees: %w", err)
	}

	return &ret, nil
}

func (o *DomainDependencyTriple) Validate() error {
	if o.DomainID == nil {
		return errors.New("domain ID not set")
	}

	if err := o.DomainID.Validate(); err != nil {
		return fmt.Errorf("domain ID: %w", err)
	}

	if len(o.Trustees) == 0 {
		return errors.New("no trustees")
	}

	for i, entry := range o.Trustees {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("trustee[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainDependencyTriple) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if err := o.DomainID.Insert(ctx, db); err != nil {
		// coverage:ignore
		return err
	}
	o.EnvironmentID = o.DomainID.ID

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	for i, entry := range o.Trustees {
		entry.OwnerID = o.ID
		entry.OwnerType = "domain_dependency_triple"

		if err := entry.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("trustee[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainDependencyTriple) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("DomainID").
		Relation("Trustees").
		Where("ddt.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, entry := range o.Trustees {
		if err := entry.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("trustee[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *DomainDependencyTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, entry := range o.Trustees {
		if err := entry.Delete(ctx, db); err != nil {
			return fmt.Errorf("trustee[%d]: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	if o.DomainID != nil {
		if err := o.DomainID.DeleteIfOrphaned(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("domain ID: %w", err)
		}
	}

	return nil
}

func init() {
	environmentOwners = append(environmentOwners,
		"domain_entries",
		"domain_dependency_triples",
		"domain_membership_triples",
	)
}
