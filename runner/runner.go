package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// LlamaRunArgsFromSection converts a model config map into CLI flags for
// llama-* binaries. Keys listed in skip are omitted. The input map is not
// modified; flag order is deterministic (sorted by key).
func LlamaRunArgsFromSection(sec map[string]string, skip map[string]bool) []string {
	var args []string

	if isMmapDisabled(sec["mmap"]) {
		args = append(args, "--no-mmap")
	}

	keys := make([]string, 0, len(sec))
	for k := range sec {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if k == "mmap" || skip[k] {
			continue
		}
		args = append(args, "--"+k, sec[k])
	}
	return args
}

// FindBinary resolves a llama binary by name (e.g. "llama-server").
// It checks LLAMA_CPP_PATH first, then falls back to PATH.
func FindBinary(name string) (string, error) {
	if dir := strings.TrimSpace(os.Getenv("LLAMA_CPP_PATH")); dir != "" {
		bin := filepath.Join(dir, name)
		if _, err := os.Stat(bin); err != nil {
			return "", fmt.Errorf("LLAMA_CPP_PATH %q does not contain %s: %w", dir, name, err)
		}
		return bin, nil
	}
	bin, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH (set LLAMA_CPP_PATH or add it to PATH): %w", name, err)
	}
	return bin, nil
}

// Exec runs a binary with the given args, wiring stdin/stdout/stderr to the
// current process. It prints the command before running.
func Exec(bin string, args ...string) error {
	fmt.Printf("Running: %s %s\n", bin, strings.Join(args, " "))
	c := exec.Command(bin, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func isMmapDisabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "disabled", "off", "0", "false", "no":
		return true
	default:
		return false
	}
}
