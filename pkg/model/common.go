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

var allModels = []any{
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
}

func RegisterModels(db *bun.DB) {
	db.RegisterModel(allModels...)
}

func ResetModels(ctx context.Context, db *bun.DB) error {
	return db.ResetModel(ctx, allModels...)
}
