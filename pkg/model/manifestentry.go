package model

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type ManifestEntry struct {
	bun.BaseModel `bun:"table:manifest_entries,alias:mte"`

	ManifestDbID int64 `bun:"manifest_db_id"`

	ManifestIDType TagIDType
	ManifestID     string

	Label string `bun:",nullzero"`

	ProfileType ProfileType `bun:",nullzero"`
	Profile     string      `bun:",nullzero"`

	NotBefore *time.Time
	NotAfter  *time.Time
}

func (o *ManifestEntry) DbID() int64 {
	return o.ManifestDbID
}

func (o *ManifestEntry) TableName() string {
	return "manifest_entries"
}

func (o *ManifestEntry) IsTable() bool {
	return false
}

func (o *ManifestEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.ManifestDbID == 0 {
		return errors.New("ManifestDbID not set")
	}

	return db.NewSelect().
		Model(o).
		Where("manifest_db_id = ?", o.ManifestDbID).
		Scan(ctx)
}

func (o *ManifestEntry) ToManifest(ctx context.Context, db bun.IDB) (*Manifest, error) {
	if o.ManifestDbID == 0 {
		return nil, errors.New("ManifestDbID not set")
	}

	man := &Manifest{ID: o.ManifestDbID}

	if err := man.Select(ctx, db); err != nil {
		return nil, err
	}

	return man, nil
}
