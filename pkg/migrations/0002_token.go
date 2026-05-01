package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

type token_v1 struct {
	bun.BaseModel `bun:"table:tokens,alias:tok"`

	ID int64 `bun:",pk,autoincrement"`

	ManifestID string
	IsSigned   bool
	Data       []byte

	Authority []*cryptoKey_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:token"`
}

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		var err error

		_, err = db.NewCreateTable().Model((*token_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		var err error

		_, err = db.NewDropTable().Model((*token_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
