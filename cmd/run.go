package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"llm/config/preset"
	"llm/model"
	"llm/runner"
	"llm/utils"

	"github.com/spf13/cobra"
)

var errHelp = errors.New("help requested")

func RunCommand(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [ini] <model> [-c|-b] [-q N] [--] [llama-args...]",
		Short: "Run llama-server, llama-cli, or llama-bench for a model",
		Long: strings.TrimSpace(`
Model Runner

Available models: qwen30b, qwen80b, qwen7b, default (qwen80b), plus any [section] in srv.ini with an hf-repo= line.

Options (after model name):
  [ini]    Optional path to a preset .ini file; overrides --config when given
  -c       CLI mode (llama-cli)
  -b       Benchmark (llama-bench)
  -q N     Override [hf-repo] quant (qwen7b: 4,8; qwen30b: 2,4,6; qwen80b: 2,3,3s,4,4s); other models: use srv.ini only

Examples:
  llm run turbo.ini qwen30b
  llm run qwen30b -b -q 6
  llm run qwen80b -b -q 3
  llm run qwen30b -c -q 4
`),
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			return Run(c, args, *configPath)
		},
	}
	return cmd
}

func Run(cmd *cobra.Command, args []string, configFlag string) error {
	configPath, args := splitRunPresetArg(args, configFlag)
	if len(args) == 0 {
		return fmt.Errorf("no model specified (use -h for help)")
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(cmd.Long)
		fmt.Println()
		return nil
	}

	baseModel := args[0]
	if baseModel == "default" {
		baseModel = "qwen80b"
	}

	mode, quantBits, passthrough, err := parseRunFlags(args[1:])
	if errors.Is(err, errHelp) {
		fmt.Print(cmd.Long)
		fmt.Println()
		return nil
	}
	if err != nil {
		return err
	}

	p, err := preset.Load(configPath)
	if err != nil {
		return err
	}

	modelConfig, ok := p.Get(baseModel)
	if !ok {
		return fmt.Errorf("model %q not found in %s", baseModel, p.Path)
	}
	sec := modelConfig.ToMap()

	hf := sec["hf-repo"]
	if hf == "" {
		return fmt.Errorf("model %q missing required field 'hf-repo' in %s", baseModel, p.Path)
	}

	repoBase, quantFromIni := utils.SplitHF(hf)
	quant := quantFromIni
	if quantBits != "" {
		quant, err = model.MapQuantFlag(baseModel, quantBits)
		if err != nil {
			return err
		}
	}
	if quant == "" {
		return fmt.Errorf("no quantization for model %s (set hf-repo in srv.ini or use -q)", baseModel)
	}
	sec["hf-repo"] = repoBase + ":" + quant

	skip := map[string]bool{}
	var cmdArgs []string

	switch mode {
	case "server":
		cmdArgs = append(cmdArgs, "--jinja")
	case "cli":
		cmdArgs = append(cmdArgs, "--jinja")
	case "bench":
		cmdArgs = []string{"-p", "1000", "-n", "50"}
		if flashAttn := strings.TrimSpace(sec["flash-attn"]); flashAttn != "" {
			cmdArgs = append(cmdArgs, "--flash-attn", flashAttn)
		}
		skip["flash-attn"] = true
		skip["ctx-size"] = true
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}

	runArgs := runner.LlamaRunArgsFromSection(sec, skip)
	cmdArgs = append(cmdArgs, runArgs...)
	cmdArgs = append(cmdArgs, passthrough...)

	bin, err := runner.FindBinary("llama-" + mode)
	if err != nil {
		return err
	}

	return runner.Exec(bin, cmdArgs...)
}

func splitRunPresetArg(args []string, configFlag string) (string, []string) {
	if len(args) < 2 {
		return configFlag, args
	}
	if !looksLikeIniPath(args[0]) {
		return configFlag, args
	}
	return args[0], args[1:]
}

func looksLikeIniPath(arg string) bool {
	if !strings.HasSuffix(strings.ToLower(arg), ".ini") {
		return false
	}
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	return strings.Contains(arg, "/") || strings.Contains(arg, "\\") || filepath.Base(arg) != arg
}

func parseRunFlags(args []string) (mode string, quantBits string, passthrough []string, err error) {
	mode = "server"
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-h" || a == "--help" {
			return "", "", nil, errHelp
		}
		if a == "--" {
			passthrough = append(passthrough, args[i+1:]...)
			break
		}
		switch a {
		case "-c":
			mode = "cli"
			i++
		case "-b":
			mode = "bench"
			i++
		case "-q":
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("-q requires a value")
			}
			quantBits = args[i+1]
			i += 2
		default:
			if strings.HasPrefix(a, "-") {
				return "", "", nil, fmt.Errorf("unknown option: %s", a)
			}
			passthrough = append(passthrough, args[i:]...)
			i = len(args)
		}
	}
	return mode, quantBits, passthrough, nil
}
