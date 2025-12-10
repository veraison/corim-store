package cmd

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/store"
)

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate a (set of) triple(s), making them available to the verifier.",
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(setActiveCommand(cmd, true))
	},
}

var deactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate a (set of) triple(s), making them unavailable to the verifier.",
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(setActiveCommand(cmd, false))
	},
}

func setActiveCommand(cmd *cobra.Command, value bool) error {
	store, err := store.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	keyTripleIDs, err := cmd.Flags().GetInt64Slice("key-triple")
	if err != nil {
		return err
	}

	if len(keyTripleIDs) != 0 {
		if err := keyTriplesSetActive(store, keyTripleIDs, value); err != nil {
			return fmt.Errorf("key triples: %w", err)
		}
	}

	valueTripleIDs, err := cmd.Flags().GetInt64Slice("value-triple")
	if err != nil {
		return err
	}

	if len(valueTripleIDs) != 0 {
		if err := valueTriplesSetActive(store, valueTripleIDs, value); err != nil {
			return fmt.Errorf("value triples: %w", err)
		}
	}

	moduleTagTextIDs, err := cmd.Flags().GetStringSlice("module-tag")
	if err != nil {
		return err
	}

	if len(moduleTagTextIDs) != 0 {
		var moduleTagIDs []int64
		for _, idText := range moduleTagTextIDs {
			var ids []int64
			err := store.DB.NewSelect().
				TableExpr("module_tags as mod").
				ColumnExpr("mod.id as id").
				Where("mod.tag_id = ?", idText).
				Scan(store.Ctx, &ids)

			if err != nil {
				return err
			}

			moduleTagIDs = append(moduleTagIDs, ids...)
		}

		if err := moduleTagsSetActive(store, moduleTagIDs, value); err != nil {
			return fmt.Errorf("module tags: %w", err)
		}
	}

	manifestTextIDs, err := cmd.Flags().GetStringSlice("manifest")
	if err != nil {
		return err
	}

	if len(manifestTextIDs) != 0 {
		var moduleTagIDs []int64
		for _, idText := range manifestTextIDs {
			var ids []int64
			err = store.DB.NewSelect().
				TableExpr("module_tags as mod").
				ColumnExpr("mod.id as id").
				Join("JOIN manifests AS man ON man.id = mod.manifest_id").
				Where("man.manifest_id = ?", idText).
				Scan(store.Ctx, &ids)

			if err != nil {
				return err
			}

			moduleTagIDs = append(moduleTagIDs, ids...)
		}

		if err := moduleTagsSetActive(store, moduleTagIDs, value); err != nil {
			return fmt.Errorf("manifests: %w", err)
		}
	}

	return nil
}

func moduleTagsSetActive(store *store.Store, ids []int64, value bool) error {
	for _, id := range ids {
		var keyTripleIDs []int64
		noKeyTriples := false

		err := store.DB.NewSelect().
			Model(&keyTripleIDs).
			TableExpr("key_triples as kt").
			Column("id").
			Where("kt.module_id = ?", id).
			Scan(store.Ctx)

		if err != nil && err != sql.ErrNoRows { // nolint:gocritic
			return fmt.Errorf("scanning key triples: %w", err)
		} else if err == sql.ErrNoRows || len(keyTripleIDs) == 0 {
			noKeyTriples = true
		} else {
			if err := keyTriplesSetActive(store, keyTripleIDs, true); err != nil {
				return err
			}
		}

		var valueTripleIDs []int64
		noValueTriples := false

		err = store.DB.NewSelect().
			Model(&valueTripleIDs).
			TableExpr("value_triples as vt").
			Column("id").
			Where("vt.module_id = ?", id).
			Scan(store.Ctx)

		if err != nil && err != sql.ErrNoRows { // nolint:gocritic
			return fmt.Errorf("scanning value triples: %w", err)
		} else if err == sql.ErrNoRows || len(valueTripleIDs) == 0 {
			noValueTriples = true
		} else {
			if err := valueTriplesSetActive(store, valueTripleIDs, true); err != nil {
				return err
			}
		}

		if noKeyTriples && noValueTriples {
			return fmt.Errorf("no triples associated with module tag ID %d", id)
		}
	}

	return nil
}

func keyTriplesSetActive(store *store.Store, ids []int64, value bool) error {
	return setActive[model.KeyTriple](store, ids, value)
}

func valueTriplesSetActive(store *store.Store, ids []int64, value bool) error {
	return setActive[model.ValueTriple](store, ids, value)
}

func setActive[T any](store *store.Store, ids []int64, value bool) error {
	_, err := store.DB.NewUpdate().
		Model((*T)(nil)).
		Set("is_active = ?", value).
		Where("id IN (?)", bun.In(ids)).
		Exec(store.Ctx)

	return err
}

func addActivateFlags(cmd *cobra.Command) {
	cmd.Flags().Int64Slice("key-triple", []int64{}, "Specify the key triple database ID.")
	cmd.Flags().Int64Slice("value-triple", []int64{}, "Specify the key triple database ID.")
	cmd.Flags().StringSlice("module-tag", []string{}, "Specify the module tag UUID.")
	cmd.Flags().StringSlice("manifest", []string{}, "Specify the manifest ID.")
}

func init() {
	addActivateFlags(activateCmd)
	addActivateFlags(deactivateCmd)
	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(deactivateCmd)
}
