package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim-store/pkg/util"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries of a particular type.",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runListCommand(cmd, args))
	},
}

func runListCommand(cmd *cobra.Command, args []string) error {
	var err error
	what := util.Normalize(args[0])

	store, err := store.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	var header []any
	var rows [][]any

	switch what {
	case "manifests", "corims":
		header, rows, err = listManifests(store)
	case "modules", "module_tags", "comids":
		header, rows, err = listModuleTags(store)
	case "entities":
		header, rows, err = listEntities(store)
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
			retRow = append(retRow, match[col.(string)])
		}

		ret = append(ret, retRow)
	}

	return retCols, ret, nil
}

func listModuleTags(store *store.Store) ([]any, [][]any, error) {
	columns := []string{"tag_id", "language", "manifest", "label"}
	retCols := make([]any, 0, len(columns)+1)

	var matches []map[string]any
	err := store.DB.NewSelect().TableExpr("module_tags AS mod").
		ColumnExpr("tag_id").
		ColumnExpr("language").
		ColumnExpr(fmt.Sprintf("%s AS entities", store.StringAggregatorExpr("ent.name"))).
		ColumnExpr("man.manifest_id as manifest").
		ColumnExpr("man.label as label").
		Join("LEFT JOIN entities as ent ON ent.owner_id = mod.id AND ent.owner_type = 'module_tag'").
		Join("LEFT JOIN manifests as man ON man.id = mod.manifest_id").
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
			retRow = append(retRow, match[col.(string)])
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
			retRow = append(retRow, match[col])
		}
		retRow = append(retRow, match["roles"])

		ret = append(ret, retRow)
	}

	return retCols, ret, nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}
