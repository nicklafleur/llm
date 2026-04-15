package preset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// IniPreset represents a parsed .ini model preset configuration
type IniPreset struct {
	Path   string                 // absolute path to the .ini file
	Models map[string]ModelConfig // model name -> configuration
}

// ModelConfig represents a single model's configuration from an .ini section
type ModelConfig struct {
	Name        string            `flag:"-"`            // section name (model name)
	HFRepo      string            `flag:"hf-repo"`      // huggingface repository
	CtxSize     int               `flag:"ctx-size"`     // context size (0 if not set)
	CacheTypeK  string            `flag:"cache-type-k"` // cache type for keys
	CacheTypeV  string            `flag:"cache-type-v"` // cache type for values
	FlashAttn   string            `flag:"flash-attn"`   // flash attention setting
	NCPUMoe     int               `flag:"n-cpu-moe"`    // number of CPU MoE experts (0 if not set)
	Temp        float64           `flag:"temp"`         // temperature (0.0 if not set)
	MinP        float64           `flag:"min-p"`        // min-p sampling (0.0 if not set)
	TopP        float64           `flag:"top-p"`        // top-p sampling (0.0 if not set)
	TopK        int               `flag:"top-k"`        // top-k sampling (0 if not set)
	Mmap        string            `flag:"mmap"`         // mmap setting
	Embeddings  bool              `flag:"embeddings"`   // embeddings mode
	OtherFields map[string]string `flag:"-"`            // any other fields not explicitly parsed
}

// Get returns a model config by section name.
func (p *IniPreset) Get(name string) (ModelConfig, bool) {
	model, ok := p.Models[name]
	return model, ok
}

// ToMap converts the model config back into key/value pairs expected by runners.
func (m ModelConfig) ToMap() map[string]string {
	out := make(map[string]string, len(m.OtherFields)+10)

	t := reflect.TypeFor[ModelConfig]()
	v := reflect.ValueOf(m)

	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldType.IsExported() {
			continue
		}

		// Get the "flag" tag
		flagTag := fieldType.Tag.Get("flag")
		if flagTag == "" || flagTag == "-" {
			continue
		}

		switch fieldType.Type.Kind() {
		case reflect.String:
			if val := fieldValue.String(); val != "" {
				out[flagTag] = val
			}
		case reflect.Int:
			if val := fieldValue.Int(); val != 0 {
				out[flagTag] = fmt.Sprintf("%d", val)
			}
		case reflect.Float64:
			if val := fieldValue.Float(); val != 0 {
				out[flagTag] = fmt.Sprintf("%g", val)
			}
		case reflect.Bool:
			if val := fieldValue.Bool(); val {
				out[flagTag] = "true"
			}
		}
	}

	// Add any other fields from OtherFields
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

			// Try to set the field by name
			if !setField(currentModel, key, val) {
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

// setField attempts to set a struct field by its name (case-sensitive)
// Returns false if no matching field is found
func setField(model *ModelConfig, key, val string) bool {
	v := reflect.ValueOf(model).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		flagName := fieldType.Tag.Get("flag")
		if flagName == "" || flagName == "-" {
			continue
		}

		// INI keys map to the struct's `flag` tags, not Go field names.
		if flagName != key {
			continue
		}

		fieldValue := v.Field(i)

		// Set value based on field type
		switch fieldType.Type.Kind() {
		case reflect.String:
			fieldValue.SetString(val)
		case reflect.Int:
			i, err := strconv.Atoi(val)
			if err != nil {
				return false
			}
			fieldValue.SetInt(int64(i))
		case reflect.Float64:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return false
			}
			fieldValue.SetFloat(f)
		case reflect.Bool:
			fieldValue.SetBool(strings.ToLower(val) == "true")
		default:
			return false
		}

		return true
	}

	// No matching field found
	return false
}
