package main

import (
	"fmt"
	"os"

	"llm/cmd"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "llm: load .env: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.RootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}