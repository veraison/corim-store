package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/veraison/corim-store/pkg/build"
)

var configFile string
var cliConfig *Config

var rootCmd = &cobra.Command{
	Use:     "corim-store COMMAND COMMAND_ARGS...",
	Short:   "CoRIM store utility",
	Version: build.Version.String(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		CheckErr(cliConfig.Check())
	},
}

func Execute() {
	_ = rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(func() {
		cliConfig = NewConfig()
		cliConfig.Init(configFile)
	})

	rootCmd.PersistentFlags().StringVar(
		&configFile, "config", "",
		"Path to the config file (default is $XDG_CONFIG_HOME/corim-store.yaml).",
	)

	rootCmd.PersistentFlags().Bool(
		"no-color", false, "Disable color output.",
	)

	rootCmd.PersistentFlags().Bool(
		"insecure", false, "Allow insecure operations.",
	)

	rootCmd.PersistentFlags().Bool(
		"trace-sql", false, "Enable SQL tracing.",
	)

	rootCmd.PersistentFlags().Bool(
		"force", false, "Force an operation that would otherwise fail (use with care!).",
	)

	rootCmd.PersistentFlags().StringP(
		"dbms", "D", "sqlite", "DataBase Management System type",
	)

	rootCmd.PersistentFlags().StringP(
		"dsn", "N", "file:store.db?cache=shared", "Datadase System Name",
	)

	rootCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "config" {
			// it doesn't make sense to bind the location of the config file to
			// a config inside that file.
			return
		}

		CheckErr(viper.BindPFlag(flag.Name, flag))
	})

}
