package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func IntegerityRegistersFromCoRIM(origin *comid.IntegrityRegisters) ([]*IntegrityRegister, error) {
	if origin == nil {
		return nil, nil
	}

	var err error
	ret := make([]*IntegrityRegister, 0, len(origin.IndexMap))

	for i, corimDigests := range origin.IndexMap {
		var reg IntegrityRegister

		reg.Digests, err = DigestsFromCoRIM(&corimDigests)
		if err != nil {
			return nil, fmt.Errorf("could not convert digess for index %v: %w", i, err)
		}

		switch t := i.(type) {
		case string:
			reg.IndexText = &t
		case uint64:
			reg.IndexUint = &t
		case uint:
			val := uint64(t)
			reg.IndexUint = &val
		default:
			return nil, fmt.Errorf("unexpected index type %T for index %v", t, i)
		}

		ret = append(ret, &reg)
	}

	return ret, nil
}

func IntegerityRegistersToCoRIM(origin []*IntegrityRegister) (*comid.IntegrityRegisters, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewIntegrityRegisters()

	for _, reg := range origin {
		var idx comid.IRegisterIndex

		if reg.IndexText != nil && reg.IndexUint != nil { // nolint:gocritic
			return nil, fmt.Errorf(
				"both uint and string indices are set: %d, %s (ID %d)",
				*reg.IndexUint,
				*reg.IndexText,
				reg.ID,
			)
		} else if reg.IndexText != nil {
			idx = comid.IRegisterIndex(*reg.IndexText)
		} else if reg.IndexUint != nil {
			idx = comid.IRegisterIndex(*reg.IndexUint)
		} else {
			return nil, fmt.Errorf("neither index set at ID %d", reg.ID)
		}

		digests, err := DigestsToCoRIM(reg.Digests)
		if err != nil {
			return nil, fmt.Errorf("could not convert digests for index %v: %w", idx, err)
		}

		if digests == nil {
			return nil, fmt.Errorf("no digests for index %v", idx)
		}

		if err := ret.AddDigests(idx, *digests); err != nil {
			return nil, fmt.Errorf("could not add digests for index %v: %w", idx, err)
		}
	}

	return ret, nil
}

type IntegrityRegister struct {
	bun.BaseModel `bun:"table:integrity_registers,alias:int"`

	ID int64 `bun:",pk,autoincrement"`

	IndexUint *uint64
	IndexText *string

	Digests []*Digest `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic"`

	MeasurementID int64 `bun:",nullzero"`
}

// StringIndex return the index of this IntegrityRegister as a string. If
// IndexText is set, it returns the value it points at, If IndexUint is set,
// the value it points to formatted as a string is returned. Otherwise, the
// string "nil" is returned.
func (o *IntegrityRegister) StringIndex() string {
	if o.IndexText != nil { // nolint: gocritic
		return *o.IndexText
	} else if o.IndexUint != nil {
		return fmt.Sprint(*o.IndexUint)
	} else {
		return "nil"
	}
}

func (o *IntegrityRegister) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, digest := range o.Digests {
		digest.OwnerID = o.ID
		digest.OwnerType = "integrity_register"

		if err := digest.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting digest %d: %w", i, err)
		}
	}

	return nil
}

func (o *IntegrityRegister) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().
		Model(o).
		Relation("Digests").
		Where("id = ?", o.ID).
		Scan(ctx)
}

func (o *IntegrityRegister) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, digest := range o.Digests {
		if err := digest.Delete(ctx, db); err != nil {
			return fmt.Errorf("digest at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
