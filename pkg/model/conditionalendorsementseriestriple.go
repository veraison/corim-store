package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func ConditionalEndorsementSeriesRecordsFromCoRIM(
	origin comid.CondEndorseSeriesRecords,
) ([]*ConditionalEndorsementSeriesRecord, error) {
	if len(origin.Values) == 0 {
		return nil, errors.New("no records")
	}

	ret := make([]*ConditionalEndorsementSeriesRecord, 0, len(origin.Values))
	for i, origRecord := range origin.Values {
		record, err := NewConditionalEndorsementSeriesRecordFromCoRIM(&origRecord)
		if err != nil {
			// coverage:ignore
			return nil, fmt.Errorf("record[%d]: %w", i, err)
		}

		ret = append(ret, record)
	}

	return ret, nil
}

func ConditionalEndorsementSeriesRecordsToCoRIM(
	origin []*ConditionalEndorsementSeriesRecord,
) (comid.CondEndorseSeriesRecords, error) {
	ret := comid.NewCondEndorseSeriesRecords()

	for i, origRecord := range origin {
		record, err := origRecord.ToCoRIM()
		if err != nil {
			// coverage:ignore
			return comid.CondEndorseSeriesRecords{}, fmt.Errorf("record[%d]: %w", i, err)
		}

		ret.Add(record)
	}

	return *ret, nil
}

type ConditionalEndorsementSeriesRecord struct {
	bun.BaseModel `bun:"table:conditional_endorsement_series_records,alias:cesr"`

	ID int64 `bun:",pk,autoincrement"`

	Selection []*Measurement `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:ces_record_selection"`
	Addition  []*Measurement `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:ces_record_addition"`
	TripleID  int64
}

func NewConditionalEndorsementSeriesRecordFromCoRIM(
	origin *comid.CondEndorseSeriesRecord,
) (*ConditionalEndorsementSeriesRecord, error) {
	var ret ConditionalEndorsementSeriesRecord

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *ConditionalEndorsementSeriesRecord) DbID() int64 {
	return o.ID
}

func (o *ConditionalEndorsementSeriesRecord) OwnerDbID() int64 {
	return o.TripleID
}

func (o *ConditionalEndorsementSeriesRecord) OwnerName() string {
	return "conditional_endorsement_series_triple"
}

func (o *ConditionalEndorsementSeriesRecord) TableName() string {
	return "conditional_endorsement_series_records"
}

func (o *ConditionalEndorsementSeriesRecord) IsTable() bool {
	return true
}

func (o *ConditionalEndorsementSeriesRecord) FromCoRIM(origin *comid.CondEndorseSeriesRecord) error {
	selection, err := MeasurementsFromCoRIM(origin.Selection)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("selection: %w", err)
	}

	addition, err := MeasurementsFromCoRIM(origin.Addition)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("addition: %w", err)
	}

	o.Selection = selection
	o.Addition = addition

	return nil
}

func (o *ConditionalEndorsementSeriesRecord) ToCoRIM() (*comid.CondEndorseSeriesRecord, error) {
	selection, err := MeasurementsToCoRIM(o.Selection)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("selection: %w", err)
	}

	addition, err := MeasurementsToCoRIM(o.Addition)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("addition: %w", err)
	}

	return &comid.CondEndorseSeriesRecord{
		Selection: selection,
		Addition:  addition,
	}, nil
}

func (o *ConditionalEndorsementSeriesRecord) Validate() error {
	if len(o.Selection) == 0 {
		return errors.New("empty selection")
	}

	if len(o.Addition) == 0 {
		return errors.New("empty addition")
	}

	return nil
}

func (o *ConditionalEndorsementSeriesRecord) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	for i, measurement := range o.Selection {
		measurement.OwnerID = o.ID
		measurement.OwnerType = "ces_record_selection"

		if err := measurement.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("selection[%d]: %w", i, err)
		}
	}

	for i, measurement := range o.Addition {
		measurement.OwnerID = o.ID
		measurement.OwnerType = "ces_record_addition"

		if err := measurement.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("addition[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementSeriesRecord) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Selection").
		Relation("Addition").
		Where("cesr.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, measurement := range o.Selection {
		if err := measurement.Select(ctx, db); err != nil {
			return fmt.Errorf("selection[%d]: %w", i, err)
		}
	}

	for i, measurement := range o.Addition {
		if err := measurement.Select(ctx, db); err != nil {
			return fmt.Errorf("addition[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementSeriesRecord) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, measurement := range o.Selection {
		if err := measurement.Delete(ctx, db); err != nil {
			return fmt.Errorf("selection[%d]: %w", i, err)
		}
	}

	for i, measurement := range o.Addition {
		if err := measurement.Delete(ctx, db); err != nil {
			return fmt.Errorf("addition[%d]: %w", i, err)
		}
	}

	return nil
}

func ConditionalEndorsementSeriesTriplesFromCoRIM(
	origin *comid.CondEndorseSeriesTriples,
) ([]*ConditionalEndorsementSeriesTriple, error) {
	if origin == nil || len(origin.Values) == 0 {
		return nil, nil
	}

	ret := make([]*ConditionalEndorsementSeriesTriple, 0, len(origin.Values))

	for i, originTriple := range origin.Values {
		triple, err := NewConditionalEndorsementSeriesTripleFromCoRIM(&originTriple)
		if err != nil {
			return nil, fmt.Errorf("triple[%d]: %w", i, err)
		}

		ret = append(ret, triple)
	}

	return ret, nil
}

func ConditionalEndorsementSeriesTriplesToCoRIM(
	origin []*ConditionalEndorsementSeriesTriple,
) (*comid.CondEndorseSeriesTriples, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	ret := comid.NewCondEndorseSeriesTriples()

	for i, originTriple := range origin {
		triple, err := originTriple.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("triple[%d]: %w", i, err)
		}

		ret.Add(triple)
	}

	return ret, nil
}

type ConditionalEndorsementSeriesTriple struct {
	bun.BaseModel `bun:"table:conditional_endorsement_series_triples,alias:cest"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64        `bun:",nullzero"`
	Environment   *Environment `bun:"rel:belongs-to,join:environment_id=id"`

	Measurements []*Measurement `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:ces_condition"`

	AuthorizedBy []*CryptoKey `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:ces_condition"`

	Series []*ConditionalEndorsementSeriesRecord `bun:"rel:has-many,join:id=triple_id"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

func NewConditionalEndorsementSeriesTripleFromCoRIM(
	origin *comid.CondEndorseSeriesTriple,
) (*ConditionalEndorsementSeriesTriple, error) {
	var ret ConditionalEndorsementSeriesTriple

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectConditionalEndorsementSeriesTriple(
	ctx context.Context,
	db bun.IDB,
	id int64,
) (*ConditionalEndorsementSeriesTriple, error) {
	ret := ConditionalEndorsementSeriesTriple{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *ConditionalEndorsementSeriesTriple) DbID() int64 {
	return o.ID
}

func (o *ConditionalEndorsementSeriesTriple) OwnerDbID() int64 {
	return o.ModuleID
}

func (o *ConditionalEndorsementSeriesTriple) OwnerName() string {
	return "module_tag"
}

func (o *ConditionalEndorsementSeriesTriple) TableName() string {
	return "conditional_endorsement_series_triples"
}

func (o *ConditionalEndorsementSeriesTriple) IsTable() bool {
	return true
}

func (o *ConditionalEndorsementSeriesTriple) FromCoRIM(origin *comid.CondEndorseSeriesTriple) error {
	env, err := NewEnvironmentFromCoRIM(&origin.Condition.Environment)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("condition environment: %w", err)
	}

	meas, err := MeasurementsFromCoRIM(origin.Condition.Measurements)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("condition measurements: %w", err)
	}

	auth, err := CryptoKeysFromCoRIM(origin.Condition.AuthorizedBy)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("condition authorized-by: %w", err)
	}

	series, err := ConditionalEndorsementSeriesRecordsFromCoRIM(origin.Series)
	if err != nil {
		// coverage:ignore
		return fmt.Errorf("series: %w", err)
	}

	o.Environment = env
	o.Measurements = meas
	o.AuthorizedBy = auth
	o.Series = series

	return nil
}

func (o *ConditionalEndorsementSeriesTriple) ToCoRIM() (*comid.CondEndorseSeriesTriple, error) {
	var ret comid.CondEndorseSeriesTriple

	env, err := o.Environment.ToCoRIM()
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("condition environment: %w", err)
	}
	ret.Condition.Environment = *env

	meas, err := MeasurementsToCoRIM(o.Measurements)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("condition measurements: %w", err)
	}
	ret.Condition.Measurements = meas

	auth, err := CryptoKeysToCoRIM(o.AuthorizedBy)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("condition authorized-by: %w", err)
	}
	ret.Condition.AuthorizedBy = auth

	series, err := ConditionalEndorsementSeriesRecordsToCoRIM(o.Series)
	if err != nil {
		// coverage:ignore
		return nil, fmt.Errorf("series: %w", err)
	}
	ret.Series = series

	return &ret, nil
}

func (o *ConditionalEndorsementSeriesTriple) Validate() error {
	if o.Environment == nil {
		return errors.New("condition environment not set")
	}

	if err := o.Environment.Validate(); err != nil {
		return fmt.Errorf("condition environment: %w", err)
	}

	if len(o.Series) == 0 {
		return fmt.Errorf("empty series")
	}

	for i, record := range o.Series {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("series[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementSeriesTriple) Insert(ctx context.Context, db bun.IDB) error {
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
		mea.OwnerType = "ces_condition"

		if err := mea.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition measurement[%d]: %w", i, err)
		}
	}

	for i, auth := range o.AuthorizedBy {
		auth.OwnerID = o.ID
		auth.OwnerType = "ces_condition"

		if err := auth.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition authorized-by[%d]: %w", i, err)
		}
	}

	for i, record := range o.Series {
		record.TripleID = o.ID

		if err := record.Insert(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("series[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementSeriesTriple) Select(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Environment").
		Relation("Measurements").
		Relation("AuthorizedBy").
		Relation("Series").
		Where("cest.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	for i, measurement := range o.Measurements {
		if err := measurement.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition measurement[%d]: %w", i, err)
		}
	}

	for i, auth := range o.AuthorizedBy {
		if err := auth.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition authorized-by[%d]: %w", i, err)
		}
	}

	for i, record := range o.Series {
		if err := record.Select(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("series[%d]: %w", i, err)
		}
	}

	return nil
}

func (o *ConditionalEndorsementSeriesTriple) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, measurement := range o.Measurements {
		if err := measurement.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition measurement[%d]: %w", i, err)
		}
	}

	for i, auth := range o.AuthorizedBy {
		if err := auth.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("condition authorized-by[%d]: %w", i, err)
		}
	}

	for i, record := range o.Series {
		if err := record.Delete(ctx, db); err != nil {
			// coverage:ignore
			return fmt.Errorf("series[%d]: %w", i, err)
		}
	}

	if _, err := db.NewDelete().Model(o).WherePK().Exec(ctx); err != nil {
		// coverage:ignore
		return err
	}

	if err := o.Environment.DeleteIfOrphaned(ctx, db); err != nil {
		// coverage:ignore
		return fmt.Errorf("environment: %w", err)
	}

	return nil
}

func init() {
	environmentOwners = append(environmentOwners, "conditional_endorsement_series_triples")
}
