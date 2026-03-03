package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/veraison/corim-store/pkg/model"
	storemod "github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim-store/pkg/util"
)

var listCmd = &cobra.Command{
	Use:   "list WHAT",
	Short: "List entries of a particular type.",
	Long: `List all entries of a particular type in the store.

The WHAT can be "manifests"/"corims", "modules"/"module_tags"/"comids",
"entities", or "triples" (slashes indicate alternate names for the same type
of entry). When the  WHAT is \"triples\", flags can be used to filter the
results by environment elements (e.g. by model or instance ID)."` + flagsHelp + timeHelp,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runListCommand(cmd, args))
	},
}

func runListCommand(cmd *cobra.Command, args []string) error {
	var err error
	what := util.Normalize(args[0])

	store, err := storemod.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	var header []any
	var rows [][]any

	switch what {
	case "manifests", "corims":
		header, rows, err = listManifests(store, cmd.Flags())
	case "modules", "module_tags", "comids":
		header, rows, err = listModuleTags(store, cmd.Flags())
	case "entities":
		header, rows, err = listEntities(store, cmd.Flags())
	case "triples":
		header, rows, err = listTriples(store, cmd.Flags())
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

func listManifests(store *storemod.Store, flags *pflag.FlagSet) ([]any, [][]any, error) {
	query, err := BuildManifestQuery(flags)
	if err != nil {
		return nil, nil, err
	}

	manifests, err := store.QueryManifestModels(query)
	if err != nil {
		return nil, nil, err
	}

	columns := []any{"label", "manifest_id", "profile", "entities",
		"not_before", "not_after", "digest", "time_added"}
	rows := make([][]any, 0, len(columns))
	for _, manifest := range manifests {
		entityParts := make([]string, 0, len(manifest.Entities))
		for _, entity := range manifest.Entities {
			entityParts = append(entityParts, entity.Name)
		}

		rows = append(rows, []any{
			manifest.Label,
			manifest.ManifestID,
			manifest.Profile,
			strings.Join(entityParts, "\n"),
			formatTimeColumn(manifest.NotBefore),
			formatTimeColumn(manifest.NotAfter),
			base64.StdEncoding.EncodeToString(manifest.Digest),
			formatTimeColumn(&manifest.TimeAdded),
		})
	}

	return columns, rows, nil
}

func listModuleTags(store *storemod.Store, flags *pflag.FlagSet) ([]any, [][]any, error) {
	query, err := BuildModuleTagQuery(flags)
	if err != nil {
		return nil, nil, err
	}

	entries, err := store.QueryModuleTagEntries(query)
	if err != nil {
		return nil, nil, err
	}

	columns := []any{"tag_id", "version", "language", "entities", "manifest", "label"}
	rows := make([][]any, 0, len(columns))
	for _, entry := range entries {
		moduleTag, err := entry.ToModuleTag(store.Ctx, store.DB)
		if err != nil {
			return nil, nil, err
		}

		entityParts := make([]string, 0, len(moduleTag.Entities))
		for _, entity := range moduleTag.Entities {
			entityParts = append(entityParts, entity.Name)
		}

		rows = append(rows, []any{
			entry.ModuleTagID,
			entry.ModuleTagVersion,
			formatStringPtr(entry.Language),
			strings.Join(entityParts, "\n"),
			entry.ManifestID,
			entry.Label,
		})
	}

	return columns, rows, nil
}

func listEntities(store *storemod.Store, flags *pflag.FlagSet) ([]any, [][]any, error) {
	query, err := BuildEntityQuery(flags)
	if err != nil {
		return nil, nil, err
	}

	models, err := store.QueryEntityModels(query)
	if err != nil {
		return nil, nil, err
	}

	columns := []any{"name", "uri", "owner", "roles"}
	rows := make([][]any, 0, len(columns))
	for _, model := range models {
		rows = append(rows, []any{
			model.Name,
			model.URI,
			fmt.Sprintf("%s(%d)", model.OwnerType, model.OwnerID),
			strings.Join(model.Roles(), "\n"),
		})
	}

	return columns, rows, nil
}

func listTriples(store *storemod.Store, flags *pflag.FlagSet) ([]any, [][]any, error) {
	keyQuery, err := BuildKeyTripleQuery(flags)
	if err != nil {
		return nil, nil, err
	}

	valueQuery, err := BuildValueTripleQuery(flags)
	if err != nil {
		return nil, nil, err
	}

	keysMatched := true
	keyTriples, err := store.QueryKeyTripleEntries(keyQuery)
	if err != nil {
		if errors.Is(err, storemod.ErrNoMatch) {
			keysMatched = false
		} else {
			return nil, nil, err
		}
	}

	valuesMatched := true
	valueTriples, err := store.QueryValueTripleEntries(valueQuery)
	if err != nil {
		if errors.Is(err, storemod.ErrNoMatch) {
			valuesMatched = false
		} else {
			return nil, nil, err
		}
	}

	if !keysMatched && !valuesMatched {
		return nil, nil, storemod.ErrNoMatch
	}

	columns := []any{"id", "active", "label", "source", "type", "environment"}
	rows := make([][]any, 0, len(valueTriples)+len(keyTriples))
	for _, entry := range keyTriples {
		model, err := entry.ToTriple(store.Ctx, store.DB)
		if err != nil {
			return nil, nil, fmt.Errorf("key triple %d: %w", entry.TripleDbID, err)
		}

		envText, err := formatEnvironment(model.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("environment for key triple %d: %w", entry.TripleDbID, err)
		}

		rows = append(rows, []any{
			entry.TripleDbID,
			entry.IsActive,
			entry.Label,
			fmt.Sprintf("manifest: %s\nmodule: %s", entry.ManifestID, entry.ModuleTagID),
			entry.TripleType,
			envText,
		})
	}

	for _, entry := range valueTriples {
		model, err := entry.ToTriple(store.Ctx, store.DB)
		if err != nil {
			return nil, nil, fmt.Errorf("key triple %d: %w", entry.TripleDbID, err)
		}

		envText, err := formatEnvironment(model.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("environment for key triple %d: %w", entry.TripleDbID, err)
		}

		rows = append(rows, []any{
			entry.TripleDbID,
			entry.IsActive,
			entry.Label,
			fmt.Sprintf("manifest: %s\nmodule: %s", entry.ManifestID, entry.ModuleTagID),
			entry.TripleType,
			envText,
		})
	}

	return columns, rows, nil
}

func formatTimeColumn(val *time.Time) string {
	if val == nil {
		return ""
	}

	return val.Format(time.RFC3339)
}

func formatStringPtr(val *string) string {
	if val == nil {
		return ""
	}

	return *val
}

func formatEnvironment(env *model.Environment) (string, error) {
	parts, err := env.RenderParts()
	if err != nil {
		return "", err
	}

	ret := ""
	for _, part := range parts {
		ret += fmt.Sprintf("%s: %s\n", part[0], part[1])
	}

	return ret, nil
}

func init() {
	AddQueryFlags(listCmd)
	rootCmd.AddCommand(listCmd)
}
