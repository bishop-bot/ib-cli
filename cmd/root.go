package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Commit is set at build time via -ldflags.
var Commit = "unknown"

var (
	configPath string
	verbose    bool
	rootCmd    = &cobra.Command{
		Use:           "ib-cli",
		Short:         "Interactive Broker Client Portal Web API CLI",
		Long:          `A production-grade CLI for interacting with Interactive Broker's\nClient Portal Web API. Supports historical market data, authentication,\nand various account operations.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       fmt.Sprintf("%s (commit: %s)", Version, Commit),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: config.toml in current dir, ~/.ib-cli, /etc/ib-cli)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// fatal prints error and exits.
func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// info prints info message.
func info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}