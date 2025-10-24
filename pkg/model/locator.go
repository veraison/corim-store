package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
	"github.com/veraison/swid"
)

func LocatorsFromCoRIM(origin *[]corim.Locator) ([]*Locator, error) {
	if origin == nil || len(*origin) == 0 {
		return nil, nil
	}

	ret := make([]*Locator, 0, len(*origin))
	for i, origLocator := range *origin {
		locator, err := NewLocatorFromCoRIM(origLocator)
		if err != nil {
			return nil, fmt.Errorf("locator at index %d: %w", i, err)
		}

		ret = append(ret, locator)
	}

	return ret, nil
}

func LocatorsToCoRIM(origin []*Locator) (*[]corim.Locator, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := make([]corim.Locator, 0, len(origin))
	for i, origLocator := range origin {
		locator, err := origLocator.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("locator at index %d: %w", i, err)
		}

		ret = append(ret, locator)
	}

	return &ret, nil
}

type Locator struct {
	bun.BaseModel `bun:"table:locators,alias:loc"`

	ID int64 `bun:",pk,autoincrement"`

	Href       string
	Thumbprint []*Digest `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:locator"`

	ManifestID int64
}

func NewLocatorFromCoRIM(origin corim.Locator) (*Locator, error) {
	var ret Locator

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Locator) ToCoRIM() (corim.Locator, error) {
	ret := corim.Locator{
		Href: comid.TaggedURI(o.Href),
	}

	if len(o.Thumbprint) == 1 {
		ret.Thumbprint = &swid.HashEntry{
			HashAlgID: o.Thumbprint[0].AlgID,
			HashValue: o.Thumbprint[0].Value,
		}
	} else if len(o.Thumbprint) > 1 {
		return corim.Locator{}, errors.New("multiple digests not supported by corim library")
	}

	return ret, nil
}

func (o *Locator) FromCoRIM(origin corim.Locator) error {
	o.Href = string(origin.Href)

	if origin.Thumbprint != nil {
		o.Thumbprint = []*Digest{
			NewDigest(origin.Thumbprint.HashAlgID, origin.Thumbprint.HashValue),
		}
	}

	return nil
}

func (o *Locator) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, digest := range o.Thumbprint {
		digest.OwnerID = o.ID
		digest.OwnerType = "locator"

		if err := digest.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting digest %d: %w", i, err)
		}
	}

	return nil
}

func (o *Locator) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Thumbprint").
		Where("loc.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *Locator) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, digest := range o.Thumbprint {
		if err := digest.Delete(ctx, db); err != nil {
			return fmt.Errorf("digest at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
