package preset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesTaggedIniKeys(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "turbo.ini")
	ini := `[nomic-embed]
hf-repo = nomic-ai/nomic-embed-text-v2-moe-GGUF:F32
embeddings = true
ctx-size = 8192
`
	if err := os.WriteFile(iniPath, []byte(ini), 0o644); err != nil {
		t.Fatalf("write ini: %v", err)
	}

	p, err := Load(iniPath)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}

	model, ok := p.Get("nomic-embed")
	if !ok {
		t.Fatal("expected nomic-embed section")
	}
	if model.HFRepo != "nomic-ai/nomic-embed-text-v2-moe-GGUF:F32" {
		t.Fatalf("unexpected hf-repo: %q", model.HFRepo)
	}
	if !model.Embeddings {
		t.Fatal("expected embeddings=true to be parsed")
	}
	if model.CtxSize != 8192 {
		t.Fatalf("unexpected ctx-size: %d", model.CtxSize)
	}
}
