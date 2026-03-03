package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/client9/nowandlater"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/veraison/corim-store/pkg/model"
	storemod "github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim/comid"
)

func CheckErr(msg interface{}) {
	if msg != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", Red("ERROR"), msg)
		os.Exit(1)
	}
}

func Red(msg string) string {
	if cliConfig != nil && cliConfig.NoColor {
		return msg
	}

	return fmt.Sprintf("\033[1;31m%s\033[0m", msg)
}

func Amber(msg string) string {
	if cliConfig != nil && cliConfig.NoColor {
		return msg
	}

	return fmt.Sprintf("\033[1;33m%s\033[0m", msg)
}

func Green(msg string) string {
	if cliConfig != nil && cliConfig.NoColor {
		return msg
	}

	return fmt.Sprintf("\033[1;32m%s\033[0m", msg)
}

const flagsHelp = `

When specifying IDs (flags that end with -id), the value can optionally be
prefixed with a type followed by ":". For --comid-id/--module-tag-id and
--corim-id/--manifest-id, the supported types are "string" and "uuid".

When types are specified for an ID, both the type and the value have to match.
For --class-id, --instance-id, and --group-id, the type also determines how the
value text is decoded into bytes: 

 - for "uuid", the value has to be 32 hexadecimal digits separated by hyphens
   into groups of 8, 4, 4, 4, and 12
 - for "oid" the value has to be formed of decimal digits separated by dots
 - for "hex" the value has to be a sequence of an even number of hexadecimal
   digits (without the leading "0x")
 - for all other types, the value is interpreted as base64 encoding of the bytes

You can use the type to only control the decoding of the value and not be part
of the match by suffixing it with "*" (just before the ":").

For example, 

"--class-id 3q2+7w==" will attempt to match the class ID value to 0xDEADBEEF.
"--class-id foo:3q2+7w==" will attempt to match the class ID type to "foo" and
value to 0xDEADBEEF.
"--class-id hex*:deadbeef" will attempt to match the class ID value to 0xDEADBEEF
(but will NOT match type).`

const timeHelp = `

--added-before, --added-after, and --valid-on expect a time/date value; --added
expects a time period. These arguments are parsed using nowandlater library that
can handle most commonly used formats as well as natural language expressions such
as "today" or "2 days ago". For more examples of supported formats please see

    https://github.com/client9/nowandlater#supported-formats
`

func AddQueryFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("class-id", "C", "", "Environment class ID.")
	cmd.Flags().StringP("vendor", "V", "", "Environment vendor.")
	cmd.Flags().StringP("model", "M", "", "Environment model.")
	cmd.Flags().Int64P("layer", "L", -1, "Environment layer.")
	cmd.Flags().Int64P("index", "I", -1, "Environment index.")
	cmd.Flags().StringP("instance-id", "i", "", "Environment instance ID")
	cmd.Flags().StringP("group-id", "g", "", "Environment group ID")

	cmd.Flags().Int64("id", 0, "Database ID of the item.")

	cmd.Flags().StringP("label", "l", "", "Manifest (CoRIM) store label.")
	cmd.Flags().String("manifest-id", "", "Manifest (CoRIM) ID. (DO NOT use with --corim-id)")
	cmd.Flags().String("corim-id", "", "Manifest (CoRIM) ID. (DO NOT use with --manifest-id)")
	cmd.Flags().String("profile", "", "Manifest (CoRIM) profile.")
	cmd.Flags().Bool("valid", false, "today is within the validity period")
	cmd.Flags().String("valid-on", "", "specified day is within the validity period")

	cmd.Flags().String("added", "", "added within the specified period")
	cmd.Flags().String("added-before", "", "added before the specified time")
	cmd.Flags().String("added-after", "", "added after the specified time")

	cmd.Flags().String("module-tag-id", "", "Module tag (CoMID) ID. (DO NOT use with --comid-id)")
	cmd.Flags().String("comid-id", "", "Module tag (CoMID) ID. (DO NOT use with --module-tag-id)")
	cmd.Flags().Uint("version", 0, "Module tag (CoMID) version.")
	cmd.Flags().String("language", "", "Language set in the module tag (CoMID).")

	cmd.Flags().BoolP("exact", "e", false,
		"Match environments exactly, including null fields. The default is to assume that "+
			"null fields (i.e. fields not explicitly specified) can match any value.")

}

func BuildManifestQuery(flags *pflag.FlagSet) (*storemod.ManifestQuery, error) {
	query := storemod.NewManifestQuery()

	if err := updateManifestCommonQueryFromFlags(&query.ManifestCommonQuery, flags); err != nil {
		return nil, err
	}

	id, err := flags.GetInt64("id")
	if err != nil {
		panic(err)
	}
	if id != 0 {
		query.ManifestDbID(id)
	}

	timeParser := nowandlater.Parser{}

	addedText, err := flags.GetString("added")
	if err != nil {
		panic(err)
	}
	if addedText != "" {
		start, end, err := timeParser.ParseInterval(addedText)
		if err != nil {
			return nil, fmt.Errorf("added: %w", err)
		}

		query.AddedBetween(start, end)
	}

	addedBeforeText, err := flags.GetString("added-before")
	if err != nil {
		panic(err)
	}
	if addedBeforeText != "" {
		t, err := timeParser.Parse(addedBeforeText)
		if err != nil {
			return nil, fmt.Errorf("added-before: %w", err)
		}

		query.AddedBefore(t)
	}

	addedAfterText, err := flags.GetString("added-after")
	if err != nil {
		panic(err)
	}
	if addedAfterText != "" {
		t, err := timeParser.Parse(addedAfterText)
		if err != nil {
			return nil, fmt.Errorf("added-after: %w", err)
		}

		query.AddedAfter(t)
	}

	return query, nil
}

func BuildModuleTagQuery(flags *pflag.FlagSet) (*storemod.ModuleTagQuery, error) {
	query := storemod.NewModuleTagQuery()

	if err := updateManifestCommonQueryFromFlags(&query.ManifestCommonQuery, flags); err != nil {
		return nil, err
	}

	if err := updateModuleTagCommonQueryFromFlags(&query.ModuleTagCommonQuery, flags); err != nil {
		return nil, err
	}

	id, err := flags.GetInt64("id")
	if err != nil {
		panic(err)
	}
	if id != 0 {
		query.ModuleTagDbID(id)
	}

	return query, nil
}

func BuildEntityQuery(flags *pflag.FlagSet) (*storemod.EntityQuery, error) {
	query := storemod.NewEntityQuery()

	id, err := flags.GetInt64("id")
	if err != nil {
		panic(err)
	}
	if id != 0 {
		query.ID(id)
	}

	return query, nil
}

func BuildValueTripleQuery(flags *pflag.FlagSet) (*storemod.ValueTripleQuery, error) {
	query := storemod.NewValueTripleQuery()

	if err := updateManifestCommonQueryFromFlags(&query.ManifestCommonQuery, flags); err != nil {
		return nil, err
	}

	if err := updateModuleTagCommonQueryFromFlags(&query.ModuleTagCommonQuery, flags); err != nil {
		return nil, err
	}

	if err := updateEnvironmentQueryFromFlags(query.EnvironmentSubquery(), flags); err != nil {
		return nil, err
	}

	id, err := flags.GetInt64("id")
	if err != nil {
		panic(err)
	}
	if id != 0 {
		query.TripleDbID(id)
	}

	return query, nil
}

func BuildKeyTripleQuery(flags *pflag.FlagSet) (*storemod.KeyTripleQuery, error) {
	query := storemod.NewKeyTripleQuery()

	if err := updateManifestCommonQueryFromFlags(&query.ManifestCommonQuery, flags); err != nil {
		return nil, err
	}

	if err := updateModuleTagCommonQueryFromFlags(&query.ModuleTagCommonQuery, flags); err != nil {
		return nil, err
	}

	if err := updateEnvironmentQueryFromFlags(query.EnvironmentSubquery(), flags); err != nil {
		return nil, err
	}

	id, err := flags.GetInt64("id")
	if err != nil {
		panic(err)
	}
	if id != 0 {
		query.TripleDbID(id)
	}

	return query, nil
}

func updateManifestCommonQueryFromFlags(query *storemod.ManifestCommonQuery, flags *pflag.FlagSet) error {
	label, err := flags.GetString("label")
	if err != nil {
		panic(err)
	}
	if label != "" {
		query.Label(label)
	}

	manifestIDText, err := flags.GetString("manifest-id")
	if err != nil {
		panic(err)
	}
	altManifestIDText, err := flags.GetString("corim-id")
	if err != nil {
		panic(err)
	}
	if altManifestIDText != "" {
		if manifestIDText != "" {
			return fmt.Errorf("only one of --corim-id or --manifest-id should be used")
		}
		manifestIDText = altManifestIDText
	}
	if manifestIDText != "" {
		parts := strings.SplitN(manifestIDText, ":", 2)
		if len(parts) == 2 {
			useType := true
			if strings.HasSuffix(parts[0], "*") {
				useType = false
				parts[0] = strings.TrimRight(parts[0], "*")
			}

			switch parts[0] {
			case string(model.StringTagID):
				if useType {
					query.ManifestID(model.StringTagID, parts[1])
				} else {
					query.ManifestIDValue(parts[1])
				}
			case string(model.UUIDTagID):
				if useType {
					query.ManifestID(model.UUIDTagID, parts[1])
				} else {
					query.ManifestIDValue(parts[1])
				}
			default:
				return fmt.Errorf("invalid manifest ID type: %s", parts[0])
			}
		} else {
			query.ManifestIDValue(parts[0])
		}
	}

	profileText, err := flags.GetString("profile")
	if err != nil {
		panic(err)
	}
	if profileText != "" {
		parts := strings.SplitN(manifestIDText, ":", 2)
		if len(parts) == 2 {
			useType := true
			if strings.HasSuffix(parts[0], "*") {
				useType = false
				parts[0] = strings.TrimRight(parts[0], "*")
			}

			switch parts[0] {
			case string(model.OIDProfile):
				if useType {
					query.Profile(model.OIDProfile, parts[1])
				} else {
					query.ProfileValue(parts[1])
				}
			case string(model.URIProfile):
				if useType {
					query.Profile(model.URIProfile, parts[1])
				} else {
					query.ProfileValue(parts[1])
				}
			default:
				return fmt.Errorf("invalid profile type: %s", parts[0])
			}
		} else {
			query.ProfileValue(parts[0])
		}
	}

	timeParser := nowandlater.Parser{}

	validOnText, err := flags.GetString("valid-on")
	if err != nil {
		panic(err)
	}
	if validOnText != "" {
		t, err := timeParser.Parse(validOnText)
		if err != nil {
			return fmt.Errorf("valid-on: %w", err)
		}

		query.ValidOn(t)
	}

	valid, err := flags.GetBool("valid")
	if err != nil {
		panic(err)
	}
	if valid {
		query.ValidOn(time.Now())
	}

	return nil
}

func updateModuleTagCommonQueryFromFlags(query *storemod.ModuleTagCommonQuery, flags *pflag.FlagSet) error {
	moduleTagIDText, err := flags.GetString("module-tag-id")
	if err != nil {
		panic(err)
	}
	altModuleTagIDText, err := flags.GetString("comid-id")
	if err != nil {
		panic(err)
	}
	if altModuleTagIDText != "" {
		if moduleTagIDText != "" {
			return fmt.Errorf("only one of --comid-id or --module-tag-id should be used")
		}
		moduleTagIDText = altModuleTagIDText
	}
	if moduleTagIDText != "" {
		parts := strings.SplitN(moduleTagIDText, ":", 2)
		if len(parts) == 2 {
			useType := true
			if strings.HasSuffix(parts[0], "*") {
				useType = false
				parts[0] = strings.TrimRight(parts[0], "*")
			}

			switch parts[0] {
			case string(model.StringTagID):
				if useType {
					query.ModuleTagID(model.StringTagID, parts[1])
				} else {
					query.ModuleTagIDValue(parts[1])
				}
			case string(model.UUIDTagID):
				if useType {
					query.ModuleTagID(model.UUIDTagID, parts[1])
				} else {
					query.ModuleTagIDValue(parts[1])
				}
			default:
				return fmt.Errorf("invalid module tag ID type: %s", parts[0])
			}
		} else {
			query.ModuleTagIDValue(parts[0])
		}
	}

	language, err := flags.GetString("language")
	if err != nil {
		panic(err)
	}
	if language != "" {
		query.Language(language)
	}

	version, err := flags.GetUint("version")
	if err != nil {
		panic(err)
	}
	if version != 0 {
		query.ModuleTagVersion(version)
	}

	return nil
}

func updateEnvironmentQueryFromFlags(query *storemod.EnvironmentQuery, flags *pflag.FlagSet) error {
	exact, err := flags.GetBool("exact")
	if err != nil {
		panic(err)
	}
	if exact {
		query.Exact = true
	}

	vendor, err := flags.GetString("vendor")
	if err != nil {
		panic(err)
	}
	if vendor != "" {
		query.Vendor(vendor)
	}

	model, err := flags.GetString("model")
	if err != nil {
		panic(err)
	}
	if model != "" {
		query.Model(model)
	}

	layerInt, err := flags.GetInt64("layer")
	if err != nil {
		panic(err)
	}
	if layerInt > -1 {
		query.Layer(uint64(layerInt))
	}

	indexInt, err := flags.GetInt64("index")
	if err != nil {
		panic(err)
	}
	if indexInt > -1 {
		query.Index(uint64(indexInt))
	}

	classIDText, err := flags.GetString("class-id")
	if err != nil {
		panic(err)
	}

	if classIDText != "" {
		classIDBytes, classIDType, useType, err := parseEnvironmentID(classIDText)
		if err != nil {
			return fmt.Errorf("class-id: %w", err)
		}

		if useType {
			query.ClassID(classIDType, classIDBytes)
		} else {
			query.ClassIDBytes(classIDBytes)
		}
	}

	instanceIDText, err := flags.GetString("instance-id")
	if err != nil {
		panic(err)
	}

	if instanceIDText != "" {
		instanceIDBytes, instanceIDType, useType, err := parseEnvironmentID(instanceIDText)
		if err != nil {
			return fmt.Errorf("instance-id: %w", err)
		}

		if useType {
			query.Instance(instanceIDType, instanceIDBytes)
		} else {
			query.InstanceBytes(instanceIDBytes)
		}
	}

	groupIDText, err := flags.GetString("group-id")
	if err != nil {
		panic(err)
	}

	if groupIDText != "" {
		groupIDBytes, groupIDType, useType, err := parseEnvironmentID(groupIDText)
		if err != nil {
			return fmt.Errorf("group-id: %w", err)
		}

		if useType {
			query.Group(groupIDType, groupIDBytes)
		} else {
			query.GroupBytes(groupIDBytes)
		}
	}

	return nil
}

func parseEnvironmentID(text string) ([]byte, string, bool, error) {
	var typeText string
	var valueText string
	var useType bool

	parts := strings.SplitN(text, ":", 2)
	if len(parts) == 2 {
		useType = true
		if strings.HasSuffix(parts[0], "*") {
			parts[0] = strings.TrimRight(parts[0], "*")
			useType = false
		}

		typeText = parts[0]
		valueText = parts[1]
	} else {
		useType = false
		valueText = text
	}

	switch typeText {
	case "uuid":
		ret, err := uuid.Parse(valueText)
		return ret[:], "uuid", useType, err
	case "oid":
		var ret comid.OID
		if err := ret.FromString(valueText); err != nil {
			return nil, "oid", false, err
		}
		return []byte(ret), "oid", useType, nil
	case "hex":
		ret, err := hex.DecodeString(valueText)
		return ret, "bytes", useType, err
	default: // assume base64
		// remove padding
		valueText = strings.Trim(valueText, "=")
		// if URL, convert to standard
		valueText = strings.ReplaceAll(valueText, "-", "+")
		valueText = strings.ReplaceAll(valueText, "_", "/")

		ret, err := base64.RawStdEncoding.DecodeString(valueText)
		return ret, typeText, useType, err
	}
}
