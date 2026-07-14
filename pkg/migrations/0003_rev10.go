package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/uptrace/bun"
)

const CREATE_KEY_TRIPLE_VIEW_REV10_SQL = `
CREATE VIEW key_triple_entries AS
SELECT
  kt.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  kt.environment_id AS environment_db_id,
  kt.type AS triple_type,
  kt.cond_key_type AS cond_key_type,
  kt.cond_key_bytes AS cond_key_bytes,
  kt.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  key_triples AS kt
INNER JOIN module_tags AS mt
  ON kt.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const CREATE_VALUE_TRIPLE_VIEW_REV10_SQL = `
CREATE VIEW value_triple_entries AS
SELECT
  vt.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  vt.environment_id AS environment_db_id,
  vt.type AS triple_type,
  vt.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  value_triples AS vt
INNER JOIN module_tags AS mt
  ON vt.owner_id = mt.id AND vt.owner_type = 'module_tag'
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const CREATE_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL = `
CREATE VIEW domain_dependency_triple_entries AS
SELECT
  ddt.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  ddt.environment_id AS environment_db_id,
  ddt.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  domain_dependency_triples AS ddt
INNER JOIN module_tags AS mt
  ON ddt.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL = `DROP VIEW domain_dependency_triple_entries`

const CREATE_DOMAIN_MEMBERSHIP_TRIPLE_VIEW_SQL = `
CREATE VIEW domain_membership_triple_entries AS
SELECT
  dmt.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  dmt.environment_id AS environment_db_id,
  dmt.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  domain_membership_triples AS dmt
INNER JOIN module_tags AS mt
  ON dmt.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_DOMAIN_MEMBERSHIP_TRIPLE_VIEW_SQL = `DROP VIEW domain_membership_triple_entries`

const CREATE_CONDITIONAL_ENDORSEMENT_TRIPLE_VIEW_SQL = `
CREATE VIEW conditional_endorsement_triple_entries AS
SELECT
  cet.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  cet.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  conditional_endorsement_triples AS cet
INNER JOIN module_tags AS mt
  ON cet.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_CONDITIONAL_ENDORSEMENT_TRIPLE_VIEW_SQL = `DROP VIEW conditional_endorsement_triple_entries`

const CREATE_CONDITIONAL_ENDORSEMENT_SERIES_TRIPLE_VIEW_SQL = `
CREATE VIEW conditional_endorsement_series_triple_entries AS
SELECT
  cest.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  cest.environment_id AS environment_db_id,
  cest.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  conditional_endorsement_series_triples AS cest
INNER JOIN module_tags AS mt
  ON cest.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_CONDITIONAL_ENDORSEMENT_SERIES_TRIPLE_VIEW_SQL = `DROP VIEW conditional_endorsement_series_triple_entries`

type StatementMap map[string]string

func execStatement(db *bun.DB, statementMap StatementMap, args ...any) (sql.Result, error) {
	dialect := db.Dialect().Name().String()

	for dialects, statement := range statementMap {
		if slices.Contains(strings.Split(dialects, "|"), dialect) {
			return db.Exec(statement, args...)
		}
	}

	return nil, fmt.Errorf("unhandled dialect %q", dialect)
}

type href_v1 struct {
	bun.BaseModel `bun:"table:hrefs,alias:href"`

	ID int64 `bun:",pk,autoincrement"`

	Value string `bun:"value"`

	LocatorID int64 `bun:"locator_id"`
}

type statefulEnvironment_v1 struct {
	bun.BaseModel `bun:"table:stateful_environments,alias:senv"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64           `bun:",nullzero"`
	Environment   *environment_v1 `bun:"rel:belongs-to,join:environment_id=id"`

	Measurements []*measurement_v1 `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:stateful_environment"`

	TripleID int64 `bun:",nullzero"`
}

type conditionalEndorsementTriple_v1 struct {
	bun.BaseModel `bun:"table:conditional_endorsement_triples,alias:cet"`

	ID int64 `bun:",pk,autoincrement"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

type conditionalEndorsementSeriesRecord_v1 struct {
	bun.BaseModel `bun:"table:conditional_endorsement_series_records,alias:cesr"`

	ID int64 `bun:",pk,autoincrement"`

	TripleID int64
}

type conditionalEndorsementSeriesTriple_v1 struct {
	bun.BaseModel `bun:"table:conditional_endorsement_series_triples,alias:cest"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64 `bun:",nullzero"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

type domainEntry_v1 struct {
	bun.BaseModel `bun:"table:domain_entries,alias:de"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64 `bun:",nullzero"`

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

type domainMembershipTriple_v1 struct {
	bun.BaseModel `bun:"table:domain_membership_triples,alias:dmt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64 `bun:",nullzero"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

type domainDependencyTriple_v1 struct {
	bun.BaseModel `bun:"table:domain_dependency_triples,alias:ddt"`

	ID int64 `bun:",pk,autoincrement"`

	EnvironmentID int64 `bun:",nullzero"`

	IsActive bool

	ModuleID int64 `bun:",nullzero"`
}

var newModels_rev10 = [][2]any{
	{"hrefs", (*href_v1)(nil)},
	{"stateful_environments", (*statefulEnvironment_v1)(nil)},
	{"conditional_endorsement_triples", (*conditionalEndorsementTriple_v1)(nil)},
	{"conditional_endorsement_series_records", (*conditionalEndorsementSeriesRecord_v1)(nil)},
	{"conditional_endorsement_series_triples", (*conditionalEndorsementSeriesTriple_v1)(nil)},
	{"domain_entries", (*domainEntry_v1)(nil)},
	{"domain_dependency_triples", (*domainDependencyTriple_v1)(nil)},
	{"domain_membership_triples", (*domainMembershipTriple_v1)(nil)},
}

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		var err error

		for _, entry := range newModels_rev10 {
			_, err = db.NewCreateTable().Model(entry[1]).IfNotExists().Exec(ctx)
			if err != nil {
				return fmt.Errorf("%s: %w", entry[0].(string), err)
			}
		}

		statements := []StatementMap{
			{
				"pg|sqlite|mysql": `
					INSERT INTO hrefs(value, locator_id)
					SELECT href, id
					FROM locators
					`,
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE locators DROP COLUMN href",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE digests RENAME COLUMN alg_id TO alg_id_int",
			},
			{
				"pg|sqlite": "ALTER TABLE digests ADD COLUMN alg_id_text TEXT",
				"mysql":     "ALTER TABLE digests ADD COLUMN alg_id_text VARCHAR(255)",
			},
			{
				"pg|sqlite": "ALTER TABLE key_triples ADD COLUMN cond_key_type TEXT",
				"mysql":     "ALTER TABLE key_triples ADD COLUMN cond_key_type VARCHAR(255)",
			},
			{
				"mysql|sqlite": "ALTER TABLE key_triples ADD COLUMN cond_key_bytes BLOB",
				"pg":           "ALTER TABLE key_triples ADD COLUMN cond_key_bytes BYTEA",
			},
			{
				"pg|sqlite|mysql": DROP_KEY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_KEY_TRIPLE_VIEW_REV10_SQL,
			},
			{
				"pg|sqlite": "ALTER TABLE value_triples ADD COLUMN owner_type TEXT",
				"mysql":     "ALTER TABLE value_triples ADD COLUMN owner_type VARCHAR(255)",
			},
			{
				"pg|sqlite|mysql": "UPDATE value_triples SET owner_type = 'module_tag'",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE value_triples RENAME COLUMN module_id TO owner_id",
			},
			{
				"pg|sqlite|mysql": DROP_VALUE_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_VALUE_TRIPLE_VIEW_REV10_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_CONDITIONAL_ENDORSEMENT_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_CONDITIONAL_ENDORSEMENT_SERIES_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_DOMAIN_MEMBERSHIP_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": "UPDATE cryptokeys SET owner_type = 'measurement_auth' WHERE owner_type = 'measurement'",
			},
		}

		for _, statementMap := range statements {
			_, err = execStatement(db, statementMap)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		var err error
		statements := []StatementMap{
			{
				"pg|sqlite|mysql": "DELETE FROM cryptokeys WHERE owner_type = 'measurement'",
			},
			{
				"pg|sqlite|mysql": "UPDATE cryptokeys SET owner_type = 'measurement' WHERE owner_type = 'measurement_auth'",
			},
			{
				"pg|sqlite|mysql": DROP_KEY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_KEY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE key_triples DROP COLUMN cond_key_bytes",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE key_triples DROP COLUMN cond_key_type",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE digests DROP COLUMN alg_id_text",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE digests RENAME COLUMN alg_id_int TO alg_id",
			},
			{
				"pg|sqlite": "ALTER TABLE locators ADD COLUMN href TEXT",
				"mysql":     "ALTER TABLE locators ADD COLUMN href VARCHAR(255)",
			},
			{
				"sqlite|pg": `
					UPDATE locators
					SET href = hrefs.value
					FROM hrefs
					WHERE locators.id = hrefs.locator_id
					`,
				"mysql": `
					UPDATE locators
					JOIN hrefs ON locators.id = hrefs.locator_id
					SET locators.href = hrefs.value
					`,
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE value_triples RENAME COLUMN owner_id TO module_id",
			},
			{
				"pg|sqlite|mysql": "DELETE FROM value_triples WHERE owner_type <> 'module_tag'",
			},
			{
				"pg|sqlite|mysql": DROP_VALUE_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_VALUE_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE value_triples DROP COLUMN owner_type",
			},
			{
				"pg|sqlite|mysql": DROP_CONDITIONAL_ENDORSEMENT_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": DROP_CONDITIONAL_ENDORSEMENT_SERIES_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": DROP_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": DROP_DOMAIN_MEMBERSHIP_TRIPLE_VIEW_SQL,
			},
		}

		for _, statementMap := range statements {
			_, err = execStatement(db, statementMap)
			if err != nil {
				return err
			}
		}

		for _, entry := range slices.Backward(newModels_rev10) {
			_, err = db.NewDropTable().Model(entry[1]).IfExists().Exec(ctx)
			if err != nil {
				return fmt.Errorf("%s: %w", entry[0].(string), err)
			}
		}

		return nil
	})
}
