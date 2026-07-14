package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func StatefulEnvironmentsFromCoRIM(origin *comid.StatefulEnvironments) ([]*StatefulEnvironment, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*StatefulEnvironment, 0, len(origin.Values))

	for i, originEnv := range origin.Values {
		env, err := NewStatefulEnvironmentFromCoRIM(&originEnv)
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("error converting stateful environment at index %d: %w", i, err)
		}

		ret = append(ret, env)
	}

	return ret, nil
}

func StatefulEnvironmentsToCoRIM(origin []*StatefulEnvironment) (*comid.StatefulEnvironments, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewStatefulEnvironments()

	for i, originEnv := range origin {
		env, err := originEnv.ToCoRIM()
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("could not convert stateful environment at index %d: %w", i, err)
		}

		ret.Add(env)
	}

	return ret, nil
}

type StatefulEnvironment struct {
	bun.BaseModel `bun:"table:stateful_environments,alias:senv"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	Environment   *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Measurements []*Measurement `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:stateful_environment"`

	TripleID int64 `bun:",nullzero"`
}

func NewStatefulEnvironmentFromCoRIM(origin *comid.StatefulEnvironment) (*StatefulEnvironment, error) {
	var ret StatefulEnvironment

	if err := ret.FromCoRIM(origin); err != nil {
		// coverage:ignore
		return nil, err
	}

	return &ret, nil
}

func SelectStatefulEnvironment(ctx context.Context, db bun.IDB, id int64) (*StatefulEnvironment, error) {
	ret := StatefulEnvironment{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		// coverage:ignore
		return nil, err
	}

	return &ret, nil
}

func (o *StatefulEnvironment) DbID() int64 {
	return o.ID
}

func (o *StatefulEnvironment) TableName() string {
	return "stateful_environments"
}

func (o *StatefulEnvironment) IsTable() bool {
	return true
}

func (o *StatefulEnvironment) OwnerDbID() int64 {
	return o.TripleID
}

func (o *StatefulEnvironment) OwnerName() string {
	return "conditional_endorsement_triple"
}

func (o *StatefulEnvironment) FromCoRIM(origin *comid.StatefulEnvironment) error {
	env, err := NewEnvironmentFromCoRIM(&origin.Environment)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("environment: %w", err)
	}

	meas, err := MeasurementsFromCoRIM(origin.Measurements)
	if err != nil {
		// coverage:ignore
		return err
	}

	o.Environment = env
	o.Measurements = meas

	return nil
}

func (o *StatefulEnvironment) ToCoRIM() (*comid.StatefulEnvironment, error) {
	var ret comid.StatefulEnvironment

	env, err := o.Environment.ToCoRIM()
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("environment: %w", err)
	}
	ret.Environment = *env

	meas, err := MeasurementsToCoRIM(o.Measurements)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("key list: %w", err)
	}
	ret.Measurements = meas

	return &ret, nil
}

func (o *StatefulEnvironment) Validate() error {
	if o.Environment == nil {
		return errors.New("environment not set")
	}

	if err := o.Environment.Validate(); err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	return nil
}

func (o *StatefulEnvironment) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if err := o.Environment.Insert(ctx, db); err != nil {
		// coverage:ignore
		return err
	}
	o.EnvironmentID = o.Environment.ID

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	for i, mea := range o.Measurements {
		mea.OwnerID = o.ID
		mea.OwnerType = "stateful_environment"

		if err := mea.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("measurement at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *StatefulEnvironment) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Environment").
		Relation("Measurements").
		Where("senv.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, mea := range o.Measurements {
		if err := mea.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("measurement at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *StatefulEnvironment) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, measurement := range o.Measurements {
		if err := measurement.Delete(ctx, db); err != nil {
			return fmt.Errorf("measurement at index %d: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	if o.Environment != nil {
		if err := o.Environment.DeleteIfOrphaned(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("environment: %w", err)
		}
	}

	return nil
}

func init() {
	environmentOwners = append(environmentOwners, "stateful_environments")
}
