package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

type TagRelation string

const (
	SupplementsRelation TagRelation = "supplements"
	ReplacesRelation    TagRelation = "replaces"
)

func LinkedTagsFromCoRIM(origin *comid.LinkedTags) ([]*LinkedTag, error) {
	if origin == nil || len(*origin) == 0 {
		return nil, nil
	}

	ret := make([]*LinkedTag, 0, len(*origin))

	for i, originLink := range *origin {
		link, err := NewLinkedTagFromCoRIM(&originLink)
		if err != nil {
			return nil, fmt.Errorf("could not convert linked tag at index %d: %w", i, err)
		}

		ret = append(ret, link)
	}

	return ret, nil
}

func LinkedTagsToCoRIM(origin []*LinkedTag) (*comid.LinkedTags, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewLinkedTags()
	for i, originLink := range origin {
		link, err := originLink.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("could not convert linked tag at index %d: %w", i, err)
		}

		ret.AddLinkedTag(*link)
	}

	return ret, nil
}

func NewLinkedTagFromCoRIM(origin *comid.LinkedTag) (*LinkedTag, error) {
	var ret LinkedTag

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectLinkedTag(ctx context.Context, db bun.IDB, id int64) (*LinkedTag, error) {
	var ret LinkedTag
	ret.ID = id

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

type LinkedTag struct {
	bun.BaseModel `bun:"table:linked_tags,alias:lnk"`

	ID int64 `bun:",pk,autoincrement"`

	LinkedTagIDType TagIDType
	LinkedTagID     string
	TagRelation     TagRelation

	ModuleID int64 `bun:",nullzero"`
}

func (o *LinkedTag) FromCoRIM(origin *comid.LinkedTag) error {
	o.LinkedTagID = origin.LinkedTagID.String()

	// swid.TagID does not expose the underlying type in any way, but we
	// need it to correctly reconstruct it inside ToCoRIM(), so we guess
	// by seeing if the ID parses as a valid UUID.
	if _, err := uuid.Parse(o.LinkedTagID); err == nil {
		o.LinkedTagIDType = UUIDTagID
	} else {
		o.LinkedTagIDType = StringTagID
	}

	switch origin.Rel {
	case comid.RelSupplements:
		o.TagRelation = SupplementsRelation
	case comid.RelReplaces:
		o.TagRelation = ReplacesRelation
	default:
		return fmt.Errorf("unexpected tag relation: %d", origin.Rel)
	}

	return nil
}

func (o *LinkedTag) ToCoRIM() (*comid.LinkedTag, error) {
	var tagID *swid.TagID

	switch o.LinkedTagIDType {
	case StringTagID:
		tagID = swid.NewTagID(o.LinkedTagID)
	case UUIDTagID:
		tagUUID, err := uuid.Parse(o.LinkedTagID)
		if err != nil {
			return nil, err
		}

		tagID = swid.NewTagID(tagUUID)
	default:
		return nil, fmt.Errorf("unexpected linked tag ID type: %s", o.LinkedTagIDType)
	}

	if tagID == nil {
		return nil, fmt.Errorf("could not create swid.TagID from %q", o.LinkedTagID)
	}

	ret := comid.LinkedTag{LinkedTagID: *tagID}

	switch o.TagRelation {
	case SupplementsRelation:
		ret.Rel = comid.RelSupplements
	case ReplacesRelation:
		ret.Rel = comid.RelReplaces
	default:
		return nil, fmt.Errorf("unexpected tag relation: %s", o.TagRelation)
	}

	return &ret, nil
}

func (o *LinkedTag) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("lnk.id = ?", o.ID).Scan(ctx)
}

func (o *LinkedTag) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *LinkedTag) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
