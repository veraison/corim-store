package cmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim/comid"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get triples matching specified environment.",
	Args:  cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runGetCommand(cmd, args))
	},
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	selector, err := buildSelector(cmd)
	if err != nil {
		return err
	}

	env, err := buildEnvironment(cmd)
	if err != nil {
		return err
	}

	if env.IsEmpty() {
		return errors.New("at least one enviroment field specifier must be provided (see --help)")
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

	var result comid.Triples

	if selector.Endorsements || selector.ReferenceValues {
		found, err := store.GetValueTriples(env, label, exact)
		if err != nil {
			return err
		}

		result.ReferenceValues, err = model.ValueTriplesToCoRIM(found, model.ReferenceValueTriple)
		if err != nil {
			return err
		}

		result.EndorsedValues, err = model.ValueTriplesToCoRIM(found, model.EndorsedValueTriple)
		if err != nil {
			return err
		}
	}

	if selector.TrustAnchors {
		found, err := store.GetKeyTriples(env, label, exact)
		if err != nil {
			return err
		}

		result.AttestVerifKeys, err = model.KeyTriplesToCoRIM(found, model.AttestKeyTriple)
		if err != nil {
			return err
		}
	}

	json, err := result.MarshalJSON()
	if err != nil {
		return err
	}

	fmt.Println(string(json))

	return nil
}

type LookupMap struct {
	ReferenceValues bool
	Endorsements    bool
	TrustAnchors    bool
}

func buildSelector(cmd *cobra.Command) (*LookupMap, error) {
	var ret LookupMap
	var err error

	ret.ReferenceValues, err = cmd.Flags().GetBool("reference-values")
	if err != nil {
		return nil, fmt.Errorf("reference values: %w", err)
	}

	ret.Endorsements, err = cmd.Flags().GetBool("endorsements")
	if err != nil {
		return nil, fmt.Errorf("endorsements: %w", err)
	}

	ret.TrustAnchors, err = cmd.Flags().GetBool("trust-anchors")
	if err != nil {
		return nil, fmt.Errorf("trust-anchors: %w", err)
	}

	// if no category is explicitly specified, look up all categories.
	if !ret.ReferenceValues && !ret.Endorsements && !ret.TrustAnchors {
		ret.ReferenceValues = true
		ret.Endorsements = true
		ret.TrustAnchors = true
	}

	return &ret, nil
}

func buildEnvironment(cmd *cobra.Command) (*model.Environment, error) {
	var ret model.Environment

	vendor, err := cmd.Flags().GetString("vendor")
	if err != nil {
		return nil, fmt.Errorf("vendor: %w", err)
	}
	if vendor != "" {
		ret.Vendor = &vendor
	}

	model, err := cmd.Flags().GetString("model")
	if err != nil {
		return nil, fmt.Errorf("model: %w", err)
	}
	if model != "" {
		ret.Model = &model
	}

	layerInt, err := cmd.Flags().GetInt64("layer")
	if err != nil {
		return nil, fmt.Errorf("layer: %w", err)
	}
	if layerInt > -1 {
		layer := uint64(layerInt)
		ret.Layer = &layer
	}

	indexInt, err := cmd.Flags().GetInt64("index")
	if err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}
	if indexInt > -1 {
		index := uint64(indexInt)
		ret.Index = &index
	}

	classIDText, err := cmd.Flags().GetString("class-id")
	if err != nil {
		return nil, fmt.Errorf("class-id: %w", err)
	}

	if classIDText != "" {
		classIDBytes, classIDType, err := parseID(classIDText)
		if err != nil {
			return nil, fmt.Errorf("class-id: %w", err)
		}

		ret.ClassBytes = &classIDBytes
		if classIDType != "" {
			ret.ClassType = &classIDType
		}
	}

	instanceIDText, err := cmd.Flags().GetString("instance-id")
	if err != nil {
		return nil, fmt.Errorf("instance-id: %w", err)
	}

	if instanceIDText != "" {
		instanceIDBytes, instanceIDType, err := parseID(instanceIDText)
		if err != nil {
			return nil, fmt.Errorf("instance-id: %w", err)
		}

		ret.InstanceBytes = &instanceIDBytes
		if instanceIDType != "" {
			ret.InstanceType = &instanceIDType
		}
	}

	groupIDText, err := cmd.Flags().GetString("group-id")
	if err != nil {
		return nil, fmt.Errorf("group-id: %w", err)
	}

	if groupIDText != "" {
		groupIDBytes, groupIDType, err := parseID(groupIDText)
		if err != nil {
			return nil, fmt.Errorf("group-id: %w", err)
		}

		ret.GroupBytes = &groupIDBytes
		if groupIDType != "" {
			ret.GroupType = &groupIDType
		}
	}

	return &ret, nil
}

func parseID(text string) ([]byte, string, error) {
	var typeText string
	var valueText string

	parts := strings.SplitN(text, ":", 2)
	if len(parts) == 2 {
		typeText = parts[0]
		valueText = parts[1]
	} else {
		valueText = text
	}

	switch typeText {
	case "uuid":
		ret, err := uuid.Parse(valueText)
		return ret[:], "uuid", err
	case "oid":
		var ret comid.OID
		if err := ret.FromString(valueText); err != nil {
			return nil, "oid", err
		}
		return []byte(ret), "oid", nil
	case "hex":
		ret, err := hex.DecodeString(valueText)
		return ret, "hex", err
	default: // assume base64
		// remove padding
		valueText = strings.Trim(valueText, "=")
		// if URL, convert to standard
		valueText = strings.ReplaceAll(valueText, "-", "+")
		valueText = strings.ReplaceAll(valueText, "_", "/")

		ret, err := base64.RawStdEncoding.DecodeString(valueText)
		return ret, typeText, err
	}
}

func init() {
	getCmd.Flags().StringP("class-id", "C", "", "Environment class ID.")
	getCmd.Flags().StringP("vendor", "V", "", "Environment vendor.")
	getCmd.Flags().StringP("model", "M", "", "Environment model.")
	getCmd.Flags().Int64P("layer", "L", -1, "Environment layer.")
	getCmd.Flags().Int64P("index", "I", -1, "Environment index.")
	getCmd.Flags().StringP("instance-id", "i", "", "Environment instance ID")
	getCmd.Flags().StringP("group-id", "g", "", "Environment group ID")

	getCmd.Flags().BoolP("reference-values", "R", false, "Look up reference values.")
	getCmd.Flags().BoolP("endorsements", "E", false, "Look up endorsements.")
	getCmd.Flags().BoolP("trust-anchors", "T", false, "Look up trust anchors.")

	getCmd.Flags().StringP("label", "l", "",
		"Label that will be applied to the manifest in the store.")

	getCmd.Flags().BoolP("exact", "e", false,
		"Match environments exactly, including null fields. The default is to assume that "+
			"null fields (i.e. fields not explicitly specified) can match any value.")

	rootCmd.AddCommand(getCmd)
}
