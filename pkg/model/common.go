package model

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/veraison/swid"
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

// OwnedModel is a Model that is owned by another Model.
type OwnedModel interface {
	Model

	// OwnerDbID returns the unique database ID of the owner of this
	// model.
	OwnerDbID() int64

	// OwnerName returns the name of the owner model. In case of Bun
	// polymorphic ownership, this must match the values specified in the
	// "polymorphic" part of the  field's tag (i.e. OwnerType). For
	// non-polymorphic ownership, a constant value is returned. In all
	// cases, the value typically correspeonds to the singular version of
	// the owner's table name. For example, values owned by a ModuleTag
	// (table name "module_tags") will return "module_tag".
	OwnerName() string
}

// ParseSWIDTagID convets a swid.TagID into a string representation of its
// value and a TagIDType indicting the value's type.
func ParseSWIDTagID(tag swid.TagID) (TagIDType, string) {
	tagID := tag.String()

	// swid.TagID does not expose the underlying type in any way, but we
	// need it to correctly reconstruct it inside ToCoRIM(), so we guess
	// by seeing if the ID parses as a valid UUID.
	var typ TagIDType
	if _, err := uuid.Parse(tagID); err == nil {
		typ = UUIDTagID
	} else {
		typ = StringTagID
	}

	return typ, tagID
}

var tableModels = []any{
	(*ConditionalEndorsementTriple)(nil),
	(*ConditionalEndorsementSeriesRecord)(nil),
	(*ConditionalEndorsementSeriesTriple)(nil),
	(*CryptoKey)(nil),
	(*Digest)(nil),
	(*DomainEntry)(nil),
	(*DomainDependencyTriple)(nil),
	(*DomainMembershipTriple)(nil),
	(*Entity)(nil),
	(*Environment)(nil),
	(*ExtensionValue)(nil),
	(*Flag)(nil),
	(*Href)(nil),
	(*IntegrityRegister)(nil),
	(*KeyTriple)(nil),
	(*LinkedTag)(nil),
	(*Locator)(nil),
	(*Manifest)(nil),
	(*Measurement)(nil),
	(*MeasurementValueEntry)(nil),
	(*ModuleTag)(nil),
	(*RoleEntry)(nil),
	(*StatefulEnvironment)(nil),
	(*ValueTriple)(nil),
	(*Token)(nil),
}

var viewModels = []any{
	(*ConditionalEndorsementTripleEntry)(nil),
	(*ConditionalEndorsementSeriesTripleEntry)(nil),
	(*DomainDependencyTripleEntry)(nil),
	(*DomainMembershipTripleEntry)(nil),
	(*KeyTripleEntry)(nil),
	(*ValueTripleEntry)(nil),
}

var allModels = append(tableModels, viewModels...)

func RegisterModels(db *bun.DB) {
	db.RegisterModel(allModels...)
}

func ResetModels(ctx context.Context, db *bun.DB) error {
	for _, table := range tableModels {
		if _, err := db.NewTruncateTable().Model(table).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
