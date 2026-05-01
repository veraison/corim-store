package model

import (
	"context"

	"github.com/uptrace/bun"
)

type TagIDType string

const (
	StringTagID TagIDType = "string"
	UUIDTagID   TagIDType = "uuid"
)

var SupportedTagIDTypes = []TagIDType{
	StringTagID,
	UUIDTagID,
}

// Model interface is implemented by all models.
type Model interface {
	// DbID is the database ID of this Model; it is guaranteed to be unique
	// among all models of its type. For table models, this correponds to
	// the value of the primary key "id" column of the table; for view
	// models, this corresponds to the "id" column of the primary element
	// comprising the view.
	DbID() int64
	// IsTable returns true if the Model corresponds to a database table;
	// otherwise, the model correponds to a view.
	IsTable() bool
	// TableName is the name of the table/view in the database this Model
	// is populated from.
	TableName() string
	// Select populates the Model from the database, including any nested
	// sub-models. If the field corresponding to the Model's DB ID is not
	// set, an error is returned.
	Select(context.Context, bun.IDB) error
}

var tableModels = []any{
	(*CryptoKey)(nil),
	(*Digest)(nil),
	(*Entity)(nil),
	(*Environment)(nil),
	(*ExtensionValue)(nil),
	(*Flag)(nil),
	(*IntegrityRegister)(nil),
	(*KeyTriple)(nil),
	(*LinkedTag)(nil),
	(*Locator)(nil),
	(*Manifest)(nil),
	(*Measurement)(nil),
	(*MeasurementValueEntry)(nil),
	(*ModuleTag)(nil),
	(*RoleEntry)(nil),
	(*ValueTriple)(nil),
	(*Token)(nil),
}

var viewModels = []any{
	(*KeyTripleEntry)(nil),
	(*ValueTripleEntry)(nil),
}

var allModels = append(tableModels, viewModels...)

func RegisterModels(db *bun.DB) {
	db.RegisterModel(allModels...)
}

func ResetModels(ctx context.Context, db *bun.DB) error {
	return db.ResetModel(ctx, tableModels...)
}
