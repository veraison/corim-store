package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

type ValueTripleType string

const (
	ReferenceValueTriple ValueTripleType = "reference"
	EndorsedValueTriple  ValueTripleType = "endorsement"
)

func ValueTriplesFromCoRIM(origin *comid.ValueTriples, typ ValueTripleType) ([]*ValueTriple, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*ValueTriple, 0, len(origin.Values))

	for i, originTriple := range origin.Values {
		triple, err := NewValueTripleFromCoRIM(&originTriple)
		if err != nil {
			return nil, fmt.Errorf("error converting %s value at index %d: %w", typ, i, err)
		}

		triple.Type = typ
		ret = append(ret, triple)
	}

	return ret, nil
}

func ValueTriplesToCoRIM(origin []*ValueTriple, typ ValueTripleType) (*comid.ValueTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewValueTriples()

	for i, originTriple := range origin {
		if typ != originTriple.Type {
			continue
		}

		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("could not conver value triple at index %d: %w", i, err)
		}

		ret.Add(triple)
	}

	return ret, nil
}

type ValueTriple struct {
	bun.BaseModel `bun:"table:value_triples,alias:vt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	Environment   *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Type         ValueTripleType
	Measurements []*Measurement `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:value_triple"`
	ModuleID     int64          `bun:",nullzero"`
}

func NewValueTripleFromCoRIM(origin *comid.ValueTriple) (*ValueTriple, error) {
	var ret ValueTriple

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectValueTriple(ctx context.Context, db bun.IDB, id int64) (*ValueTriple, error) {
	ret := ValueTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *ValueTriple) FromCoRIM(origin *comid.ValueTriple) error {
	env, err := NewEnvironmentFromCoRIM(&origin.Environment)
	if err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	meas, err := MeasurementsFromCoRIM(origin.Measurements)
	if err != nil {
		return err
	}

	o.Environment = env
	o.Measurements = meas

	return nil
}

func (o *ValueTriple) ToCoRIM() (*comid.ValueTriple, error) {
	var ret comid.ValueTriple

	env, err := o.Environment.ToCoRIM()
	if err != nil {
		return nil, fmt.Errorf("environment: %w", err)
	}
	ret.Environment = *env

	meas, err := MeasurementsToCoRIM(o.Measurements)
	if err != nil {
		return nil, fmt.Errorf("key list: %w", err)
	}
	ret.Measurements = meas

	return &ret, nil
}

func (o *ValueTriple) Validate() error {
	if o.Type == "" {
		return errors.New("value triple type not set")
	}

	if o.Environment == nil {
		return errors.New("environment not set")
	}

	if err := o.Environment.Validate(); err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	if len(o.Measurements) == 0 {
		return errors.New("no measurements")
	}

	return nil
}

func (o *ValueTriple) Insert(ctx context.Context, db bun.IDB) error {
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

	for i, mea := range o.Measurements {
		mea.OwnerID = o.ID
		mea.OwnerType = "value_triple"

		if err := mea.Insert(ctx, db); err != nil {
			return fmt.Errorf("key list index %d: %w", i, err)
		}
	}

	return nil
}

func (o *ValueTriple) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Environment").
		Relation("Measurements").
		Where("vt.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, mea := range o.Measurements {
		if err := mea.Select(ctx, db); err != nil {
			return fmt.Errorf("measurement at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *ValueTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, measurement := range o.Measurements {
		if err := measurement.Delete(ctx, db); err != nil {
			return fmt.Errorf("measurement at index %d: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		return err
	}

	return o.Environment.DeleteIfOrphaned(ctx, db)
}

func (o *ValueTriple) TripleType() string {
	return "value"
}

func (o *ValueTriple) DatabaseID() int64 {
	return o.ID
}
