package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/veraison/corim-store/pkg/store"
	"github.com/veraison/corim/corim"
)

var corimCmd = &cobra.Command{
	Use:   "corim",
	Short: "CoRIM-related operations.",
	Long: `CoRIM-related operations.

Subcommands allow adding and removing CoRIMs from the store. See help for the
individual subcommands.
	`,

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runAddCommand(cmd, args))
	},
}

var addCmd = &cobra.Command{
	Use:   "add PATH [PATH ...]",
	Short: "Add a CoRIM's contents to the store.",
	Long: `Add a CoRIM's contents to the store.

The specified CoRIM(s) will be parsed and added as a "manifest" to the store.
Currently, CoRIMs containing only CoMID tags, and CoMID tags containing only
reference-triple's, endorsed-triple's, and attest-key-triple's, are supported.
	`,
	Args: cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runAddCommand(cmd, args))
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete PATH_OR_MANIFEST_ID",
	Short: "Delete data associated with the specified CoRIM or manifest ID.",
	Long: `Delete data associated with the specified CoRIM or manifest ID.

You can specify the manifest ID directly, or you can specify a path to a CoRIM, in which case
the ID will be extracted from it (note: either way, the matching is done based on the ID so
the CoRIM specified does not literally have to be the same file that was previously added).
	`,
	Args: cobra.MinimumNArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runDeleteCommand(cmd, args))
	},
}

var dumpCmd = &cobra.Command{
	Use:   "dump MANIFEST_ID",
	Short: "Write a CoRIM containing data associated with the specified manifest ID.",
	Long: `Write a CoRIM containing data associated with the specified manifest ID.

This produces an unsigned CoRIM token containing the data associated with the
specified MANIFEST_ID. It is a way to easily "retrieve" a previously added
CoRIM.
	`,
	Args: cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		CheckErr(runDumpCommand(cmd, args))
	},
}

func runAddCommand(cmd *cobra.Command, args []string) error {
	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	activate, err := cmd.Flags().GetBool("activate")
	if err != nil {
		return err
	}

	store, err := store.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	for _, path := range args {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", path, err)
		}

		if err := store.AddBytes(bytes, label, activate); err != nil {
			return fmt.Errorf("error adding %s: %w", path, err)
		}

		fmt.Printf("added %s\n", path)
	}

	fmt.Println(Green("ok"))
	return nil
}

func runDumpCommand(cmd *cobra.Command, args []string) error {
	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	outpath, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	if _, err = os.Stat(outpath); err == nil && !cliConfig.Force {
		return fmt.Errorf("output file exists: %s (use --force to overwrite)", outpath)
	}

	store, err := store.Open(context.Background(), cliConfig.Store())
	if err != nil {
		return err
	}
	defer func() { CheckErr(store.Close()) }()

	manifest, err := store.GetManifest(args[0], label)
	if err != nil {
		return err
	}

	corim, err := manifest.ToCoRIM()
	if err != nil {
		return fmt.Errorf("could not convert manifest to CoRIM: %w", err)
	}

	bytes, err := corim.ToCBOR()
	if err != nil {
		return fmt.Errorf("could not encode CoRIM: %w", err)
	}

	if err := os.WriteFile(outpath, bytes, 0664); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}

	fmt.Println(Green("ok"))
	return nil
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	for _, pathOrID := range args {
		var manifestID string

		fmt.Printf("Deleting %s...\n", pathOrID)

		if _, err := os.Stat(pathOrID); err == nil {
			buf, err := os.ReadFile(pathOrID)
			if err != nil {
				return fmt.Errorf("could not read %s: %w", pathOrID, err)
			}

			var unsigned corim.UnsignedCorim
			if buf[0] == 0xd2 { // nolint:gocritic
				// tag 18 -> COSE_Sign1 -> signed corim
				var signed corim.SignedCorim
				if err := signed.FromCOSE(buf); err != nil {
					return err
				}

				unsigned = signed.UnsignedCorim
			} else if slices.Equal(buf[:3], []byte{0xd9, 0x01, 0xf5}) {
				// tag 501 -> unsigned corim
				if err := unsigned.FromCBOR(buf); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unrecognized input format")
			}

			manifestID = unsigned.GetID()
		} else {
			forceCorim, err := cmd.Flags().GetBool("corim")
			if err != nil {
				return err
			}

			if forceCorim {
				return fmt.Errorf("could not read CoRIM from %q: does not exist", pathOrID)
			}

			manifestID = pathOrID
		}

		store, err := store.Open(context.Background(), cliConfig.Store())
		if err != nil {
			return err
		}
		defer func() { CheckErr(store.Close()) }()

		if err := store.DeleteManifest(manifestID, label); err != nil {
			return err
		}
	}

	fmt.Println(Green("ok"))
	return nil
}

func init() {
	corimCmd.PersistentFlags().StringP("label", "l", "",
		"Label that will be applied to the manifest in the store.")

	addCmd.Flags().BoolP("activate", "a", false, "Activate added triples.")

	deleteCmd.Flags().BoolP("corim", "C", false,
		"force interpretation the positional argument as a path to CoRIM")

	dumpCmd.Flags().StringP("output", "o", "store-corim.cbor",
		"Output path to which the CoRIM will be written")

	corimCmd.AddCommand(addCmd)
	corimCmd.AddCommand(deleteCmd)
	corimCmd.AddCommand(dumpCmd)

	rootCmd.AddCommand(corimCmd)
}
