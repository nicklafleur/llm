package preset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IniPreset represents a parsed .ini model preset configuration
type IniPreset struct {
	Path     string                 // absolute path to the .ini file
	Models   map[string]ModelConfig // model name -> configuration
}

// ModelConfig represents a single model's configuration from an .ini section
type ModelConfig struct {
	Name          string // section name (model name)
	HFRepo        string // huggingface repository
	CtxSize       int    // context size (0 if not set)
	CacheTypeK    string // cache type for keys
	CacheTypeV    string // cache type for values
	FlashAttn     string // flash attention setting
	NCPUMoe       int    // number of CPU MoE experts (0 if not set)
	Temp          float64 // temperature (0.0 if not set)
	MinP          float64 // min-p sampling (0.0 if not set)
	TopP          float64 // top-p sampling (0.0 if not set)
	TopK          int    // top-k sampling (0 if not set)
	Mmap          string // mmap setting
	Embeddings    bool   // embeddings mode
	OtherFields   map[string]string // any other fields not explicitly parsed
}

// Get returns a model config by section name.
func (p *IniPreset) Get(name string) (ModelConfig, bool) {
	model, ok := p.Models[name]
	return model, ok
}

// ToMap converts the model config back into key/value pairs expected by runners.
func (m ModelConfig) ToMap() map[string]string {
	out := make(map[string]string, len(m.OtherFields)+10)
	if m.HFRepo != "" {
		out["hf-repo"] = m.HFRepo
	}
	if m.CtxSize != 0 {
		out["ctx-size"] = fmt.Sprintf("%d", m.CtxSize)
	}
	if m.CacheTypeK != "" {
		out["cache-type-k"] = m.CacheTypeK
	}
	if m.CacheTypeV != "" {
		out["cache-type-v"] = m.CacheTypeV
	}
	if m.FlashAttn != "" {
		out["flash-attn"] = m.FlashAttn
	}
	if m.NCPUMoe != 0 {
		out["n-cpu-moe"] = fmt.Sprintf("%d", m.NCPUMoe)
	}
	if m.Temp != 0 {
		out["temp"] = fmt.Sprintf("%g", m.Temp)
	}
	if m.MinP != 0 {
		out["min-p"] = fmt.Sprintf("%g", m.MinP)
	}
	if m.TopP != 0 {
		out["top-p"] = fmt.Sprintf("%g", m.TopP)
	}
	if m.TopK != 0 {
		out["top-k"] = fmt.Sprintf("%d", m.TopK)
	}
	if m.Mmap != "" {
		out["mmap"] = m.Mmap
	}
	if m.Embeddings {
		out["embeddings"] = "true"
	}
	for k, v := range m.OtherFields {
		out[k] = v
	}
	return out
}

// Load parses an .ini file and returns a preset configuration
// path can be absolute, relative, or empty (will search default locations)
func Load(path string) (*IniPreset, error) {
	iniPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(iniPath)
	if err != nil {
		return nil, fmt.Errorf("open .ini file: %w", err)
	}
	defer f.Close()

	preset := &IniPreset{
		Path:   iniPath,
		Models: make(map[string]ModelConfig),
	}

	scanner := bufio.NewScanner(f)
	var currentModel *ModelConfig
	var currentSectionName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.Trim(line, "[]")

			// If we have a previous model, save it
			if currentModel != nil && currentSectionName != "" {
				preset.Models[currentSectionName] = *currentModel
			}

			// Start new model section
			currentModel = &ModelConfig{
				Name:        sectionName,
				OtherFields: make(map[string]string),
			}
			currentSectionName = sectionName
			continue
		}

		// Parse key=value pairs within a section
		if currentModel == nil {
			continue
		}

		if key, val, ok := strings.Cut(line, "="); ok {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)

			// Set specific fields based on key name
			switch key {
			case "hf-repo":
				currentModel.HFRepo = val
			case "ctx-size":
				currentModel.CtxSize = parseInt(val)
			case "cache-type-k":
				currentModel.CacheTypeK = val
			case "cache-type-v":
				currentModel.CacheTypeV = val
			case "flash-attn":
				currentModel.FlashAttn = val
			case "n-cpu-moe":
				currentModel.NCPUMoe = parseInt(val)
			case "temp":
				currentModel.Temp = parseFloat(val)
			case "min-p":
				currentModel.MinP = parseFloat(val)
			case "top-p":
				currentModel.TopP = parseFloat(val)
			case "top-k":
				currentModel.TopK = parseInt(val)
			case "mmap":
				currentModel.Mmap = val
			case "embeddings":
				currentModel.Embeddings = strings.ToLower(val) == "true"
			default:
				currentModel.OtherFields[key] = val
			}
		}
	}

	// Save the last model section
	if currentModel != nil && currentSectionName != "" {
		preset.Models[currentSectionName] = *currentModel
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read .ini file: %w", err)
	}

	// Validate required fields
	if err := validatePreset(preset); err != nil {
		return nil, err
	}

	return preset, nil
}

// validatePreset checks that the preset has required data
func validatePreset(preset *IniPreset) error {
	if len(preset.Models) == 0 {
		return fmt.Errorf("no models found in .ini file: %s", preset.Path)
	}

	for name, model := range preset.Models {
		if model.HFRepo == "" {
			return fmt.Errorf("model %q missing required field 'hf-repo' in %s", name, preset.Path)
		}
	}

	return nil
}

// resolvePath resolves the .ini file path, following the same logic as ResolveSrvIni
func resolvePath(path string) (string, error) {
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			return "", fmt.Errorf(".ini file not found: %s", abs)
		}
		return abs, nil
	}

	// Check environment variable
	if v := os.Getenv("llm_INI"); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs, nil
		}
	}

	// Check default candidates
	candidates := []string{
		"srv.ini",
		filepath.Join("bin", "srv.ini"),
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "srv.ini"),
			filepath.Join(exeDir, "bin", "srv.ini"),
			filepath.Join(exeDir, "..", "bin", "srv.ini"),
		)
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs, nil
		}
	}

	return "", fmt.Errorf(".ini file not found (use --config, set llm_INI, or run from repo with srv.ini)")
}

// parseInt safely parses an integer string
func parseInt(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

// parseFloat safely parses a float string
func parseFloat(s string) float64 {
	var n float64
	_, _ = fmt.Sscanf(s, "%f", &n)
	return n
}