package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/migrate"
	"github.com/veraison/corim-store/pkg/db"
	"github.com/veraison/corim-store/pkg/migrations"
	"github.com/veraison/corim-store/pkg/model"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Operations on the database underpinning the CoRIM store.",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the migrations meta-tables.",

	Run: func(cmd *cobra.Command, args []string) {
		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		ctx := context.Background()
		migrator := migrate.NewMigrator(db, migrations.Migrations)
		CheckErr(migrator.Init(ctx))

		CheckErr(migrator.Lock(ctx))
		defer func() { CheckErr(migrator.Unlock(ctx)) }()

		_, err = migrator.Migrate(ctx)
		CheckErr(err)
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database schema to the latest version.",

	Run: func(cmd *cobra.Command, args []string) {
		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		migrator := migrate.NewMigrator(db, migrations.Migrations)
		ctx := context.Background()

		CheckErr(migrator.Lock(ctx))
		defer CheckErr(migrator.Unlock(ctx))

		group, err := migrator.Migrate(ctx)
		CheckErr(err)

		if group.IsZero() {
			fmt.Println("Everything is up-to-date.")
		} else {
			fmt.Println("Migrated to", group)
		}
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Roll back the last migration group.",

	Run: func(cmd *cobra.Command, args []string) {
		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		migrator := migrate.NewMigrator(db, migrations.Migrations)
		ctx := context.Background()

		CheckErr(migrator.Lock(ctx))
		defer func() { CheckErr(migrator.Unlock(ctx)) }()

		group, err := migrator.Rollback(ctx)
		CheckErr(err)

		if group.IsZero() {
			fmt.Println("Nothing to roll back.")
		} else {
			fmt.Println("Rolled back", group)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the currently configured database.",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("                DBMS: %s\n", cliConfig.DBMS)
		fmt.Printf("                 DSN: %s\n", cliConfig.DSN)

		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		migrator := migrate.NewMigrator(db, migrations.Migrations)
		ctx := context.Background()

		status, err := migrator.MigrationsWithStatus(ctx)
		if err != nil && strings.Contains(err.Error(), "no such table") {
			err = errors.New("database has not been initialized (run init sub-command)")
		}
		CheckErr(err)

		fmt.Printf("          migrations: %s\n", status)
		fmt.Printf("unapplied migrations: %s\n", status.Unapplied())
		fmt.Printf("last migration group: %s\n", status.LastGroup())

		if len(status.Unapplied()) == 0 {
			fmt.Println(Green("ok"))
		} else {
			fmt.Println(Amber("migration required"))
		}
	},
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Display database schema as SQL.",

	Run: func(cmd *cobra.Command, args []string) {
		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		ctx := context.Background()

		switch cliConfig.DBMS {
		case "sqlite", "sqlite3":
			var sql []string

			err = db.NewSelect().
				TableExpr("sqlite_master").
				Column("sql").
				Where("type IN (?)", bun.In([]string{"table", "view"})).
				Scan(ctx, &sql)

			CheckErr(err)
			for _, statement := range sql {
				fmt.Printf("%s;\n", statement)
			}
		default:
			CheckErr(fmt.Errorf("unsupported DBMS: %s", cliConfig.DBMS))
		}

	},
}

var fixturesCmd = &cobra.Command{
	Use:   "fixtures",
	Short: "Commands related to fixtures (see https://bun.uptrace.dev/guide/fixtures.html).",
	Args:  cobra.NoArgs,
}

var loadFixturesCmd = &cobra.Command{
	Use:   "load",
	Short: "Load fixtures from YAML files.",
	Args:  cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		truncate, err := cmd.Flags().GetBool("truncate")
		CheckErr(err)

		var opts []dbfixture.FixtureOption
		if truncate {
			opts = append(opts, dbfixture.WithTruncateTables())
		}

		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		model.RegisterModels(db)

		fixture := dbfixture.New(db, opts...)
		ctx := context.Background()

		for _, path := range args {
			fmt.Printf("Loading from %s...\n", path)
			dir, file := filepath.Split(path)
			err = fixture.Load(ctx, os.DirFS(dir), file)
			CheckErr(err)
		}

		fmt.Println(Green("ok"))
	},
}

var saveFixturesCmd = &cobra.Command{
	Use:   "save OUTFILE",
	Short: "Save contents of tables as YAML fixtures.",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Amber("WARNING") + ": due to issues with the bun fixture encoder, generated fixtures may need to be manually fixed up before they can be used.")
		db, err := db.Open(cliConfig.DB())
		CheckErr(err)
		defer func() { CheckErr(db.Close()) }()

		ctx := context.Background()
		rowSets, err := selectModels(cmd.Flags(), db, ctx)
		CheckErr(err)

		var buf bytes.Buffer
		encoder := dbfixture.NewEncoder(db, &buf)
		err = encoder.Encode(rowSets...)
		CheckErr(err)

		fixedBuf, err := fixUpEncoderOutput(buf.String())
		CheckErr(err)

		file, err := os.Create(args[0])
		CheckErr(err)
		defer func() { CheckErr(file.Close()) }()

		_, err = file.WriteString(fixedBuf)
		CheckErr(err)

		fmt.Println(Green("ok"))
	},
}

var tableSelectors = []string{
	"crypto-keys", "digests", "entities", "environments", "extensions", "integrity-registers",
	"key-triples", "linked-tags", "locators", "manifests", "measurements", "measurement-values",
	"module-tags", "roles", "value-triples",
}

func selectModels(pflags *pflag.FlagSet, db bun.IDB, ctx context.Context) ([]any, error) {
	all, err := pflags.GetBool("all")
	if err != nil {
		return nil, err
	}

	flags := make(map[string]bool)
	for _, flagName := range tableSelectors {
		flags[flagName], err = pflags.GetBool(flagName)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", flagName, err)
		}

		inverseName := "no-" + flagName
		flags[inverseName], err = pflags.GetBool(inverseName)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", inverseName, err)
		}

		if flags[flagName] && flags[inverseName] {
			return nil, fmt.Errorf("can't specify both --%s and --%s", flagName, inverseName)
		}
	}

	if !all {
		atLeastOneTrue := false
		for _, flag := range flags {
			if flag {
				atLeastOneTrue = true
				break
			}
		}

		if !atLeastOneTrue {
			return nil, fmt.Errorf("you must specify at least one flag selecting a table (see --help)")
		}
	}

	var ret []any

	if all && !flags["no-crypto-keys"] || flags["crypto-keys"] {
		selected, err := selectModel[model.CryptoKey](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting crypto keys: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-digests"] || flags["digests"] {
		selected, err := selectModel[model.Digest](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting digests: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-entities"] || flags["entities"] {
		selected, err := selectModel[model.Entity](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting entities: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-environments"] || flags["environments"] {
		selected, err := selectModel[model.Environment](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting environments: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-extensions"] || flags["extensions"] {
		selected, err := selectModel[model.ExtensionValue](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting extensions: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-flags"] || flags["flags"] {
		selected, err := selectModel[model.Flag](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting flags: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-integrity-registers"] || flags["integrity-registers"] {
		selected, err := selectModel[model.IntegrityRegister](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting integrity registers: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["key-triples"] || flags["key-triples"] {
		selected, err := selectModel[model.KeyTriple](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting key triples: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["linked-tags"] || flags["linked-tags"] {
		selected, err := selectModel[model.LinkedTag](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting linked tags: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-locators"] || flags["locators"] {
		selected, err := selectModel[model.Locator](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting locators: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-manifests"] || flags["manifests"] {
		selected, err := selectModel[model.Manifest](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting manifests: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-measurements"] || flags["measurements"] {
		selected, err := selectModel[model.Measurement](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting measurements: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-measurement-values"] || flags["measurement-values"] {
		selected, err := selectModel[model.MeasurementValueEntry](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting measurement values: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-module-tags"] || flags["module-tags"] {
		selected, err := selectModel[model.ModuleTag](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting module tags: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["no-roles"] || flags["roles"] {
		selected, err := selectModel[model.RoleEntry](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting roles: %w", err)
		}

		ret = append(ret, selected)
	}

	if all && !flags["value-triples"] || flags["value-triples"] {
		selected, err := selectModel[model.ValueTriple](db, ctx)
		if err != nil {
			return nil, fmt.Errorf("error selecting value triples: %w", err)
		}

		ret = append(ret, selected)
	}

	return ret, nil
}

func selectModel[T any](db bun.IDB, ctx context.Context) ([]T, error) {
	var ret []T

	if err := db.NewSelect().Model(&ret).OrderExpr("id").Scan(ctx); err != nil {
		return nil, err
	}

	return ret, nil
}

// The Encoder in the dbfixture package is _very_ broken. It basically just
// serializes the passed struct to YAML, which causes a number of issues:
//   - It serializes the embedded BaseModel
//   - It serializes field names (as all lower case) rather than doing translation
//     to column names.
//   - It serializes slices of other models that have bun's has-many relationship, and
//     so do not correspond to columns
//
// This attempts to fix some of the issues by doing regex replacements on the output.
// TODO: This is CLUDGE, however an actual fix would require a significant re-write of the
// dbfixture encoder.
func fixUpEncoderOutput(text string) (string, error) {
	type regexReplace struct {
		re  *regexp.Regexp
		rep string
	}

	regExps := []regexReplace{
		{regexp.MustCompile(`- basemodel: \{\}\n      id:`), `- id:`},
		{regexp.MustCompile(`      \w+: null\n`), ``},
		{regexp.MustCompile(`      \w+: \[\]\n`), ``},
		{regexp.MustCompile(`(?P<name>[a-z]+)bytes:`), `${name}_bytes:`},
		{regexp.MustCompile(`(?P<name>[a-z]+)idtype:`), `${name}_id_type:`},
		{regexp.MustCompile(`(?P<name>[a-z]+)type:`), `${name}_type:`},
		{regexp.MustCompile(`(?P<name>[a-z]+)id:`), `${name}_id:`},
		{regexp.MustCompile(`timeadded`), `time_added`},
		{regexp.MustCompile(`notbefore`), `not_before`},
		{regexp.MustCompile(`notafter`), `not_after`},
		{regexp.MustCompile(`indexuint`), `index_uint`},
		{regexp.MustCompile(`indextext`), `index_text`},
		{regexp.MustCompile(`codepoint`), `code_point`},
		{regexp.MustCompile(`tagversion`), `tag_version`},
	}

	for _, r := range regExps {
		text = r.re.ReplaceAllString(text, r.rep)
	}

	return text, nil
}

func init() {
	loadFixturesCmd.Flags().Bool("truncate", false, "Truncate existing data before loading the fixtures.")

	saveFixturesCmd.Flags().Bool("all", false, "Select all tables.")
	for _, flagName := range tableSelectors {
		saveFixturesCmd.Flags().Bool(flagName, false, "Select the corresponding table.")

		inverseName := "no-" + flagName
		saveFixturesCmd.Flags().Bool(inverseName, false,
			"Deselect the corresponding table (when used with --all).")
	}

	fixturesCmd.AddCommand(saveFixturesCmd)
	fixturesCmd.AddCommand(loadFixturesCmd)

	dbCmd.AddCommand(initCmd)
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(rollbackCmd)
	dbCmd.AddCommand(statusCmd)
	dbCmd.AddCommand(schemaCmd)
	dbCmd.AddCommand(fixturesCmd)

	rootCmd.AddCommand(dbCmd)
}
