package model // nolint:dupl

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type TrustDependencyTripleEntry struct {
	bun.BaseModel `bun:"table:trust_dependency_triple_entries,alias:tdte"`

	TripleDbID    int64 `bun:"triple_db_id"`
	ManifestDbID  int64 `bun:"manifest_db_id"`
	ModuleTagDbID int64 `bun:"module_tag_db_id"`
	EnvironmentID int64 `bun:"environment_db_id"`

	IsActive bool

	ManifestIDType TagIDType
	ManifestID     string

	ModuleTagIDType  TagIDType
	ModuleTagID      string
	ModuleTagVersion uint

	Language *string

	Label string `bun:",nullzero"`

	ProfileType ProfileType `bun:",nullzero"`
	Profile     string      `bun:",nullzero"`

	NotBefore *time.Time
	NotAfter  *time.Time
}

func (o *TrustDependencyTripleEntry) DbID() int64 {
	return o.TripleDbID
}

func (o *TrustDependencyTripleEntry) TableName() string {
	return "trust_dependency_triple_entries"
}

func (o *TrustDependencyTripleEntry) IsTable() bool {
	return false
}

func (o *TrustDependencyTripleEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.TripleDbID == 0 {
		return errors.New("TripleDbID not set")
	}

	return db.NewSelect().
		Model(o).
		Where("triple_db_id = ?", o.TripleDbID).
		Scan(ctx)
}

func (o *TrustDependencyTripleEntry) ToManifest(ctx context.Context, db bun.IDB) (*Manifest, error) {
	if o.ManifestDbID == 0 {
		return nil, errors.New("ManifestDbID not set")
	}

	man := &Manifest{ID: o.ManifestDbID}

	if err := man.Select(ctx, db); err != nil {
		return nil, err
	}

	return man, nil
}

func (o *TrustDependencyTripleEntry) ToModuleTag(ctx context.Context, db bun.IDB) (*ModuleTag, error) {
	if o.ModuleTagDbID == 0 {
		return nil, errors.New("ModuleTagDbID not set")
	}

	mt := &ModuleTag{ID: o.ModuleTagDbID}

	if err := mt.Select(ctx, db); err != nil {
		return nil, err
	}

	return mt, nil
}

func (o *TrustDependencyTripleEntry) ToTriple(ctx context.Context, db bun.IDB) (*TrustDependencyTriple, error) {
	if o.TripleDbID == 0 {
		return nil, errors.New("TripleDbID not set")
	}

	triple := &TrustDependencyTriple{ID: o.TripleDbID}

	if err := triple.Select(ctx, db); err != nil {
		return nil, err
	}

	return triple, nil
}

type DomainMembershipTripleEntry struct {
	bun.BaseModel `bun:"table:domain_membership_triple_entries,alias:dmte"`

	TripleDbID    int64 `bun:"triple_db_id"`
	ManifestDbID  int64 `bun:"manifest_db_id"`
	ModuleTagDbID int64 `bun:"module_tag_db_id"`
	EnvironmentID int64 `bun:"environment_db_id"`

	IsActive bool

	ManifestIDType TagIDType
	ManifestID     string

	ModuleTagIDType  TagIDType
	ModuleTagID      string
	ModuleTagVersion uint

	Language *string

	Label string `bun:",nullzero"`

	ProfileType ProfileType `bun:",nullzero"`
	Profile     string      `bun:",nullzero"`

	NotBefore *time.Time
	NotAfter  *time.Time
}

func (o *DomainMembershipTripleEntry) DbID() int64 {
	return o.TripleDbID
}

func (o *DomainMembershipTripleEntry) TableName() string {
	return "domain_membership_triple_entries"
}

func (o *DomainMembershipTripleEntry) IsTable() bool {
	return false
}

func (o *DomainMembershipTripleEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.TripleDbID == 0 {
		return errors.New("TripleDbID not set")
	}

	return db.NewSelect().
		Model(o).
		Where("triple_db_id = ?", o.TripleDbID).
		Scan(ctx)
}

func (o *DomainMembershipTripleEntry) ToManifest(ctx context.Context, db bun.IDB) (*Manifest, error) {
	if o.ManifestDbID == 0 {
		return nil, errors.New("ManifestDbID not set")
	}

	man := &Manifest{ID: o.ManifestDbID}

	if err := man.Select(ctx, db); err != nil {
		return nil, err
	}

	return man, nil
}

func (o *DomainMembershipTripleEntry) ToModuleTag(ctx context.Context, db bun.IDB) (*ModuleTag, error) {
	if o.ModuleTagDbID == 0 {
		return nil, errors.New("ModuleTagDbID not set")
	}

	mt := &ModuleTag{ID: o.ModuleTagDbID}

	if err := mt.Select(ctx, db); err != nil {
		return nil, err
	}

	return mt, nil
}

func (o *DomainMembershipTripleEntry) ToTriple(ctx context.Context, db bun.IDB) (*DomainMembershipTriple, error) {
	if o.TripleDbID == 0 {
		return nil, errors.New("TripleDbID not set")
	}

	triple := &DomainMembershipTriple{ID: o.TripleDbID}

	if err := triple.Select(ctx, db); err != nil {
		return nil, err
	}

	return triple, nil
}
