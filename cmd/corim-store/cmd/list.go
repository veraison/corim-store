package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim-store/pkg/util"
)

var listCmd = &cobra.Command{
	Use:   "list WHAT",
	Short: "List entries of a particular type.",
	Long: `List all entries of a particular type in the store.

The WHAT can be "manifests"/"corims", "modules"/"module_tags"/"comids",
"entities", or "triples" (slashes indicate alternate names for the same type
of entry). When the  WHAT is \"triples\", flags can be used to filter the
results by environment elements (e.g. by model or instance ID)."`,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runListCommand(cmd, args))
	},
}

func runListCommand(cmd *cobra.Command, args []string) error {
	var err error
	what := util.Normalize(args[0])

	env, err := BuildEnvironment(cmd)
	CheckErr(err)

	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	exact, err := cmd.Flags().GetBool("exact")
	if err != nil {
		return err
	}

	store, err := store.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	var header []any
	var rows [][]any

	if what != "triples" && !env.IsEmpty() {
		return errors.New("environment specifiers are only allowed for triples")
	}

	switch what {
	case "manifests", "corims":
		header, rows, err = listManifests(store)
	case "modules", "module_tags", "comids":
		header, rows, err = listModuleTags(store)
	case "entities":
		header, rows, err = listEntities(store)
	case "triples":
		header, rows, err = listTriples(store, env, label, exact)
	default:
		return fmt.Errorf("unsupported list target: %s", what)
	}

	if err != nil {
		return err
	}

	tw := table.NewWriter()
	tw.AppendHeader(table.Row(header))
	for _, row := range rows {
		tw.AppendRow(table.Row(row))
	}

	colConfigs := make([]table.ColumnConfig, 0, len(header))
	for _, h := range header {
		colConfigs = append(colConfigs, table.ColumnConfig{
			Name:        h.(string),
			AlignHeader: text.AlignCenter,
			VAlign:      text.VAlignMiddle,
		})
	}
	tw.SetColumnConfigs(colConfigs)
	tw.SetStyle(table.StyleLight)

	fmt.Println(tw.Render())

	return nil
}

func listManifests(store *store.Store) ([]any, [][]any, error) {
	columns := []string{"label", "manifest_id", "profile",
		"not_before", "not_after", "digest", "time_added"}
	retCols := make([]any, 0, len(columns)+1)

	var matches []map[string]any
	err := store.DB.NewSelect().TableExpr("manifests AS man").
		ColumnExpr("label").
		ColumnExpr("manifest_id").
		ColumnExpr("profile").
		ColumnExpr(fmt.Sprintf("%s AS entities", store.StringAggregatorExpr("ent.name"))).
		ColumnExpr("not_before").
		ColumnExpr("not_after").
		ColumnExpr(fmt.Sprintf("%s AS digest", store.HexExpr("digest"))).
		ColumnExpr("time_added").
		Join("LEFT JOIN entities as ent ON ent.owner_id = man.id AND ent.owner_type = 'manifest'").
		GroupExpr(strings.Join(columns, ", ")).
		Scan(store.Ctx, &matches)

	if err != nil {
		return nil, nil, err
	}

	for _, col := range columns {
		retCols = append(retCols, col)
	}
	retCols = append(retCols[:3], append([]any{"entities"}, retCols[3:]...)...)

	ret := make([][]any, 0, len(matches))
	for _, match := range matches {
		retRow := make([]any, 0, len(retCols)+1)
		for _, col := range retCols {
			retRow = append(retRow, unwrapNullableTypes(match[col.(string)]))
		}

		ret = append(ret, retRow)
	}

	return retCols, ret, nil
}

func listModuleTags(store *store.Store) ([]any, [][]any, error) {
	columns := []string{"tag_id", "language", "manifest", "label"}
	retCols := make([]any, 0, len(columns)+1)

	var matches []map[string]any
	err := store.DB.NewSelect().TableExpr("module_tags AS mt").
		ColumnExpr("tag_id").
		ColumnExpr("language").
		ColumnExpr(fmt.Sprintf("%s AS entities", store.StringAggregatorExpr("ent.name"))).
		ColumnExpr("man.manifest_id as manifest").
		ColumnExpr("man.label as label").
		Join("LEFT JOIN entities as ent ON ent.owner_id = mt.id AND ent.owner_type = 'module_tag'").
		Join("LEFT JOIN manifests as man ON man.id = mt.manifest_id").
		GroupExpr(strings.Join(columns, ", ")).
		Scan(store.Ctx, &matches)

	if err != nil {
		return nil, nil, err
	}

	for _, col := range columns {
		retCols = append(retCols, col)
	}
	retCols = append(retCols[:2], append([]any{"entities"}, retCols[2:]...)...)

	ret := make([][]any, 0, len(matches))
	for _, match := range matches {
		retRow := make([]any, 0, len(retCols)+1)
		for _, col := range retCols {
			retRow = append(retRow, unwrapNullableTypes(match[col.(string)]))
		}

		ret = append(ret, retRow)
	}

	return retCols, ret, nil
}

func listEntities(store *store.Store) ([]any, [][]any, error) {
	var matches []map[string]any
	columns := []string{"name", "uri", "owner"}
	retCols := make([]any, 0, len(columns)+1)
	ownerExpr := store.ConcatExpr("owner_type", "'('", "owner_id", "')'")

	err := store.DB.NewSelect().TableExpr("entities AS ent").
		ColumnExpr("name").
		ColumnExpr("uri").
		ColumnExpr(fmt.Sprintf("%s AS owner", ownerExpr)).
		ColumnExpr(fmt.Sprintf("%s AS roles", store.StringAggregatorExpr("r.role"))).
		Join("LEFT JOIN roles as r ON r.entity_id = ent.id").
		GroupExpr(strings.Join(columns, ", ")).
		Scan(store.Ctx, &matches)

	if err != nil {
		return nil, nil, err
	}

	for _, col := range columns {
		retCols = append(retCols, col)
	}
	retCols = append(retCols, "roles")

	ret := make([][]any, 0, len(matches))
	for _, match := range matches {
		retRow := make([]any, 0, len(retCols)+1)
		for _, col := range columns {
			retRow = append(retRow, unwrapNullableTypes(match[col]))
		}
		retRow = append(retRow, unwrapNullableTypes(match["roles"]))

		ret = append(ret, retRow)
	}

	return retCols, ret, nil
}

func listTriples(
	store *store.Store,
	env *model.Environment,
	label string,
	exact bool,
) ([]any, [][]any, error) {
	var matches []map[string]any
	err := store.DB.NewSelect().TableExpr("module_tags AS mt").
		ColumnExpr("mt.id as id").
		ColumnExpr("tag_id as module").
		ColumnExpr("man.manifest_id as manifest").
		ColumnExpr("man.label as label").
		Join("LEFT JOIN manifests as man ON man.id = mt.manifest_id").
		Scan(store.Ctx, &matches)

	if err != nil {
		return nil, nil, err
	}

	lookup := make(map[int64]map[string]any)
	for _, match := range matches {
		lookup[match["id"].(int64)] = match
	}

	keyTriples, err := store.GetKeyTriples(env, label, exact)
	if err != nil {
		return nil, nil, fmt.Errorf("getting key triples: %w", err)
	}

	valueTriples, err := store.GetValueTriples(env, label, exact)
	if err != nil {
		return nil, nil, fmt.Errorf("getting value triples: %w", err)
	}

	columns := []any{"id", "active", "label", "manifest", "module", "type", "environment"}
	rows := make([][]any, 0, len(valueTriples)+len(keyTriples))

	for _, kt := range keyTriples {
		module, ok := lookup[kt.ModuleID]
		if !ok {
			return nil, nil, fmt.Errorf("orphan key triple: %d", kt.ID)
		}

		envText, err := RenderEnviroment(kt.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("environment for key triple %d: %w", kt.ID, err)
		}

		rows = append(rows, []any{
			kt.ID,
			kt.IsActive,
			unwrapNullableTypes(module["label"]),
			unwrapNullableTypes(module["manifest"]),
			unwrapNullableTypes(module["module"]),
			fmt.Sprintf("%s key", kt.Type),
			envText,
		})
	}

	for _, vt := range valueTriples {
		module, ok := lookup[vt.ModuleID]
		if !ok {
			return nil, nil, fmt.Errorf("orphan value triple: %d", vt.ID)
		}

		envText, err := RenderEnviroment(vt.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("environment for value triple %d: %w", vt.ID, err)
		}

		rows = append(rows, []any{
			vt.ID,
			vt.IsActive,
			unwrapNullableTypes(module["label"]),
			unwrapNullableTypes(module["manifest"]),
			unwrapNullableTypes(module["module"]),
			fmt.Sprintf("%s value", vt.Type),
			envText,
		})
	}

	return columns, rows, nil
}

func unwrapNullableTypes(val any) any {
	switch t := val.(type) {
	case sql.NullString:
		if t.Valid {
			return t.String
		} else {
			return nil
		}
	case sql.NullInt64:
		if t.Valid {
			return t.Int64
		} else {
			return nil
		}
	case sql.NullInt32:
		if t.Valid {
			return t.Int32
		} else {
			return nil
		}
	case sql.NullInt16:
		if t.Valid {
			return t.Int16
		} else {
			return nil
		}
	case sql.NullByte:
		if t.Valid {
			return t.Byte
		} else {
			return nil
		}
	case sql.NullBool:
		if t.Valid {
			return t.Bool
		} else {
			return nil
		}
	default:
		return t
	}
}

func init() {
	AddEnviromentFlags(listCmd)
	listCmd.Flags().StringP("label", "l", "",
		"Label that will be applied to the manifest in the store.")

	listCmd.Flags().BoolP("exact", "e", false,
		"Match environments exactly, including null fields. The default is to assume that "+
			"null fields (i.e. fields not explicitly specified) can match any value.")

	rootCmd.AddCommand(listCmd)
}
