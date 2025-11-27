package model

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

func DigestsFromCoRIM(origin *comid.Digests) ([]*Digest, error) {
	if origin == nil {
		return nil, nil
	}

	ret := make([]*Digest, 0, len(*origin))

	for _, hashEntry := range *origin {
		ret = append(ret, &Digest{AlgID: hashEntry.HashAlgID, Value: hashEntry.HashValue})
	}

	return ret, nil
}

func DigestsToCoRIM(origin []*Digest) (*comid.Digests, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := make(comid.Digests, 0, len(origin))

	for _, digest := range origin {
		ret = append(ret, swid.HashEntry{HashAlgID: digest.AlgID, HashValue: digest.Value})
	}

	return &ret, nil
}

type Digest struct {
	bun.BaseModel `bun:"table:digests,alias:dgt"`

	ID int64 `bun:",pk,autoincrement"`

	AlgID uint64
	Value []byte

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

func NewDigest(alg_id uint64, val []byte) *Digest {
	return &Digest{
		AlgID: alg_id,
		Value: val,
	}
}

func (o *Digest) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *Digest) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("id = ?", o.ID).Scan(ctx)
}

func (o *Digest) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
