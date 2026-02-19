package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/model"
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

func AddEnviromentFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("class-id", "C", "", "Environment class ID.")
	cmd.Flags().StringP("vendor", "V", "", "Environment vendor.")
	cmd.Flags().StringP("model", "M", "", "Environment model.")
	cmd.Flags().Int64P("layer", "L", -1, "Environment layer.")
	cmd.Flags().Int64P("index", "I", -1, "Environment index.")
	cmd.Flags().StringP("instance-id", "i", "", "Environment instance ID")
	cmd.Flags().StringP("group-id", "g", "", "Environment group ID")
}

func BuildEnvironment(cmd *cobra.Command) (*comid.Environment, error) {
	var ret comid.Environment
	var cls comid.Class

	vendor, err := cmd.Flags().GetString("vendor")
	if err != nil {
		return nil, fmt.Errorf("vendor: %w", err)
	}
	if vendor != "" {
		cls.Vendor = &vendor
	}

	model, err := cmd.Flags().GetString("model")
	if err != nil {
		return nil, fmt.Errorf("model: %w", err)
	}
	if model != "" {
		cls.Model = &model
	}

	layerInt, err := cmd.Flags().GetInt64("layer")
	if err != nil {
		return nil, fmt.Errorf("layer: %w", err)
	}
	if layerInt > -1 {
		layer := uint64(layerInt)
		cls.Layer = &layer
	}

	indexInt, err := cmd.Flags().GetInt64("index")
	if err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}
	if indexInt > -1 {
		index := uint64(indexInt)
		cls.Index = &index
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

		cls.ClassID, err = comid.NewClassID(classIDBytes, classIDType)
		if err != nil {
			return nil, fmt.Errorf("class-id: %w", err)
		}
	}

	if cls.ClassID != nil || cls.Vendor != nil || cls.Model != nil || cls.Layer != nil || cls.Index != nil {
		ret.Class = &cls
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

		ret.Instance, err = comid.NewInstance(instanceIDBytes, instanceIDType)
		if err != nil {
			return nil, fmt.Errorf("instance-id: %w", err)
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

		ret.Group, err = comid.NewGroup(groupIDBytes, groupIDType)
		if err != nil {
			return nil, fmt.Errorf("group-id: %w", err)
		}
	}

	return &ret, nil
}

func EnvironmentIsEmpty(env *comid.Environment) bool {
	return env.Class == nil && env.Instance == nil && env.Group == nil
}

func RenderEnviroment(env *model.Environment) (string, error) {
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

func parseID(text string) ([]byte, string, error) {
	var typeText string
	var valueText string

	parts := strings.SplitN(text, ":", 2)
	if len(parts) == 2 {
		typeText = parts[0]
		valueText = parts[1]
	} else {
		typeText = "bytes"
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
		return ret, "bytes", err
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
