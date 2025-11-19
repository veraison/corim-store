package model

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
	"github.com/veraison/eat"
	"github.com/veraison/swid"
)

type ManifestIDType string

type ProfileType string

type Manifest struct {
	bun.BaseModel `bun:"table:manifests,alias:man"`

	ID int64 `bun:",pk,autoincrement"`

	ManifestIDType TagIDType
	ManifestID     string

	Digest    []byte
	TimeAdded time.Time
	Label     string `bun:",nullzero"`

	ProfileType ProfileType `bun:",nullzero"`
	Profile     string      `bun:",nullzero"`

	Entities      []*Entity  `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:manifest"`
	DependentRIMs []*Locator `bun:"rel:has-many,join:id=manifest_id"`

	NotBefore *time.Time
	NotAfter  *time.Time

	ModuleTags []*ModuleTag `bun:"rel:has-many,join:id=manifest_id"`

	Extensions []*ExtensionValue `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`
}

func NewManifestFromCoRIM(origin *corim.UnsignedCorim) (*Manifest, error) {
	var ret Manifest

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectManifest(ctx context.Context, db bun.IDB, id int64) (*Manifest, error) {
	var ret Manifest

	ret.ID = id
	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Manifest) FromCoRIM(origin *corim.UnsignedCorim) error {
	var err error

	o.ManifestID = origin.ID.String()

	// swid.TagID does not expose the underlying type in any way, but we
	// need it to correctly reconstruct it inside ToCoRIM(), so we guess
	// by seeing if the ID parses as a valid UUID.
	if _, err = uuid.Parse(o.ManifestID); err == nil {
		o.ManifestIDType = UUIDTagID
	} else {
		o.ManifestIDType = StringTagID
	}

	if origin.Profile != nil {
		o.Profile, err = origin.Profile.Get()
		if err != nil {
			return fmt.Errorf("profile: %w", err)
		}

		if origin.Profile.IsOID() { // nolint:gocritic
			o.ProfileType = "oid"
		} else if origin.Profile.IsURI() {
			o.ProfileType = "uri"
		} else {
			return fmt.Errorf("invalid profile in origin: %+v", origin.Profile)
		}
	}

	o.DependentRIMs, err = LocatorsFromCoRIM(origin.DependentRims)
	if err != nil {
		return err
	}

	o.Entities, err = CoRIMEntitiesFromCoRIM(origin.Entities)
	if err != nil {
		return err
	}

	for i, tag := range origin.Tags {
		if tag.Number != corim.ComidTag {
			return fmt.Errorf(
				"tag %d at index %d; only CoMID tags (%d) are supported",
				tag.Number, i, corim.ComidTag,
			)
		}

		var origComid comid.Comid
		if err = origComid.FromCBOR(tag.Content); err != nil {
			return fmt.Errorf("could not decode CoMID at index %d: %w", i, err)
		}

		modTag, err := NewModuleTagFromCoRIM(&origComid)
		if err != nil {
			return fmt.Errorf("could not create module tag at index %d: %w", i, err)
		}

		o.ModuleTags = append(o.ModuleTags, modTag)
	}

	if origin.RimValidity != nil {
		o.NotBefore = origin.RimValidity.NotBefore
		o.NotAfter = &origin.RimValidity.NotAfter
	}

	o.Extensions, err = CoRIMExtensionsFromCoRIM(origin.Extensions)
	if err != nil {
		return err
	}

	return nil
}

func (o *Manifest) ToCoRIM() (*corim.UnsignedCorim, error) {
	var ret corim.UnsignedCorim

	var manifestID *swid.TagID
	var err error

	switch o.ManifestIDType {
	case StringTagID:
		manifestID = swid.NewTagID(o.ManifestID)
	case UUIDTagID:
		tagUUID, err := uuid.Parse(o.ManifestID)
		if err != nil {
			return nil, err
		}

		manifestID = swid.NewTagID(tagUUID)
	default:
		return nil, fmt.Errorf("unexpected manifest ID type: %s", o.ManifestIDType)
	}

	if manifestID == nil {
		return nil, fmt.Errorf("could not create swid.TagID from %q", o.ManifestID)
	}

	ret.ID = *manifestID

	if o.Profile != "" {
		ret.Profile, err = eat.NewProfile(o.Profile)
		if err != nil {
			return nil, fmt.Errorf("profile: %w", err)
		}
	}

	ret.DependentRims, err = LocatorsToCoRIM(o.DependentRIMs)
	if err != nil {
		return nil, err
	}

	ret.Entities, err = CoRIMEntitiesToCoRIM(o.Entities)
	if err != nil {
		return nil, err
	}

	for i, moduleTag := range o.ModuleTags {
		newComid, err := moduleTag.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("module tag at index %d: %w", i, err)
		}

		comidBytes, err := newComid.ToCBOR()
		if err != nil {
			return nil, fmt.Errorf("could not encode CoMID at index %d: %w", i, err)
		}

		ret.Tags = append(ret.Tags, corim.Tag{Number: corim.ComidTag, Content: comidBytes})
	}

	if o.NotAfter != nil {
		ret.RimValidity = corim.NewValidity().Set(*o.NotAfter, o.NotBefore)
	} else if o.NotBefore != nil {
		return nil, errors.New("not-before is set but not-after isn't")
	}

	ret.Extensions, err = CoRIMExtensionsToCoRIM(o.Extensions)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Manifest) SetActive(value bool) {
	for _, mt := range o.ModuleTags {
		mt.SetActive(value)
	}
}

func (o *Manifest) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for _, entity := range o.Entities {
		entity.OwnerID = o.ID
		entity.OwnerType = "manifest"

		if err := entity.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting linked tag %+v: %w", entity, err)
		}
	}

	for _, locator := range o.DependentRIMs {
		locator.ManifestID = o.ID

		if err := locator.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting locator %+v: %w", locator, err)
		}
	}

	for _, moduleTag := range o.ModuleTags {
		moduleTag.ManifestID = o.ID

		if err := moduleTag.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting module tag %+v: %w", moduleTag, err)
		}
	}

	for _, ext := range o.Extensions {
		ext.OwnerID = o.ID
		ext.OwnerType = "manifest"

		if err := ext.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting extension %+v: %w", ext, err)
		}
	}

	return nil
}

func (o *Manifest) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Entities").
		Relation("ModuleTags").
		Relation("DependentRIMs").
		Relation("Extensions").
		Where("man.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, entity := range o.Entities {
		if err := entity.Select(ctx, db); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	for i, moduleTag := range o.ModuleTags {
		if err := moduleTag.Select(ctx, db); err != nil {
			return fmt.Errorf("module tag at index %d: %w", i, err)
		}
	}

	for i, locator := range o.DependentRIMs {
		if err := locator.Select(ctx, db); err != nil {
			return fmt.Errorf("depndency RIM at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *Manifest) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, entity := range o.Entities {
		if err := entity.Delete(ctx, db); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	for i, moduleTag := range o.ModuleTags {
		if err := moduleTag.Delete(ctx, db); err != nil {
			return fmt.Errorf("module tag at index %d: %w", i, err)
		}
	}

	for i, locator := range o.DependentRIMs {
		if err := locator.Delete(ctx, db); err != nil {
			return fmt.Errorf("dependency RIM at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *Manifest) Validate() error {
	if len(o.ManifestIDType) == 0 || len(o.ManifestID) == 0 {
		return errors.New("manifest ID not set (both type and value must be set)")
	}

	supprtedIDTypes := []TagIDType{StringTagID, UUIDTagID}
	if !slices.Contains(supprtedIDTypes, o.ManifestIDType) {
		return fmt.Errorf("unsupported manifest ID type: %s", o.ManifestIDType)
	}

	if len(o.ModuleTags) == 0 {
		return errors.New("no module tags")
	}

	for i, moduleTag := range o.ModuleTags {
		if err := moduleTag.Validate(); err != nil {
			return fmt.Errorf("module tag at index %d: %w", i, err)
		}
	}

	if o.NotAfter == nil && o.NotBefore != nil {
		return errors.New("not-before is set but not-after isn't")
	}

	for i, entity := range o.Entities {
		if err := entity.Validate(); err != nil {
			return fmt.Errorf("entity at index %d: %w", i, err)
		}
	}

	return nil
}
