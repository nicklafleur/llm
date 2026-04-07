package model

import (
	"fmt"
	"sort"
	"strings"
)

var quantByModel = map[string]map[string]string{
	"qwen7b": {
		"4": "Q4_K_M",
		"8": "Q8_0",
	},
	"qwen30b": {
		"2": "UD-Q2_K_XL",
		"4": "UD-Q4_K_XL",
		"6": "UD-Q6_K_XL",
	},
	"qwen80b": {
		"2":  "UD-Q2_K_XL",
		"3":  "UD-Q3_K_XL",
		"3s": "UD-IQ3_XSS",
		"4":  "UD-Q4_K_XL",
		"i4n": "UD-IQ4_NL",
		"i4s": "UD-IQ4_XS",
		"q4s": "UD-Q4_K_S",
	},
}

func MapQuantFlag(model, bits string) (string, error) {
	per, ok := quantByModel[model]
	if !ok {
		return "", fmt.Errorf("model %q has no -q presets; omit -q and set quant in srv.ini [hf], or add an entry to quantByModel", model)
	}
	q, ok := per[bits]
	if !ok {
		allowed := make([]string, 0, len(per))
		for k := range per {
			allowed = append(allowed, k)
		}
		sort.Strings(allowed)
		return "", fmt.Errorf("invalid -q %q for model %q (allowed: %s)", bits, model, strings.Join(allowed, ", "))
	}
	return q, nil
}