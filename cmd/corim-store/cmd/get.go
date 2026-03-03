package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/model"
	storemod "github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim/comid"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get triples matching specified environment.",
	Long: `Get triples matching specified environment.

Flags are used to specify the elements of the environment. Multiple flags can
be used together (e.g. you can specify a class ID and a model). If a particular
environment element is not specified, it can be any value in the matched
environments; unless --exact flag is also used, in which case unspecified elements
must also be unset in the matched environments.

In addition to environment matching, flags can be used to specify that you only
want to get active triples, and/or only reference values or only trust anchors
(by default, all triples with matching environments will be returned).

The triples are returned encoded as JSON.` + flagsHelp + timeHelp,
	Args: cobra.NoArgs,

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runGetCommand(cmd, args))
	},
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	selector, err := buildSelector(cmd)
	if err != nil {
		return err
	}

	store, err := storemod.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	var result comid.Triples

	if selector.Endorsements || selector.ReferenceValues {
		query, err := BuildValueTripleQuery(cmd.Flags())
		if err != nil {
			return err
		}

		found, err := store.QueryValueTripleModels(query)
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
		query, err := BuildKeyTripleQuery(cmd.Flags())
		if err != nil {
			return err
		}

		found, err := store.QueryKeyTripleModels(query)
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

func init() {
	AddQueryFlags(getCmd)

	getCmd.Flags().BoolP("reference-values", "R", false, "Look up reference values.")
	getCmd.Flags().BoolP("endorsements", "E", false, "Look up endorsements.")
	getCmd.Flags().BoolP("trust-anchors", "T", false, "Look up trust anchors.")

	rootCmd.AddCommand(getCmd)
}
