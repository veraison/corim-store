package model

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/swid"
)

type ModuleTag struct {
	bun.BaseModel `bun:"table:module_tags,alias:mt"`

	ID int64 `bun:",pk,autoincrement"`

	TagIDType  TagIDType
	TagID      string
	TagVersion uint

	Language *string

	Entities []*Entity `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`

	ValueTriples []*ValueTriple `bun:"rel:has-many,join:id=module_id"`
	KeyTriples   []*KeyTriple   `bun:"rel:has-many,join:id=module_id"`

	LinkedTags []*LinkedTag `bun:"rel:has-many,join:id=module_id"`

	Extensions        []*ExtensionValue `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`
	TriplesExtensions []*ExtensionValue `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:triples"`

	ManifestID int64
}

func NewModuleTagFromCoRIM(origin *comid.Comid) (*ModuleTag, error) {
	var ret ModuleTag

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectModuleTag(ctx context.Context, db bun.IDB, id int64) (*ModuleTag, error) {
	var ret ModuleTag

	ret.ID = id
	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *ModuleTag) FromCoRIM(origin *comid.Comid) error {
	var err error

	if origin.Triples.CondEndorseSeries != nil && len(origin.Triples.CondEndorseSeries.Values) != 0 {
		return errors.New("conditional endosement series are not supported") // TODO
	}

	o.Language = origin.Language
	o.TagID = origin.TagIdentity.TagID.String()
	o.TagVersion = origin.TagIdentity.TagVersion

	// swid.TagID does not expose the underlying type in any way, but we
	// need it to correctly reconstruct it inside ToCoRIM(), so we guess
	// by seeing if the ID parses as a valid UUID.
	if _, err = uuid.Parse(o.TagID); err == nil {
		o.TagIDType = UUIDTagID
	} else {
		o.TagIDType = StringTagID
	}

	o.Entities, err = CoMIDEntitiesFromCoRIM(origin.Entities)
	if err != nil {
		return err
	}

	o.LinkedTags, err = LinkedTagsFromCoRIM(origin.LinkedTags)
	if err != nil {
		return err
	}

	o.ValueTriples, err = ValueTriplesFromCoRIM(origin.Triples.ReferenceValues, ReferenceValueTriple)
	if err != nil {
		return err
	}

	endTriples, err := ValueTriplesFromCoRIM(origin.Triples.EndorsedValues, EndorsedValueTriple)
	if err != nil {
		return err
	}
	o.ValueTriples = append(o.ValueTriples, endTriples...)

	o.KeyTriples, err = KeyTriplesFromCoRIM(origin.Triples.AttestVerifKeys, AttestKeyTriple)
	if err != nil {
		return err
	}

	identTriples, err := KeyTriplesFromCoRIM(origin.Triples.DevIdentityKeys, IdentityKeyTriple)
	if err != nil {
		return err
	}
	o.KeyTriples = append(o.KeyTriples, identTriples...)

	o.Extensions, err = CoMIDExtensionsFromCoRIM(origin.Extensions)
	if err != nil {
		return err
	}

	o.TriplesExtensions, err = CoMIDExtensionsFromCoRIM(origin.Triples.Extensions)
	if err != nil {
		return err
	}

	return nil
}

func (o *ModuleTag) ToCoRIM() (*comid.Comid, error) {
	ret := comid.NewComid()

	ret.Language = o.Language

	var tagID *swid.TagID
	var err error

	switch o.TagIDType {
	case StringTagID:
		tagID = swid.NewTagID(o.TagID)
	case UUIDTagID:
		tagUUID, err := uuid.Parse(o.TagID)
		if err != nil {
			return nil, err
		}

		tagID = swid.NewTagID(tagUUID)
	default:
		return nil, fmt.Errorf("unexpected tag ID type: %s", o.TagIDType)
	}

	if tagID == nil {
		return nil, fmt.Errorf("could not create swid.TagID from %q", o.TagID)
	}

	ret.TagIdentity.TagID = *tagID
	ret.TagIdentity.TagVersion = o.TagVersion

	ret.Entities, err = CoMIDEntitiesToCoRIM(o.Entities)
	if err != nil {
		return nil, err
	}

	ret.LinkedTags, err = LinkedTagsToCoRIM(o.LinkedTags)
	if err != nil {
		return nil, err
	}

	ret.Triples.ReferenceValues, err = ValueTriplesToCoRIM(o.ValueTriples, ReferenceValueTriple)
	if err != nil {
		return nil, err
	}

	ret.Triples.EndorsedValues, err = ValueTriplesToCoRIM(o.ValueTriples, EndorsedValueTriple)
	if err != nil {
		return nil, err
	}

	ret.Triples.AttestVerifKeys, err = KeyTriplesToCoRIM(o.KeyTriples, AttestKeyTriple)
	if err != nil {
		return nil, err
	}

	ret.Triples.DevIdentityKeys, err = KeyTriplesToCoRIM(o.KeyTriples, IdentityKeyTriple)
	if err != nil {
		return nil, err
	}

	ret.Extensions, err = CoMIDExtensionsToCoRIM(o.Extensions)
	if err != nil {
		return nil, err
	}

	ret.Triples.Extensions, err = CoMIDExtensionsToCoRIM(o.TriplesExtensions)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *ModuleTag) SetActive(value bool) {
	for _, kt := range o.KeyTriples {
		kt.IsActive = value
	}

	for _, vt := range o.ValueTriples {
		vt.IsActive = value
	}
}

func (o *ModuleTag) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for _, entity := range o.Entities {
		entity.OwnerID = o.ID
		entity.OwnerType = "module_tag"

		if err := entity.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting linked tag %+v: %w", entity, err)
		}
	}

	for _, link := range o.LinkedTags {
		link.ModuleID = o.ID

		if err := link.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting linked tag %+v: %w", link, err)
		}
	}

	for _, triple := range o.ValueTriples {
		triple.ModuleID = o.ID

		if err := triple.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting value triple %+v: %w", triple, err)
		}
	}

	for _, triple := range o.KeyTriples {
		triple.ModuleID = o.ID

		if err := triple.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting key triple %+v: %w", triple, err)
		}
	}

	for _, ext := range o.Extensions {
		ext.OwnerID = o.ID
		ext.OwnerType = "module_tag"

		if err := ext.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting extension %+v: %w", ext, err)
		}
	}

	for _, ext := range o.TriplesExtensions {
		ext.OwnerID = o.ID
		ext.OwnerType = "triples"

		if err := ext.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting extension %+v: %w", ext, err)
		}
	}

	return nil
}

func (o *ModuleTag) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Entities").
		Relation("LinkedTags").
		Relation("ValueTriples").
		Relation("KeyTriples").
		Relation("Extensions").
		Relation("TriplesExtensions").
		Where("mt.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, entity := range o.Entities {
		if err := entity.Select(ctx, db); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	for i, triple := range o.ValueTriples {
		if err := triple.Select(ctx, db); err != nil {
			return fmt.Errorf("value triple at index %d: %w", i, err)
		}
	}

	for i, triple := range o.KeyTriples {
		if err := triple.Select(ctx, db); err != nil {
			return fmt.Errorf("key triple at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *ModuleTag) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, tag := range o.LinkedTags {
		if err := tag.Delete(ctx, db); err != nil {
			return fmt.Errorf("linked tag at index %d: %w", i, err)
		}
	}

	for i, entity := range o.Entities {
		if err := entity.Delete(ctx, db); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	for i, triple := range o.ValueTriples {
		if err := triple.Delete(ctx, db); err != nil {
			return fmt.Errorf("value triple at index %d: %w", i, err)
		}
	}

	for i, triple := range o.KeyTriples {
		if err := triple.Delete(ctx, db); err != nil {
			return fmt.Errorf("key triple at index %d: %w", i, err)
		}
	}

	for i, extension := range o.Extensions {
		if err := extension.Delete(ctx, db); err != nil {
			return fmt.Errorf("extension at index %d: %w", i, err)
		}
	}

	for i, extension := range o.TriplesExtensions {
		if err := extension.Delete(ctx, db); err != nil {
			return fmt.Errorf("triples extension at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *ModuleTag) Validate() error {
	if len(o.TagIDType) == 0 || len(o.TagID) == 0 {
		return errors.New("tag ID not set (both type and value must be set)")
	}

	supprtedIDTypes := []TagIDType{StringTagID, UUIDTagID}
	if !slices.Contains(supprtedIDTypes, o.TagIDType) {
		return fmt.Errorf("unsupported tag ID type: %s", o.TagIDType)
	}

	if len(o.ValueTriples) == 0 && len(o.KeyTriples) == 0 {
		return errors.New("no triples specified")
	}

	for i, entity := range o.Entities {
		if err := entity.Validate(); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	for i, valueTriple := range o.ValueTriples {
		if err := valueTriple.Validate(); err != nil {
			return fmt.Errorf("value triple at index %d: %w", i, err)
		}
	}

	for i, keyTriple := range o.KeyTriples {
		if err := keyTriple.Validate(); err != nil {
			return fmt.Errorf("key triple at index %d: %w", i, err)
		}
	}

	return nil
}
