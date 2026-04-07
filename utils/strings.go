package utils

import "strings"

func SplitHF(hf string) (base, quant string) {
	i := strings.LastIndex(hf, ":")
	if i < 0 {
		return hf, ""
	}
	return hf[:i], hf[i+1:]
}