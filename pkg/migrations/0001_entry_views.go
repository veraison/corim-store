package migrations

import (
	"context"

	"github.com/uptrace/bun"
	dbpkg "github.com/veraison/corim-store/pkg/db"
)

const CREATE_MANIFEST_VIEW_SQL = `
CREATE VIEW manifest_entries AS
SELECT
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.digest AS digest,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  manifests AS mft
;
`

const DROP_MANIFEST_VIEW_SQL = `DROP VIEW manifest_entries`

const CREATE_MODULE_TAG_VIEW_SQL = `
CREATE VIEW module_tag_entries AS
SELECT
  mt.id AS module_tag_db_id,
  mft.id AS manifest_db_id,
  mft.manifest_id_type AS manifest_id_type,
  mft.manifest_id AS manifest_id,
  mt.tag_id_type AS module_tag_id_type,
  mt.tag_id AS module_tag_id,
  mt.tag_version AS module_tag_version,
  mt.language AS language,
  mft.label AS label,
  mft.profile_type AS profile_type,
  mft.profile AS profile,
  mft.time_added AS time_added,
  mft.not_before AS not_before,
  mft.not_after AS not_after
FROM  module_tags AS mt
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_MODULE_TAG_VIEW_SQL = `DROP VIEW module_tag_entries`

const CREATE_KEY_TRIPLE_VIEW_SQL = `
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

const DROP_KEY_TRIPLE_VIEW_SQL = `DROP VIEW key_triple_entries`

const CREATE_VALUE_TRIPLE_VIEW_SQL = `
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
  ON vt.module_id = mt.id
INNER JOIN manifests AS mft
  ON mt.manifest_id = mft.id
;
`

const DROP_VALUE_TRIPLE_VIEW_SQL = `DROP VIEW value_triple_entries`

func init() {
	createSQL := []string{
		CREATE_MANIFEST_VIEW_SQL,
		CREATE_MODULE_TAG_VIEW_SQL,
		CREATE_KEY_TRIPLE_VIEW_SQL,
		CREATE_VALUE_TRIPLE_VIEW_SQL,
	}

	deleteSQL := []string{
		DROP_VALUE_TRIPLE_VIEW_SQL,
		DROP_KEY_TRIPLE_VIEW_SQL,
		DROP_MODULE_TAG_VIEW_SQL,
		DROP_MANIFEST_VIEW_SQL,
	}

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		return dbpkg.ExecTx(ctx, db, nil, createSQL)
	}, func(ctx context.Context, db *bun.DB) error {
		return dbpkg.ExecTx(ctx, db, nil, deleteSQL)
	})
}
