package cmd

import (
	"llm/config/preset"
	"llm/runner"

	"github.com/spf13/cobra"
)

func SrvCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "srv [ini]",
		Short: "Run llama-server with multi-model preset",
		Long:  "Optional ini is a path (relative to the current directory) to the models preset file; it overrides --config when given.",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := *configPath
			if len(args) > 0 {
				path = args[0]
			}
			ini, err := preset.Load(path)
			if err != nil {
				return err
			}
			bin, err := runner.FindBinary("llama-server")
			if err != nil {
				return err
			}
			return runner.Exec(bin, "-cram", "0", "--models-preset", ini.Path)
		},
	}
}
