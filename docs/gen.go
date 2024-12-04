package main

import (
	"os"
	"path/filepath"

	"securae/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	docPath := ""
	if len(os.Args) > 1 {
		docPath = os.Args[1]
	}
	if docPath == "" {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		docPath = filepath.Join(pwd, "docs", "md")
	}
	if err := generateDocs(docPath); err != nil {
		panic(err)
	}
}

func generateDocs(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	return doc.GenMarkdownTree(cmd.RootCmd, path)
}
