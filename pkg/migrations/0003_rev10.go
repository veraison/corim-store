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

type StatementMap map[string]string

func execStatment(db *bun.DB, statementMap StatementMap, args ...any) (sql.Result, error) {
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

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		var err error

		_, err = db.NewCreateTable().Model((*href_v1)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
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
		}

		for _, statementMap := range statements {
			_, err = execStatment(db, statementMap)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		var err error
		statements := []StatementMap{
			{
				"pg|sqlite|mysql": DROP_KEY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": CREATE_KEY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE digests DROP COLUMN cond_key_bytes",
			},
			{
				"pg|sqlite|mysql": "ALTER TABLE digests DROP COLUMN cond_key_type",
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
				"pg|sqlite": `
					UPDATE locators
					SET locators.href = hrefs.value
					FROM hrefs
					WHERE locators.id = hrefs.locator_id
					`,
				"mysql": `
					UPDATE locators
					JOIN hrefs ON locators.id = hrefs.locator_id
					SET locators.href = hrefs.value
					`,
			},
		}

		for _, statementMap := range statements {
			_, err = execStatment(db, statementMap)
			if err != nil {
				return err
			}
		}

		_, err = db.NewDropTable().Model((*href_v1)(nil)).IfExists().Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
