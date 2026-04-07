package cmd

import (
	"github.com/spf13/cobra"
)

func RootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "llm",
		Short: "Manage your local LLaMA.cpp AI server",
	}

	var path string
	rootCmd.PersistentFlags().StringVar(&path, "config", "", "path to srv.ini (default: ./srv.ini or next to executable)")
	rootCmd.AddCommand(SrvCommand(&path), RunCommand(&path))

	return rootCmd
}
