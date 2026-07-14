package model

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func HrefsFromCoRIM(origin corim.OneOrMore[comid.TaggedURI]) []*Href {
	ret := make([]*Href, len(origin))

	for i, uri := range origin {
		ret[i] = &Href{Value: uri.String()}
	}

	return ret
}

func HrefsToCoRIM(origin []*Href) corim.OneOrMore[comid.TaggedURI] {
	ret := make(corim.OneOrMore[comid.TaggedURI], len(origin))

	for i, href := range origin {
		ret[i] = comid.TaggedURI(href.Value)
	}

	return ret
}

type Href struct {
	bun.BaseModel `bun:"table:hrefs,alias:href"`

	ID int64 `bun:",pk,autoincrement"`

	Value string `bun:"value"`

	LocatorID int64 `bun:"locator_id"`
}

func (o *Href) DbID() int64 {
	return o.ID
}

func (o *Href) OwnerDbID() int64 {
	return o.LocatorID
}

func (o *Href) OwnerName() string {
	return "locator"
}

func (o *Href) TableName() string {
	return "hrefs"
}

func (o *Href) IsTable() bool {
	return true
}

func (o *Href) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (o *Href) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Where("id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *Href) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
