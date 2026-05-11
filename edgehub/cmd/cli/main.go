package main

import (
	"fmt"
	"os"

	"github.com/edgehub/edgehub/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "edge",
		Short: "EdgeHub CLI - Edge Computing Aggregation Platform",
		Long: `EdgeHub CLI provides command-line interface for managing edge computing resources.
Complete documentation is available at https://docs.edgehub.io`,
		Version: "1.0.0",
	}

	rootCmd.AddCommand(cli.NewNodeCommand())
	rootCmd.AddCommand(cli.NewJobCommand())
	rootCmd.AddCommand(cli.NewMarketCommand())
	rootCmd.AddCommand(cli.NewClusterCommand())
	rootCmd.AddCommand(cli.NewConfigCommand())
	rootCmd.AddCommand(cli.NewLoginCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
