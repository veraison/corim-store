package migrations

import (
	"context"
	"reflect"
	"time"

	"github.com/uptrace/bun"
)

// The models are duplicated here to provide immutable "snapshots" that will
// be used by the migration instead of the actual models. This means the
// migration will not be impacted by any future changes to the models. This
// will minimize the possibility of clashes with subsequent migrations.

type extensionValue_v1 struct {
	bun.BaseModel `bun:"table:extensions,alias:ext"`

	ID int64 `bun:",pk,autoincrement"`

	FieldKind reflect.Kind
	FieldName string
	JSONTag   string
	CBORTag   string

	ValueBytes []byte
	ValueText  string
	ValueInt   int64
	ValueFloat float64

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

type integrityRegister_v1 struct {
	bun.BaseModel `bun:"table:integrity_registers,alias:int"`

	ID int64 `bun:",pk,autoincrement"`

	IndexUint *uint64
	IndexText *string

	Digests []*digest_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic"`

	MeasurementID int64 `bun:",nullzero"`
}

type digest_v1 struct {
	bun.BaseModel `bun:"table:digests,alias:dgt"`

	ID int64 `bun:",pk,autoincrement"`

	AlgID uint64
	Value []byte

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

type flag_v1 struct {
	bun.BaseModel `bun:"table:flags,alias:flg"`

	ID int64 `bun:",pk,autoincrement"`

	CodePoint int64
	Value     bool

	MeasurementID int64 `bun:",nullzero"`
}

type measurementValueEntry_v1 struct {
	bun.BaseModel `bun:"table:measurement_value_entries,alias:mve"`

	ID int64 `bun:",pk,autoincrement"`

	CodePoint  int64
	ValueType  string
	ValueBytes *[]byte
	ValueText  *string
	ValueInt   *int64

	MeasurementID int64
}

type cryptoKey_v1 struct {
	bun.BaseModel `bun:"table:cryptokeys,alias:ck"`

	ID int64 `bun:",pk,autoincrement"`

	KeyType  string
	KeyBytes []byte

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

type measurement_v1 struct {
	bun.BaseModel `bun:"table:measurements,alias:mea"`

	ID int64 `bun:",pk,autoincrement"`

	KeyType  *string
	KeyBytes *[]byte

	ValueEntries       []*measurementValueEntry_v1 `bun:"rel:has-many,join:id=measurement_id"`
	Digests            []*digest_v1                `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`
	Flags              []*flag_v1                  `bun:"rel:has-many,join:id=measurement_id"`
	IntegrityRegisters []*integrityRegister_v1     `bun:"rel:has-many,join:id=measurement_id"`
	Extensions         []*extensionValue_v1        `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`

	AuthorizedBy []*cryptoKey_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`
	OwnerID      int64           `bun:",nullzero"`
	OwnerType    string          `bun:",nullzero"`
}

type environment_v1 struct {
	bun.BaseModel `bun:"table:environments,alias:env"`

	ID int64 `bun:",pk,autoincrement"`

	ClassType  *string
	ClassBytes *[]byte
	Vendor     *string
	Model      *string
	Layer      *uint64
	Index      *uint64

	InstanceType  *string
	InstanceBytes *[]byte

	GroupType  *string
	GroupBytes *[]byte
}

type keyTriple_v1 struct {
	bun.BaseModel `bun:"table:key_triples,alias:kt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64
	Environment   *environment_v1 `bun:"rel:belongs-to,join:environment_id=id"`

	Type    string
	KeyList []*cryptoKey_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:key_triple"`

	AuthorizedBy []*cryptoKey_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:key_triple_auth"`
	ModuleID     int64           `bun:",nullzero"`
}

type valueTriple_v1 struct {
	bun.BaseModel `bun:"table:value_triples,alias:vt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64           `bun:",nullzero"`
	Environment   *environment_v1 `bun:"rel:belongs-to,join:environment_id=id"`

	Type         string
	Measurements []*measurement_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:value_triple"`
	ModuleID     int64             `bun:",nullzero"`
}

type linkedTag_v1 struct {
	bun.BaseModel `bun:"table:linked_tags,alias:lnk"`

	ID int64 `bun:",pk,autoincrement"`

	LinkedTagIDType string
	LinkedTagID     string
	TagRelation     string

	ModuleID int64 `bun:",nullzero"`
}

type roleEntry_v1 struct {
	bun.BaseModel `bun:"table:roles,alias:rol"`

	ID int64 `bun:",pk,autoincrement"`

	Role string

	EntityID int64
}

type entity_v1 struct {
	bun.BaseModel `bun:"table:entities,alias:ent"`

	ID int64 `bun:",pk,autoincrement"`

	NameType string
	Name     string
	URI      string `bun:",nullzero"`

	RoleEntries []roleEntry_v1 `bun:"rel:has-many,join:id=entity_id"`

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`

	Extensions []*extensionValue_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic"`
}

type moduleTag_v1 struct {
	bun.BaseModel `bun:"table:module_tags,alias:mod"`

	ID int64 `bun:",pk,autoincrement"`

	TagIDType  string
	TagID      string
	TagVersion uint

	Language *string

	Entities []*entity_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`

	ValueTriples []*valueTriple_v1 `bun:"rel:has-many,join:id=module_id"`
	KeyTriples   []*keyTriple_v1   `bun:"rel:has-many,join:id=module_id"`

	LinkedTags []*linkedTag_v1 `bun:"rel:has-many,join:id=module_id"`

	Extensions        []*extensionValue_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`
	TriplesExtensions []*extensionValue_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:triples"`

	ManifestID int64
}

type locator_v1 struct {
	bun.BaseModel `bun:"table:locators,alias:loc"`

	ID int64 `bun:",pk,autoincrement"`

	Href       string
	Thumbprint []*digest_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:locator"`

	ManifestID int64
}

type manifest_v1 struct {
	bun.BaseModel `bun:"table:manifests,alias:man"`

	ID int64 `bun:",pk,autoincrement"`

	ManifestIDType string
	ManifestID     string

	Digest    []byte
	TimeAdded time.Time
	Label     string `bun:",nullzero"`

	ProfileType string `bun:",nullzero"`
	Profile     string `bun:",nullzero"`

	Entities      []*entity_v1  `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:manifest"`
	DependentRIMs []*locator_v1 `bun:"rel:has-many,join:id=manifest_id"`

	NotBefore *time.Time
	NotAfter  *time.Time

	ModuleTags []*moduleTag_v1 `bun:"rel:has-many,join:id=manifest_id"`

	Extensions []*extensionValue_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:module_tag"`
}

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error { // nolint:dupl
		var err error

		_, err = db.NewCreateTable().Model((*manifest_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*locator_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*moduleTag_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*entity_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*roleEntry_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*linkedTag_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*valueTriple_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*keyTriple_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*cryptoKey_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*environment_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*measurement_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*measurementValueEntry_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*digest_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*flag_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*integrityRegister_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewCreateTable().Model((*extensionValue_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error { // nolint:dupl
		var err error

		_, err = db.NewDropTable().Model((*manifest_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*locator_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*moduleTag_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*entity_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*roleEntry_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*linkedTag_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*valueTriple_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*keyTriple_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*cryptoKey_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*environment_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*measurement_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*measurementValueEntry_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*digest_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*flag_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*integrityRegister_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = db.NewDropTable().Model((*extensionValue_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
