package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

type Token struct {
	bun.BaseModel `bun:"table:tokens,alias:tok"`

	ID int64 `bun:",pk,autoincrement"`

	ManifestID string `bun:",unique"`
	IsSigned   bool
	Data       []byte

	Authority []*CryptoKey `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:token"`
}

func (o *Token) DbID() int64 {
	return o.ID
}

func (o *Token) TableName() string {
	return "tokens"
}

func (o *Token) IsTable() bool {
	return true
}

func (o *Token) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("tok.id = ?", o.ID).Scan(ctx)
}

func (o *Token) Insert(ctx context.Context, db bun.IDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		// coverage:ignore
		return err
	}

	if _, err := tx.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		_ = tx.Rollback()
		return err
	}

	for _, ck := range o.Authority {
		ck.OwnerID = o.ID
		ck.OwnerType = "token"

		if err := ck.Insert(ctx, tx); err != nil {
			// coverage:ignore
			_ = tx.Rollback()
			return fmt.Errorf("error inserting crypto key %+v: %w", ck, err)
		}
	}

	return tx.Commit()
}

func (o *Token) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		// coverage:ignore
		return err
	}

	for _, ck := range o.Authority {
		if err := ck.Delete(ctx, tx); err != nil {
			// coverage:ignore
			_ = tx.Rollback()
			return err
		}
	}

	if _, err := tx.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		// coverage:ignore
		_ = tx.Rollback()
		return err

	}

	return tx.Commit()
}
