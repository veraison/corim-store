package store

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/migrations"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

var ErrNoLabel = errors.New("a label must be specified (required by store configuration)")
var ErrNoMatch = errors.New("no match found")

type Store struct {
	Ctx context.Context
	DB  bun.IDB

	cfg *Config
}

// Open a Store configured according to provided Config that will use the
// provided Context for its transactions.
func Open(ctx context.Context, cfg *Config) (*Store, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	db, err := db.Open(cfg.DB())
	if err != nil {
		return nil, err
	}

	return &Store{Ctx: ctx, cfg: cfg, DB: db}, nil
}

// OpenWithDB opens a store using an existing bun.DB and specified config options.
func OpenWithDB(ctx context.Context, db *bun.DB, options ...ConfigOption) (*Store, error) {
	cfg := NewConfig(db.Dialect().Name().String(), "<USING EXISTING DB>").WithOptions(options...)
	return &Store{Ctx: ctx, cfg: cfg, DB: db}, nil
}

// Close the Store. If the Store has a database connection (rather than a
// transaction), the database connection will be closed as well.
func (o *Store) Close() error {
	db, ok := o.DB.(*bun.DB)
	if ok {
		return db.Close()
	}

	return nil
}

// BeginTx starts an new transaction (bun.Tx) and returns a Store that uses
// that transaction as its "database". All method invocations on the returned
// Store will be part of of that transaction. The transaction can be committed
// or rolled back by accessing it via Tx() method of the returned Store.
func (o *Store) BeginTx(opts *sql.TxOptions) (*Store, error) {
	tx, err := o.DB.BeginTx(o.Ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Store{Ctx: o.Ctx, DB: tx, cfg: o.cfg}, nil
}

// Tx return *bun.Tx pointing to the transaction the Store is using as its DB.
// If Store.DB is not a transaction, nil si returned.
func (o *Store) Tx() *bun.Tx {
	tx, ok := o.DB.(bun.Tx)
	if ok {
		return &tx
	}

	return nil
}

// Init inializes a new database with the store's tables
func (o *Store) Init() error {
	db, ok := o.DB.(*bun.DB)
	if !ok {
		return errors.New("cannot Init via transaction")
	}

	migrator := migrate.NewMigrator(db, migrations.Migrations)

	if err := migrator.Init(o.Ctx); err != nil {
		return err
	}

	return o.Migrate()
}

// Migrate upates the tables in the associated database to be compatible with this store.
// (note: there is no need to run this after invoking Store.Init().)
func (o *Store) Migrate() error {
	db, ok := o.DB.(*bun.DB)
	if !ok {
		return errors.New("cannot Migrate via transaction")
	}

	migrator := migrate.NewMigrator(db, migrations.Migrations)

	if err := migrator.Lock(o.Ctx); err != nil {
		return err
	}
	defer migrator.Unlock(o.Ctx) // nolint:errcheck

	_, err := migrator.Migrate(o.Ctx)
	return err
}

// AddBytes adds the CBOR-encoded CoRIM in the provided buffer to the store.
// Signature validation of signed CoRIMs is not supported. If insecure
// transactions are allowed by the Store's configuration, signed corims will
// be added without validating their signatures. Otherwise, an error will be
// returned. If activate is true, the contained triples will be activated
// before they are added.
func (o *Store) AddBytes(buf []byte, label string, activate bool) error {
	if len(buf) < 3 {
		return fmt.Errorf("input too short")
	}

	digest := o.Digest(buf)

	if buf[0] == 0xd2 { // nolint:gocritic
		// tag 18 -> COSE_Sign1 -> signed corim
		if !o.cfg.Insecure {
			return errors.New(
				"signed CoRIM validation not supported (set insecure config to add unvalidated)",
			)
		}

		var signed corim.SignedCorim
		if err := signed.FromCOSE(buf); err != nil {
			return err
		}

		return o.AddCoRIM(&signed.UnsignedCorim, digest, label, activate)
	} else if slices.Equal(buf[:3], []byte{0xd9, 0x01, 0xf5}) {
		// tag 501 -> unsigned corim
		var unsigned corim.UnsignedCorim
		if err := unsigned.FromCBOR(buf); err != nil {
			return err
		}

		return o.AddCoRIM(&unsigned, digest, label, activate)
	} else {
		return fmt.Errorf("unrecognized input format")
	}
}

// AddCoRIM adds the provided CoRIM to the store. The digest, if not nil,
// should be the digest of the CBOR token the provided CoRIM was decoded
// from. If activate is true, the contained triples will be activated
// before they are added.
func (o *Store) AddCoRIM(c *corim.UnsignedCorim, digest []byte, label string, activate bool) error {
	m, err := model.NewManifestFromCoRIM(c)
	if err != nil {
		return err
	}

	m.Digest = digest
	m.Label = label

	if activate {
		m.SetActive(true)
	}

	return o.AddManifest(m)
}

// AddManifest adds the provided manifest to the store.
func (o *Store) AddManifest(m *model.Manifest) error {
	var existing model.Manifest

	if o.cfg.RequireLabel && m.Label == "" {
		return ErrNoLabel
	}

	err := o.DB.NewSelect().Model(&existing).Where("manifest_id = ?", m.ManifestID).Scan(o.Ctx)
	if err == nil { // found
		if o.cfg.Force {
			// select the existing manifest to fully populate its fields
			if err := existing.Select(o.Ctx, o.DB); err != nil {
				return fmt.Errorf("error selecting existing manifest: %w", err)
			}

			tx, err := o.DB.BeginTx(o.Ctx, nil)
			if err != nil {
				return err
			}

			if err := existing.Delete(o.Ctx, tx); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("error deleting existing manifest: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return err
			}

		} else {
			if len(existing.Digest) != 0 && len(m.Digest) != 0 {
				if slices.Equal(existing.Digest, m.Digest) {
					return errors.New("already in store (digests match)")
				} else {
					return errors.New(
						"already in store but digests differ")
				}
			} else {
				return errors.New("already in store")
			}
		}
	} else if err != sql.ErrNoRows {
		return err
	}

	tx, err := o.DB.BeginTx(o.Ctx, nil)
	if err != nil {
		return err
	}

	m.TimeAdded = time.Now()

	if err := m.Insert(o.Ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetManifest returns the manifest associated with the specified manifest ID.
// This is the unique ID of the manifest extracted from its token (not the internal
// database entry ID).
func (o *Store) GetManifest(manifestID string, label string) (*model.Manifest, error) {
	var ret model.Manifest

	query := o.DB.NewSelect().Model(&ret).Where("manifest_id = ?", manifestID)
	if label != "" {
		query.Where("label = ?", label)
	} else if o.cfg.RequireLabel {
		return nil, ErrNoLabel
	}

	err := query.Scan(o.Ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("manifest with ID %q not found ", manifestID)
		}

		return nil, err
	}

	// fully populate nested structures
	if err := ret.Select(o.Ctx, o.DB); err != nil {
		return nil, err
	}

	return &ret, nil
}

// DeteleManifest deletes the manifest associated with the specified manifest ID,
// and all its data, from the store. The ID is the unique ID of the manifest
// extracted from its token (not the internal database entry ID).
func (o *Store) DeleteManifest(manifestID string, label string) error {
	manifest, err := o.GetManifest(manifestID, label)
	if err != nil {
		return err
	}

	return manifest.Delete(o.Ctx, o.DB)
}

// GetActiveValueTriples returns a slice of ValueTriple's whose environment
// matches the one provided. If exact is true, any unset fields in the provided
// environment must be NULL in the database; otherwise, unset fields will
// match any value. Only active triples are returned.
func (o *Store) GetActiveValueTriples(
	env *comid.Environment,
	label string,
	exact bool,
) ([]*comid.ValueTriple, error) {
	query := NewValueTripleQuery().
		Label(label).
		ExactEnvironment(exact).
		IsActive(true).
		ValidOn(time.Now())

	if err := query.EnvironmentSubquery().UpdateFromCoRIM(env); err != nil {
		return nil, err
	}

	return o.QueryValueTriples(query)
}

// GetActiveKeyTriples returns a slice of KeyTriple's whose environment matches
// the one provided. If exact is true, any unset fields in the provided
// environment must be NULL in the database; otherwise, unset fields will
// match any value. Only active triples are returned.
func (o *Store) GetActiveKeyTriples(
	env *comid.Environment,
	label string,
	exact bool,
) ([]*comid.KeyTriple, error) {
	query := NewKeyTripleQuery().
		Label(label).
		ExactEnvironment(exact).
		IsActive(true).
		ValidOn(time.Now())

	if err := query.EnvironmentSubquery().UpdateFromCoRIM(env); err != nil {
		return nil, err
	}

	return o.QueryKeyTriples(query)
}

// QueryManifestEntries returns ManifestEntry's that match the provided query.
// If the query is empty or is nil, all ManifestEntry's in the Store are
// returned. Unlike Manifest models (see below), ManifestEntry's only contain
// fields related to the manifest itself and not any of its contents
// (ModuleTags, DependentRIMs, etc).
func (o *Store) QueryManifestEntries(query Query[*model.ManifestEntry]) ([]*model.ManifestEntry, error) {
	if query == nil {
		query = NewManifestQuery()
	}

	return query.Run(o.Ctx, o.DB)
}

// QueryManifestModels returns Manifest models that match the provided query.
// If the query is empty or is nil, all Manifest's in the Store are returned.
// The returned Manifest models are fully populated, incuding their contained
// structures such as ModuleTag's (contrast with QueryManifestEntries above).
func (o *Store) QueryManifestModels(query Query[*model.ManifestEntry]) ([]*model.Manifest, error) {
	entries, err := o.QueryManifestEntries(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.Manifest, len(entries))
	for i, entry := range entries {
		moduleTag, err := entry.ToManifest(o.Ctx, o.DB)
		if err != nil {
			return nil, fmt.Errorf("manifest ID %d: %w", moduleTag.ID, err)
		}

		ret[i] = moduleTag
	}

	return ret, nil
}

// QueryCoRIMs returns corim.UnsignedCorim's that match the provided query.
// Note that these are reconstructed from Store content, and are not parsed
// form the tokens that were originally added to the store.
func (o *Store) QueryCoRIMs(query Query[*model.ManifestEntry]) ([]*corim.UnsignedCorim, error) {
	models, err := o.QueryManifestModels(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*corim.UnsignedCorim, len(models))
	for i, model := range models {
		c, err := model.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("manifest ID %d: %w", model.ID, err)
		}

		ret[i] = c
	}

	return ret, nil
}

// QueryModuleTagEntries returns ModuleTagEntry's that match the provided query.
// If the query is empty or is nil, all ModuleTagEntry's in the Store are
// returned. Unlike ModuleTag models (see below), ModuleTagEntry's only contain
// fields related to the module tag itself and not any of its contents
// (triples, linked tags, etc).
func (o *Store) QueryModuleTagEntries(query Query[*model.ModuleTagEntry]) ([]*model.ModuleTagEntry, error) {
	if query == nil {
		query = NewModuleTagQuery()
	}

	return query.Run(o.Ctx, o.DB)
}

// QueryModuleTagModels returns ModuleTag models that match the provided query.
// If the query is empty or is nil, all ModuleTag's in the Store are returned.
// The returned ModuleTag models are fully populated, incuding their contained
// structures such as ValueTriple's (contrast with QueryModuleTagEntries above).
func (o *Store) QueryModuleTagModels(query Query[*model.ModuleTagEntry]) ([]*model.ModuleTag, error) {
	entries, err := o.QueryModuleTagEntries(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.ModuleTag, len(entries))
	for i, entry := range entries {
		moduleTag, err := entry.ToModuleTag(o.Ctx, o.DB)
		if err != nil {
			return nil, fmt.Errorf("module tag ID %d: %w", moduleTag.ID, err)
		}

		ret[i] = moduleTag
	}

	return ret, nil
}

// QueryCoMIDs returns comid.Comid's that match the provided query. Note that
// these are reconstructed from Store content, and are not parsed form the
// tokens that were originally added to the store.
func (o *Store) QueryCoMIDs(query Query[*model.ModuleTagEntry]) ([]*comid.Comid, error) {
	models, err := o.QueryModuleTagModels(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*comid.Comid, len(models))
	for i, model := range models {
		c, err := model.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("module tag ID %d: %w", model.ID, err)
		}

		ret[i] = c
	}

	return ret, nil
}

// QueryEntityModels returns Entity models that match the provided query.
// If the query is empty or is nil, all Entity's in the Store are returned.
func (o *Store) QueryEntityModels(query Query[*model.Entity]) ([]*model.Entity, error) {
	if query == nil {
		query = NewEntityQuery()
	}

	models, err := query.Run(o.Ctx, o.DB)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if err := model.Select(o.Ctx, o.DB); err != nil {
			return nil, fmt.Errorf("entity ID %d: %w", model.ID, err)
		}
	}

	return models, nil
}

// QueryCoRIMEntities returns corim.Entity's that match the provided query.
// Note that these are reconstructed from Store content, and are not parsed
// form the tokens that were originally added to the store.
func (o *Store) QueryCoRIMEntities(query *EntityQuery) ([]*corim.Entity, error) {
	if query == nil {
		query = NewEntityQuery()
	}

	models, err := o.QueryEntityModels(query.OwnerType("manifest"))
	if err != nil {
		return nil, err
	}

	ret := make([]*corim.Entity, len(models))
	for i, model := range models {
		entity, err := model.ToCoRIMCoRIM()
		if err != nil {
			return nil, fmt.Errorf("entity ID %d: %w", model.ID, err)
		}

		ret[i] = entity
	}

	return ret, nil
}

// QueryCoMIDEntities returns comid.Entity's that match the provided query.
// Note that these are reconstructed from Store content, and are not parsed
// form the tokens that were originally added to the store.
func (o *Store) QueryCoMIDEntities(query *EntityQuery) ([]*comid.Entity, error) {
	if query == nil {
		query = NewEntityQuery()
	}

	models, err := o.QueryEntityModels(query.OwnerType("module_tag"))
	if err != nil {
		return nil, err
	}

	ret := make([]*comid.Entity, len(models))
	for i, model := range models {
		entity, err := model.ToCoMIDCoRIM()
		if err != nil {
			return nil, fmt.Errorf("entity ID %d: %w", model.ID, err)
		}

		ret[i] = entity
	}

	return ret, nil
}

// QueryEnvironmentModels returns Environment models that match the provided
// query. If the query is empty or is nil, all Environment's in the Store are
// returned.
func (o *Store) QueryEnvironmentModels(query Query[*model.Environment]) ([]*model.Environment, error) {
	if query == nil {
		query = NewEnvironmentQuery(false)
	}

	return query.Run(o.Ctx, o.DB)
}

// QueryEnvironments returns comid.Environment's that match the provided query.
// Note that these are reconstructed from Store content, and are not parsed
// form the tokens that were originally added to the store.
func (o *Store) QueryEnvironments(query Query[*model.Environment]) ([]*comid.Environment, error) {
	models, err := o.QueryEnvironmentModels(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*comid.Environment, len(models))
	for i, model := range models {
		environment, err := model.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("environment ID %d: %w", model.ID, err)
		}

		ret[i] = environment
	}

	return ret, nil
}

// QueryKeyTripleEntries returns a []*model.KeyTripleEntry with entries matching
// the provided query. If the query is nil or is empty, all KeyTripleEntry's in
// the store will be returned.
func (o *Store) QueryKeyTripleEntries(query Query[*model.KeyTripleEntry]) ([]*model.KeyTripleEntry, error) {
	if query == nil {
		query = NewKeyTripleQuery()
	}

	return query.Run(o.Ctx, o.DB)
}

// QueryKeyTripleModels returns a []*model.KeyTriple containing triples matching
// the provided query. If the query is nil or is empty, all KeyTriple's in the
// store will be returned.
// Note: this will run additional queries to fully populate matching triples.
// For queries expecting large results, it is recommended to, where possible,
// use QueryKeyTripleEntries instead.
func (o *Store) QueryKeyTripleModels(query Query[*model.KeyTripleEntry]) ([]*model.KeyTriple, error) {
	entries, err := o.QueryKeyTripleEntries(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.KeyTriple, len(entries))
	for i, entry := range entries {
		triple, err := entry.ToTriple(o.Ctx, o.DB)
		if err != nil {
			return nil, fmt.Errorf("key triple ID %d: %w", entry.TripleDbID, err)
		}

		ret[i] = triple
	}

	return ret, nil
}

// QueryKeyTriples returns a []*comid.KeyTriples with triples matching provided
// query.
func (o *Store) QueryKeyTriples(query Query[*model.KeyTripleEntry]) ([]*comid.KeyTriple, error) {
	models, err := o.QueryKeyTripleModels(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*comid.KeyTriple, len(models))
	for i, model := range models {
		triple, err := model.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("value truple with ID %d: %w", model.ID, err)
		}

		ret[i] = triple
	}

	return ret, nil
}

// QueryValueTripleEntries returns a []*model.ValueTripleEntry with entries matching
// the provided query. If the query is nil or is empty, all ValueTripleEntry's in
// the store will be returned.
func (o *Store) QueryValueTripleEntries(query Query[*model.ValueTripleEntry]) ([]*model.ValueTripleEntry, error) {
	if query == nil {
		query = NewValueTripleQuery()
	}
	return query.Run(o.Ctx, o.DB)
}

// QueryValueTripleModels returns a []*model.ValueTriple containing triples matching
// the provided query. If the query is nil or is empty, all ValueTriple's in the
// store will be returned.
// Note: this will run additional queries to fully populate matching triples.
// For queries expecting large results, it is recommended to, where possible,
// use QueryValueTripleEntries instead.
func (o *Store) QueryValueTripleModels(query Query[*model.ValueTripleEntry]) ([]*model.ValueTriple, error) {
	entries, err := o.QueryValueTripleEntries(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*model.ValueTriple, len(entries))
	for i, entry := range entries {
		triple, err := entry.ToTriple(o.Ctx, o.DB)
		if err != nil {
			return nil, fmt.Errorf("entry for value triple with ID %d: %w", entry.TripleDbID, err)
		}

		ret[i] = triple
	}

	return ret, nil
}

// QueryValueTriples returns a []*comid.ValueTriples with triples matching
// provided query.
func (o *Store) QueryValueTriples(query Query[*model.ValueTripleEntry]) ([]*comid.ValueTriple, error) {
	models, err := o.QueryValueTripleModels(query)
	if err != nil {
		return nil, err
	}

	ret := make([]*comid.ValueTriple, len(models))
	for i, model := range models {
		triple, err := model.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("value truple with ID %d: %w", model.ID, err)
		}

		ret[i] = triple
	}

	return ret, nil
}

// SetKeyTriplesActive sets the active status of key triples matching the
// specified query to the specified value. On success a slice of entries
// corresponding to the updated triples is returned. The IsActive field of
// these entries is set to their old value prior to the update.
func (o *Store) SetKeyTriplesActive(
	query Query[*model.KeyTripleEntry],
	value bool,
) ([]*model.KeyTripleEntry, error) {
	entries, err := o.QueryKeyTripleEntries(query)
	if err != nil {
		return nil, err
	}

	_, err = o.DB.NewUpdate().
		Table("key_triples").
		Set("is_active = ?", value).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			for _, entry := range entries {
				q.WhereOr("id = ?", entry.TripleDbID)
			}

			return q
		}).
		Exec(o.Ctx)

	if err != nil {
		return nil, err
	}

	return entries, nil
}

// SetValueTriplesActive sets the active status of value triples matching the
// specified query to the specified value. On success a slice of entries
// corresponding to the updated triples is returned. The IsActive field of
// these entries is set to their old value prior to the update.
func (o *Store) SetValueTriplesActive(
	query Query[*model.ValueTripleEntry],
	value bool,
) ([]*model.ValueTripleEntry, error) {
	entries, err := o.QueryValueTripleEntries(query)
	if err != nil {
		return nil, err
	}

	_, err = o.DB.NewUpdate().
		Table("value_triples").
		Set("is_active = ?", value).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			for _, entry := range entries {
				q.WhereOr("id = ?", entry.TripleDbID)
			}

			return q
		}).
		Exec(o.Ctx)

	if err != nil {
		return nil, err
	}

	return entries, nil
}

// Clear removes all data from store (effectively truncating the tables
// containing CoRIM/CoMID data).
func (o *Store) Clear() error {
	db, ok := o.DB.(*bun.DB)
	if !ok {
		return errors.New("cannot Clear via transaction")
	}

	return model.ResetModels(o.Ctx, db)
}

// StringAggregatorExpr returns an expression using a dialect-specific
// function to aggregate the specified column (must be TEXT) into a
// comma-separated list.
func (o *Store) StringAggregatorExpr(columnName string) string {
	dialect := o.DB.Dialect().Name().String()
	return StringAggregatorExprForDialect(dialect, columnName)
}

// ConcatExpr returns dialect-specific expression concatenated provided
// strings.
func (o *Store) ConcatExpr(tokens ...string) string {
	dialect := o.DB.Dialect().Name().String()
	return ConcatExprForDialect(dialect, tokens...)
}

// HexExpr returns a dialect-specific expression to perform hex encoding
// on the specified column.
func (o *Store) HexExpr(columnName string) string {
	dialect := o.DB.Dialect().Name().String()
	return HexExprForDialect(dialect, columnName)
}

// Digest computes the digests of the provided buffer using the store's
// configured hashing algorithm.
func (o *Store) Digest(input []byte) []byte {
	switch o.cfg.HashAlg {
	case "md5", "MD5":
		hash := md5.Sum(input)
		return hash[:]
	case "sha256", "SHA256":
		hash := sha256.Sum256(input)
		return hash[:]
	case "sha512", "SHA512":
		hash := sha512.Sum512(input)
		return hash[:]
	default:
		// cfg was validated on creation so we should never get here
		panic(fmt.Sprintf("invalid hash algorithm: %s", o.cfg.HashAlg))
	}
}

func StringAggregatorExprForDialect(dialect, columnName string) string {
	switch dialect {
	case "pg":
		return fmt.Sprintf("STRING_AGG(%s, ', ')", columnName)
	case "mysql":
		return fmt.Sprintf("GROUP_CONCAT(%s SEPARATOR ', ')", columnName)
	case "sqlite":
		return fmt.Sprintf("GROUP_CONCAT(%s, ', ')", columnName)
	default:
		// It should be impossible to instantiate a Store with an
		// unsupported dialect.
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}
}

func ConcatExprForDialect(dialect string, tokens ...string) string {
	if len(tokens) == 0 {
		return "''"
	}

	switch dialect {
	case "mysql":
		return fmt.Sprintf("CONCAT(%s)", strings.Join(tokens, ", "))
	case "pg", "sqlite":
		return strings.Join(tokens, " || ")
	default:
		// It should be impossible to instantiate a Store with an
		// unsupported dialect.
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}
}

func HexExprForDialect(dialect, columnName string) string {

	switch dialect {
	case "mysql", "sqlite":
		return fmt.Sprintf("hex(%s)", columnName)
	case "pg":
		return fmt.Sprintf("encode(%s, 'hex')", columnName)
	default:
		// It should be impossible to instantiate a Store with an
		// unsupported dialect.
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}
}
