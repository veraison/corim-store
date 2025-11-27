package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func CoRIMEntitiesFromCoRIM(origin *corim.Entities) ([]*Entity, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*Entity, 0, len(origin.Values))

	for i, origEntity := range origin.Values {
		entity, err := NewCoRIMEntityFromCoRIM(&origEntity)
		if err != nil {
			return nil, fmt.Errorf("entity at index %d: %w", i, err)
		}

		ret = append(ret, entity)
	}

	return ret, nil
}

func CoRIMEntitiesToCoRIM(origin []*Entity) (*corim.Entities, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := corim.NewEntities()
	for i, origEntity := range origin {
		entity, err := origEntity.ToCoRIMCoRIM()
		if err != nil {
			return nil, fmt.Errorf("entity at index %d: %w", i, err)
		}

		ret.Add(entity)
	}

	return ret, nil
}

func CoMIDEntitiesFromCoRIM(origin *comid.Entities) ([]*Entity, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*Entity, 0, len(origin.Values))

	for i, origEntity := range origin.Values {
		entity, err := NewCoMIDEntityFromCoRIM(&origEntity)
		if err != nil {
			return nil, fmt.Errorf("entity at index %d: %w", i, err)
		}

		ret = append(ret, entity)
	}

	return ret, nil
}

func CoMIDEntitiesToCoRIM(origin []*Entity) (*comid.Entities, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewEntities()
	for _, origEntity := range origin {
		entity, err := origEntity.ToCoMIDCoRIM()
		if err != nil {
			return nil, fmt.Errorf("problem with entity %+v: %w", origEntity, err)
		}

		ret.Add(entity)
	}

	return ret, nil
}

type Entity struct {
	bun.BaseModel `bun:"table:entities,alias:ent"`

	ID int64 `bun:",pk,autoincrement"`

	NameType string
	Name     string
	URI      string `bun:",nullzero"`

	RoleEntries []RoleEntry `bun:"rel:has-many,join:id=entity_id"`

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`

	Extensions []*ExtensionValue `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic"`
}

func NewCoRIMEntityFromCoRIM(origin *corim.Entity) (*Entity, error) {
	var ret Entity

	if err := ret.FromCoRIMCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func NewCoMIDEntityFromCoRIM(origin *comid.Entity) (*Entity, error) {
	var ret Entity

	if err := ret.FromCoMIDCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Entity) FromCoRIMCoRIM(origin *corim.Entity) error {
	var err error

	o.NameType = origin.Name.Value.Type()
	o.Name = origin.Name.String()

	if origin.RegID != nil {
		o.URI = string(*origin.RegID)
	}

	o.RoleEntries, err = CoRIMRolesFromCoRIM(origin.Roles)
	if err != nil {
		return err
	}

	o.Extensions, err = CoRIMExtensionsFromCoRIM(origin.Extensions)
	if err != nil {
		return err
	}

	return nil
}

func (o *Entity) ToCoRIMCoRIM() (*corim.Entity, error) {
	var err error
	var ret corim.Entity

	ret.Name, err = corim.NewEntityName(o.Name, o.NameType)
	if err != nil {
		return nil, err
	}

	if o.URI != "" {
		regID := comid.TaggedURI(o.URI)
		ret.RegID = &regID
	}

	for _, roleEntry := range o.RoleEntries {
		role, err := ParseCoRIMRole(roleEntry.Role)
		if err != nil {
			return nil, err
		}

		ret.Roles = append(ret.Roles, role)
	}

	ret.Extensions, err = CoRIMExtensionsToCoRIM(o.Extensions)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Entity) FromCoMIDCoRIM(origin *comid.Entity) error {
	var err error

	o.NameType = origin.Name.Value.Type()
	o.Name = origin.Name.String()

	if origin.RegID != nil {
		o.URI = string(*origin.RegID)
	}

	o.RoleEntries, err = CoMIDRolesFromCoRIM(origin.Roles)
	if err != nil {
		return err
	}

	o.Extensions, err = CoMIDExtensionsFromCoRIM(origin.Extensions)
	if err != nil {
		return err
	}

	return nil
}

func (o *Entity) ToCoMIDCoRIM() (*comid.Entity, error) {
	var err error
	var ret comid.Entity

	ret.Name, err = comid.NewEntityName(o.Name, o.NameType)
	if err != nil {
		return nil, err
	}

	if o.URI != "" {
		regID := comid.TaggedURI(o.URI)
		ret.RegID = &regID
	}

	for _, roleEntry := range o.RoleEntries {
		role, err := ParseCoMIDRole(roleEntry.Role)
		if err != nil {
			return nil, err
		}

		ret.Roles = append(ret.Roles, role)
	}

	ret.Extensions, err = CoMIDExtensionsToCoRIM(o.Extensions)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Entity) Roles() []string {
	ret := make([]string, 0, len(o.RoleEntries))

	for _, entry := range o.RoleEntries {
		ret = append(ret, entry.Role)
	}

	return ret
}

func (o *Entity) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, roleEntry := range o.RoleEntries {
		roleEntry.EntityID = o.ID
		if err := roleEntry.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting role at index %d: %w", i, err)
		}
	}

	for _, ext := range o.Extensions {
		ext.OwnerID = o.ID
		ext.OwnerType = "entity"

		if err := ext.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting extension %+v: %w", ext, err)
		}
	}

	return nil
}

func (o *Entity) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("RoleEntries").
		Relation("Extensions").
		Where("ent.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *Entity) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, extension := range o.Extensions {
		if err := extension.Delete(ctx, db); err != nil {
			return fmt.Errorf("extension at index %d: %w", i, err)
		}
	}

	for i, entry := range o.RoleEntries {
		if err := entry.Delete(ctx, db); err != nil {
			return fmt.Errorf("role at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *Entity) Validate() error {
	if len(o.RoleEntries) == 0 {
		return fmt.Errorf("no roles")
	}

	return nil
}
