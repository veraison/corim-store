package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func DigestsFromCoRIM(origin *comid.Digests) ([]*Digest, error) {
	if origin == nil {
		return nil, nil
	}

	ret := make([]*Digest, 0, len(*origin))

	for _, digest := range *origin {
		if digest.Algorithm.IsString() {
			ret = append(ret, NewDigestText(digest.Algorithm.String(), digest.Value))
		} else {
			ret = append(ret, NewDigestInt(int64(digest.Algorithm.Int()), digest.Value))
		}
	}

	return ret, nil
}

func DigestsToCoRIM(origin []*Digest) (*comid.Digests, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := make(comid.Digests, 0, len(origin))

	for i, digest := range origin {
		var corimDigest comid.Digest
		if digest.AlgIDText != "" {
			corimDigest = *comid.NewDigestStringAlg(digest.AlgIDText, digest.Value)
		} else {
			corimDigest = *comid.NewDigestIntAlg(int(digest.AlgIDInt), digest.Value)
		}

		if err := corimDigest.Valid(); err != nil {
			return nil, fmt.Errorf("digest[%d]: %w", i, err)
		}

		ret = append(ret, corimDigest)
	}

	return &ret, nil
}

type Digest struct {
	bun.BaseModel `bun:"table:digests,alias:dgt"`

	ID int64 `bun:",pk,autoincrement"`

	AlgIDInt  int64  `bun:"alg_id_int"`
	AlgIDText string `bun:"alg_id_text"`
	Value     []byte `bun:"value"`

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

func NewDigestInt(alg_id int64, val []byte) *Digest {
	return &Digest{
		AlgIDInt: alg_id,
		Value:    val,
	}
}

func NewDigestText(alg_id string, val []byte) *Digest {
	return &Digest{
		AlgIDText: alg_id,
		Value:     val,
	}
}

func (o *Digest) DbID() int64 {
	return o.ID
}

func (o *Digest) TableName() string {
	return "digests"
}

func (o *Digest) IsTable() bool {
	return true
}

func (o *Digest) AlgID() any {
	if o.AlgIDInt != 0 {
		return o.AlgIDInt
	}

	return o.AlgIDText
}

func (o *Digest) AlgIDString() string {
	if o.AlgIDText != "" {
		return o.AlgIDText
	}

	return strconv.Itoa(int(o.AlgIDInt))
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
