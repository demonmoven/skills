package repoinfo

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"os"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
)

func WriteToStdout(fset *token.FileSet, node *ast.File) error {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, node)
	if err != nil {
		return err
	}
	raw, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(raw)
	if err != nil {
		return err
	}
	return nil
}

func WriteToFile(fset *token.FileSet, node *ast.File, path string) error {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, node)
	if err != nil {
		return err
	}

	raw, err := imports.Process(path, buf.Bytes(), nil)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, raw, 0644)
	if err != nil {
		return err
	}

	return nil
}
