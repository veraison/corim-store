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
var ErrNoMatch = errors.New("no triples matched")
var ErrNoEnvMatch = errors.New("no matching environments found")

type Store struct {
	Ctx context.Context
	DB  *bun.DB

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

// Close the Store, including its database connection.
func (o *Store) Close() error {
	return o.DB.Close()
}

// Init inializes a new database with the store's tables
func (o *Store) Init() error {
	migrator := migrate.NewMigrator(o.DB, migrations.Migrations)

	if err := migrator.Init(o.Ctx); err != nil {
		return err
	}

	return o.Migrate()
}

// Migrate upates the tables in the associated database to be compatible with this store.
// (note: there is no need to run this after invoking Store.Init().)
func (o *Store) Migrate() error {
	migrator := migrate.NewMigrator(o.DB, migrations.Migrations)

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
) ([]*model.ValueTriple, error) {
	return getTriples[*model.ValueTriple](o, env, label, exact, true)
}

// GetValueTriples returns a slice of ValueTriple's whose environment matches
// the one provided. If exact is true, any unset fields in the provided
// environment must be NULL in the database; otherwise, unset fields will
// match any value.
func (o *Store) GetValueTriples(
	env *comid.Environment,
	label string,
	exact bool,
) ([]*model.ValueTriple, error) {
	return getTriples[*model.ValueTriple](o, env, label, exact, false)
}

// GetActiveKeyTriples returns a slice of KeyTriple's whose environment matches
// the one provided. If exact is true, any unset fields in the provided
// environment must be NULL in the database; otherwise, unset fields will
// match any value. Only active triples are returned.
func (o *Store) GetActiveKeyTriples(
	env *comid.Environment,
	label string,
	exact bool,
) ([]*model.KeyTriple, error) {
	return getTriples[*model.KeyTriple](o, env, label, exact, true)
}

// GetKeyTriples returns a slice of KeyTriple's whose environment matches
// the one provided. If exact is true, any unset fields in the provided
// environment must be NULL in the database; otherwise, unset fields will
// match any value.
func (o *Store) GetKeyTriples(
	env *comid.Environment,
	label string,
	exact bool,
) ([]*model.KeyTriple, error) {
	return getTriples[*model.KeyTriple](o, env, label, exact, false)
}

func (o *Store) FindEnvironmentIDs(env *model.Environment, exact bool) ([]int64, error) {
	var ret []int64

	query := o.DB.NewSelect().Model(&ret).Table("environments").Column("id")
	model.UpdateSelectQueryFromEnvironment(query, env, exact)

	if err := query.Scan(o.Ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoEnvMatch
		}

		return nil, err
	}

	if len(ret) == 0 {
		return nil, ErrNoEnvMatch
	}

	return ret, nil
}

func (o *Store) FindModuleTagIDsForLabel(label string) ([]int64, error) {
	if label == "" {
		return nil, errors.New("no label specified")
	}

	var ret []int64

	err := o.DB.NewSelect().
		Model(&ret).
		TableExpr("module_tags AS mod").
		ColumnExpr("mod.id").
		Join("JOIN manifests AS man ON man.id = mod.manifest_id").
		Where("man.label = ?", label).
		Scan(o.Ctx)

	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Clear removes all data from store (effectively truncating the tables
// containing CoRIM/CoMID data).
func (o *Store) Clear() error {
	return model.ResetModels(o.Ctx, o.DB)
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

// In addition to implementing the methods described by the interface, a
// triple's database table must also contain environment_id and module_id
// fields.
type triple interface {
	TripleType() string
	DatabaseID() int64
	Select(ctx context.Context, db bun.IDB) error
}

func getTriples[T triple](
	store *Store,
	env *comid.Environment,
	label string,
	exact bool,
	activeOnly bool,
) ([]T, error) { // nolint:dupl
	var ret []T
	query := store.DB.NewSelect().Model(&ret)

	if activeOnly {
		query.Where("is_active = true")
	}

	modelEnv, err := model.NewEnvironmentFromCoRIM(env)
	if err != nil {
		return nil, err
	}

	if modelEnv != nil && !modelEnv.IsEmpty() {
		envIDs, err := store.FindEnvironmentIDs(modelEnv, exact)
		if err != nil {
			if errors.Is(err, ErrNoEnvMatch) {
				return nil, ErrNoMatch
			}

			return nil, err
		}

		query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, envID := range envIDs {
				q.WhereOr("environment_id = ?", envID)
			}

			return q
		})
	}

	if label != "" {
		modIDs, err := store.FindModuleTagIDsForLabel(label)
		if err != nil {
			return nil, err
		}

		query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			for _, modID := range modIDs {
				q.WhereOr("module_id = ?", modID)
			}

			return q
		})
	} else if store.cfg.RequireLabel {
		return nil, ErrNoLabel
	}

	if err := query.Scan(store.Ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoMatch
		}

		return nil, err
	}

	// fully load nested structures
	for _, triple := range ret {
		if err := triple.Select(store.Ctx, store.DB); err != nil {
			return nil, fmt.Errorf("%s triple with ID %d: %w",
				triple.TripleType(), triple.DatabaseID(), err)
		}
	}

	return ret, nil
}
