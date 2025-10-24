package cmd

import (
	"fmt"
	"os"
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
