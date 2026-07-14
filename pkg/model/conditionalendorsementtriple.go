package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func ConditionalEndorsementTriplesFromCoRIM(
	origin *comid.CondEndorseTriples,
) ([]*ConditionalEndorsementTriple, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*ConditionalEndorsementTriple, 0, len(origin.Values))

	for i, originTriple := range origin.Values {
		triple, err := NewConditionalEndorsementTripleFromCoRIM(&originTriple)
		if err != nil {
			return nil, fmt.Errorf("error converting conditional endorsement at index %d: %w", i, err)
		}

		ret = append(ret, triple)
	}

	return ret, nil
}

func ConditionalEndorsementTriplesToCoRIM(
	origin []*ConditionalEndorsementTriple,
) (*comid.CondEndorseTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewCondEndorseTriples()

	for i, originTriple := range origin {
		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("could not convert value triple at index %d: %w", i, err)
		}

		ret.Add(triple)
	}

	return ret, nil
}

type ConditionalEndorsementTriple struct {
	bun.BaseModel `bun:"table:conditional_endorsement_triples,alias:cet"`

	ID int64 `bun:",pk,autoincrement"`

	Conditions   []*StatefulEnvironment `bun:"rel:has-many,join:id=triple_id"`
	Endorsements []*ValueTriple         `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:conditional_endorsement_triple"`
	IsActive     bool

	ModuleID int64 `bun:",nullzero"`
}

func NewConditionalEndorsementTripleFromCoRIM(
	origin *comid.CondEndorseTriple,
) (*ConditionalEndorsementTriple, error) {
	var ret ConditionalEndorsementTriple

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectConditionalEndorsementTriple(
	ctx context.Context,
	db bun.IDB,
	id int64,
) (*ConditionalEndorsementTriple, error) {
	ret := ConditionalEndorsementTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *ConditionalEndorsementTriple) DbID() int64 {
	return o.ID
}

func (o *ConditionalEndorsementTriple) OwnerDbID() int64 {
	return o.ModuleID
}

func (o *ConditionalEndorsementTriple) OwnerName() string {
	return "module_tag"
}

func (o *ConditionalEndorsementTriple) TableName() string {
	return "conditional_endorsement_triples"
}

func (o *ConditionalEndorsementTriple) IsTable() bool {
	return true
}

func (o *ConditionalEndorsementTriple) FromCoRIM(origin *comid.CondEndorseTriple) error {
	conditions, err := StatefulEnvironmentsFromCoRIM(&origin.Conditions)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("conditions: %w", err)
	}

	endorsements, err := ValueTriplesFromCoRIM(&origin.Endorsements, EndorsedValueTriple)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("endorsements: %w", err)
	}

	o.Conditions = conditions
	o.Endorsements = endorsements

	return nil
}

func (o *ConditionalEndorsementTriple) ToCoRIM() (*comid.CondEndorseTriple, error) {
	var ret comid.CondEndorseTriple

	conditions, err := StatefulEnvironmentsToCoRIM(o.Conditions)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("conditions: %w", err)
	}
	ret.Conditions = *conditions

	endorsements, err := ValueTriplesToCoRIM(o.Endorsements, EndorsedValueTriple)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("endorsements: %w", err)
	}
	ret.Endorsements = *endorsements

	return &ret, nil
}

func (o *ConditionalEndorsementTriple) Validate() error {
	if len(o.Conditions) == 0 {
		return errors.New("conditions not set")
	}

	for i, cond := range o.Conditions {
		if err := cond.Validate(); err != nil {
			return fmt.Errorf("conditions[%d]: %w", i, err)
		}
	}

	if len(o.Endorsements) == 0 {
		return errors.New("endorsements not set")
	}

	for i, endorsement := range o.Endorsements {
		if err := endorsement.Validate(); err != nil {
			return fmt.Errorf("endorsements[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementTriple) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	for i, cond := range o.Conditions {
		cond.TripleID = o.ID

		if err := cond.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("conditions[%d]: %w", i, err)
		}
	}

	for i, endorsement := range o.Endorsements {
		endorsement.OwnerID = o.ID
		endorsement.OwnerType = "conditional_endorsement_triple"

		if err := endorsement.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("endorsements[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementTriple) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Conditions").
		Relation("Endorsements").
		Where("cet.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, cond := range o.Conditions {
		if err := cond.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("conditions[%d]: %w", i, err)
		}
	}

	for i, endorsement := range o.Endorsements {
		if err := endorsement.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("endorsements[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, cond := range o.Conditions {
		if err := cond.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("conditions[%d]: %w", i, err)
		}
	}

	for i, endorsement := range o.Endorsements {
		if err := endorsement.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("endorsements[%d]: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		return err
	}

	return nil
}
