package model // nolint:dupl

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type KeyTripleEntry struct {
	bun.BaseModel `bun:"table:key_triple_entries,alias:vte"`

	TripleDbID    int64 `bun:"triple_db_id"`
	ManifestDbID  int64 `bun:"manifest_db_id"`
	ModuleTagDbID int64 `bun:"module_tag_db_id"`
	EnvironmentID int64 `bun:"environment_db_id"`

	TripleType KeyTripleType

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

func (o *KeyTripleEntry) DbID() int64 {
	return o.TripleDbID
}

func (o *KeyTripleEntry) TableName() string {
	return "key_triple_entries"
}

func (o *KeyTripleEntry) IsTable() bool {
	return false
}

func (o *KeyTripleEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.TripleDbID == 0 {
		return errors.New("TripleDbID not set")
	}

	return db.NewSelect().
		Model(o).
		Where("triple_db_id = ?", o.TripleDbID).
		Scan(ctx)
}

func (o *KeyTripleEntry) Manifest(ctx context.Context, db bun.IDB) (*Manifest, error) {
	if o.ManifestDbID == 0 {
		return nil, errors.New("ManifestDbID not set")
	}

	man := &Manifest{ID: o.ManifestDbID}

	if err := man.Select(ctx, db); err != nil {
		return nil, err
	}

	return man, nil
}

func (o *KeyTripleEntry) ModuleTag(ctx context.Context, db bun.IDB) (*ModuleTag, error) {
	if o.ModuleTagDbID == 0 {
		return nil, errors.New("ModuleTagDbID not set")
	}

	mt := &ModuleTag{ID: o.ModuleTagDbID}

	if err := mt.Select(ctx, db); err != nil {
		return nil, err
	}

	return mt, nil
}

func (o *KeyTripleEntry) ToTriple(ctx context.Context, db bun.IDB) (*KeyTriple, error) {
	if o.TripleDbID == 0 {
		return nil, errors.New("TripleDbID not set")
	}

	triple := &KeyTriple{ID: o.TripleDbID}

	if err := triple.Select(ctx, db); err != nil {
		return nil, err
	}

	return triple, nil
}
