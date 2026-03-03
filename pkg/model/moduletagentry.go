package model

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type ModuleTagEntry struct {
	bun.BaseModel `bun:"table:module_tag_entries,alias:mte"`

	ManifestDbID  int64 `bun:"manifest_db_id"`
	ModuleTagDbID int64 `bun:"module_tag_db_id"`

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

func (o *ModuleTagEntry) DbID() int64 {
	return o.ModuleTagDbID
}

func (o *ModuleTagEntry) TableName() string {
	return "module_tag_entries"
}

func (o *ModuleTagEntry) IsTable() bool {
	return false
}

func (o *ModuleTagEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.ModuleTagDbID == 0 {
		return errors.New("ModuleTagDbID not set")
	}

	return db.NewSelect().
		Model(o).
		Where("module_tag_db_id = ?", o.ModuleTagDbID).
		Scan(ctx)
}

func (o *ModuleTagEntry) ToManifest(ctx context.Context, db bun.IDB) (*Manifest, error) {
	if o.ManifestDbID == 0 {
		return nil, errors.New("ManifestDbID not set")
	}

	man := &Manifest{ID: o.ManifestDbID}

	if err := man.Select(ctx, db); err != nil {
		return nil, err
	}

	return man, nil
}

func (o *ModuleTagEntry) ToModuleTag(ctx context.Context, db bun.IDB) (*ModuleTag, error) {
	if o.ModuleTagDbID == 0 {
		return nil, errors.New("ModuleTagDbID not set")
	}

	mt := &ModuleTag{ID: o.ModuleTagDbID}

	if err := mt.Select(ctx, db); err != nil {
		return nil, err
	}

	return mt, nil
}
