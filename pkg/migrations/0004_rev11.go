package migrations

import (
	"context"
	"fmt"
	"slices"

	"github.com/uptrace/bun"
)

const CREATE_TRUST_DEPENDENCY_TRIPLE_VIEW_SQL = `
CREATE VIEW trust_dependency_triple_entries AS
SELECT
  tdt.id AS triple_db_id,
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  tdt.environment_id AS environment_db_id,
  tdt.is_active AS is_active,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  trust_dependency_triples AS tdt
INNER JOIN module_tags AS mt
  ON tdt.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_TRUST_DEPENDENCY_TRIPLE_VIEW_SQL = `DROP VIEW trust_dependency_triple_entries`

type intMatcher_v1 struct {
	bun.BaseModel `bun:"table:int_matchers,alias:im"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
}

type intMatcherElement_v1 struct {
	bun.BaseModel `bun:"table:int_matcher_elements,alias:ime"`

	ID        int64 `bun:",pk,autoincrement"`
	Sign      int
	Value     uint64
	MatcherID int64
}

type floatMatcher_v1 struct {
	bun.BaseModel `bun:"table:float_matchers,alias:fm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
}

type floatMatcherElement_v1 struct {
	bun.BaseModel `bun:"table:float_matcher_elements,alias:fme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     float64
	MatcherID int64
}

type textMatcher_v1 struct {
	bun.BaseModel `bun:"table:text_matchers,alias:tm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
}

type textMatcherElement_v1 struct {
	bun.BaseModel `bun:"table:text_matcher_elements,alias:tme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     string
	MatcherID int64
}

type bytesMatcher_v1 struct {
	bun.BaseModel `bun:"table:bytes_matchers,alias:bm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
}

type bytesMatcherElement_v1 struct {
	bun.BaseModel `bun:"table:bytes_matcher_elements,alias:bme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     []byte
	MatcherID int64
}

var newModels_rev11 = [][2]any{
	{"int_matcher_elements", (*intMatcherElement_v1)(nil)},
	{"int_matchers", (*intMatcher_v1)(nil)},
	{"float_matcher_elements", (*floatMatcherElement_v1)(nil)},
	{"float_matchers", (*floatMatcher_v1)(nil)},
	{"text_matcher_elements", (*textMatcherElement_v1)(nil)},
	{"text_matchers", (*textMatcher_v1)(nil)},
	{"bytes_matcher_elements", (*bytesMatcherElement_v1)(nil)},
	{"bytes_matchers", (*bytesMatcher_v1)(nil)},
}

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		var err error

		for _, entry := range newModels_rev11 {
			_, err = db.NewCreateTable().Model(entry[1]).IfNotExists().Exec(ctx)
			if err != nil {
				return fmt.Errorf("%s: %w", entry[0].(string), err)
			}
		}

		statements := []StatementMap{
			{
				"pg|sqlite|mysql": `ALTER TABLE domain_dependency_triples
						    RENAME TO trust_dependency_triples`,
			},
			{
				"pg|sqlite|mysql": `UPDATE measurements
						    SET owner_type = 'ces_common_condition'
						    WHERE owner_type = 'ces_condition'`,
			},
			{
				"pg|sqlite|mysql": `UPDATE measurements
						    SET owner_type = 'ces_record_condition'
						    WHERE owner_type = 'ces_record_selection'`,
			},
			{
				"pg|sqlite|mysql": `UPDATE cryptokeys
						    SET owner_type = 'ces_common_condition'
						    WHERE owner_type = 'ces_condition'`,
			},
			{
				"pg|sqlite|mysql": CREATE_TRUST_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": DROP_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    ADD COLUMN int_matcher_id INT`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    ADD COLUMN float_matcher_id INT`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    ADD COLUMN bytes_matcher_id INT`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    ADD COLUMN text_matcher_id INT`,
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
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    DROP COLUMN bytes_matcher_id`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    DROP COLUMN text_matcher_id`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    DROP COLUMN float_matcher_id`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE measurements
						    DROP COLUMN int_matcher_id`,
			},
			{
				"pg|sqlite|mysql": `UPDATE cryptokeys
						    SET owner_type = 'ces_condition'
						    WHERE owner_type = 'ces_common_condition'`,
			},
			{
				"pg|sqlite|mysql": `UPDATE measurements
						    SET owner_type = 'ces_condition'
						    WHERE owner_type = 'ces_common_condition'`,
			},
			{
				"pg|sqlite|mysql": `UPDATE measurements
						    SET owner_type = 'ces_record_selection'
						    WHERE owner_type = 'ces_record_condition'`,
			},
			{
				"pg|sqlite|mysql": `ALTER TABLE trust_dependency_triples
						    RENAME TO domain_dependency_triples`,
			},
			{
				"pg|sqlite|mysql": CREATE_DOMAIN_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
			{
				"pg|sqlite|mysql": DROP_TRUST_DEPENDENCY_TRIPLE_VIEW_SQL,
			},
		}

		for _, statementMap := range statements {
			_, err = execStatement(db, statementMap)
			if err != nil {
				return err
			}
		}

		for _, entry := range slices.Backward(newModels_rev11) {
			_, err = db.NewDropTable().Model(entry[1]).IfExists().Exec(ctx)
			if err != nil {
				return fmt.Errorf("%s: %w", entry[0].(string), err)
			}
		}

		return nil
	})
}
