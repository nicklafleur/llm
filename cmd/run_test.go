package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitRunPresetArgUsesLeadingIniPath(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "custom.ini")
	if err := os.WriteFile(iniPath, []byte("[qwen80b]\nhf-repo = repo:quant\n"), 0o644); err != nil {
		t.Fatalf("write ini: %v", err)
	}

	configPath, args := splitRunPresetArg([]string{iniPath, "qwen80b", "-b"}, "")
	if configPath != iniPath {
		t.Fatalf("expected config path %q, got %q", iniPath, configPath)
	}
	if len(args) != 2 || args[0] != "qwen80b" || args[1] != "-b" {
		t.Fatalf("unexpected remaining args: %#v", args)
	}
}

func TestSplitRunPresetArgLeavesModelFirstArgsUntouched(t *testing.T) {
	configPath, args := splitRunPresetArg([]string{"qwen80b", "-b"}, "srv.ini")
	if configPath != "srv.ini" {
		t.Fatalf("expected config flag to be preserved, got %q", configPath)
	}
	if len(args) != 2 || args[0] != "qwen80b" || args[1] != "-b" {
		t.Fatalf("unexpected remaining args: %#v", args)
	}
}

func TestLooksLikeIniPathAllowsNonexistentNestedIniPath(t *testing.T) {
	if !looksLikeIniPath(filepath.Join("configs", "turbo.ini")) {
		t.Fatal("expected nested .ini path to be recognized")
	}
}
