package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "symgrep",
	Short: "symgrep is a CLI tool for extracting code symbols using Tree-sitter",
	Long:  `symgrep allows you to extract specific functions, classes, and methods from source files using Tree-sitter queries for high precision.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}
