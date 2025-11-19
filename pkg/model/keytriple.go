package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

type KeyTripleType string

const (
	AttestKeyTriple   KeyTripleType = "attest"
	IdentityKeyTriple KeyTripleType = "identity"
)

func KeyTriplesFromCoRIM(origin *comid.KeyTriples, typ KeyTripleType) ([]*KeyTriple, error) {
	if origin == nil || len(*origin) == 0 {
		return nil, nil
	}

	ret := make([]*KeyTriple, 0, len(*origin))

	for i, originTriple := range *origin {
		triple, err := NewKeyTripleFromCoRIM(&originTriple)
		if err != nil {
			return nil, fmt.Errorf("error converting %s key at index %d: %w", typ, i, err)
		}

		triple.Type = typ
		ret = append(ret, triple)
	}

	return ret, nil
}

func KeyTriplesToCoRIM(origin []*KeyTriple, typ KeyTripleType) (*comid.KeyTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewKeyTriples()

	for i, originTriple := range origin {
		if typ != originTriple.Type {
			continue
		}

		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("could not conver key triple at index %d: %w", i, err)
		}

		*ret = append(*ret, *triple)
	}

	return ret, nil
}

type KeyTriple struct {
	bun.BaseModel `bun:"table:key_triples,alias:kt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64
	Environment   *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Type    KeyTripleType
	KeyList []*CryptoKey `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:key_triple"`

	AuthorizedBy []*CryptoKey `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:key_triple_auth"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

func NewKeyTripleFromCoRIM(origin *comid.KeyTriple) (*KeyTriple, error) {
	var ret KeyTriple

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectKeyTriple(ctx context.Context, db bun.IDB, id int64) (*KeyTriple, error) {
	ret := KeyTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *KeyTriple) FromCoRIM(origin *comid.KeyTriple) error {
	var env Environment

	if err := env.FromCoRIM(&origin.Environment); err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	keys, err := CryptoKeysFromCoRIM(&origin.VerifKeys)
	if err != nil {
		return err
	}

	o.KeyList = keys
	o.Environment = &env

	return nil
}

func (o *KeyTriple) ToCoRIM() (*comid.KeyTriple, error) {
	var ret comid.KeyTriple

	env, err := o.Environment.ToCoRIM()
	if err != nil {
		return nil, fmt.Errorf("environment: %w", err)
	}
	ret.Environment = *env

	keys, err := CryptoKeysToCoRIM(o.KeyList)
	if err != nil {
		return nil, fmt.Errorf("key list: %w", err)
	}
	ret.VerifKeys = *keys

	return &ret, nil
}

func (o *KeyTriple) Validate() error {
	if o.Type == "" {
		return errors.New("key triple type not set")
	}

	if o.Environment == nil {
		return errors.New("environment not set")
	}

	if err := o.Environment.Validate(); err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	if len(o.KeyList) == 0 {
		return errors.New("empty key list")
	}

	return nil
}

func (o *KeyTriple) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if err := o.Environment.Insert(ctx, db); err != nil {
		return err
	}
	o.EnvironmentID = o.Environment.ID

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, key := range o.KeyList {
		key.OwnerID = o.ID
		key.OwnerType = "key_triple"

		if err := key.Insert(ctx, db); err != nil {
			return fmt.Errorf("key list index %d: %w", i, err)
		}
	}

	for i, key := range o.AuthorizedBy {
		key.OwnerID = o.ID
		key.OwnerType = "key_triple_auth"

		if err := key.Insert(ctx, db); err != nil {
			return fmt.Errorf("authorized-by index %d: %w", i, err)
		}
	}

	return nil
}

func (o *KeyTriple) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Environment").
		Relation("KeyList").
		Relation("AuthorizedBy").
		Where("kt.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *KeyTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, key := range o.KeyList {
		if err := key.Delete(ctx, db); err != nil {
			return fmt.Errorf("crypto key at index %d: %w", i, err)
		}
	}

	for i, key := range o.AuthorizedBy {
		if err := key.Delete(ctx, db); err != nil {
			return fmt.Errorf("authorized-by key at index %d: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		return err
	}

	return o.Environment.DeleteIfOrphaned(ctx, db)
}

func (o *KeyTriple) TripleType() string {
	return "key"
}

func (o *KeyTriple) DatabaseID() int64 {
	return o.ID
}
