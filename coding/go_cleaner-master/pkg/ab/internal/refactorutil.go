package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
)

func save(file *ast.File, fset *token.FileSet) {
	bf := &bytes.Buffer{}
	if err := printer.Fprint(bf, fset, file); err != nil {
		panic(err)
	}

	data, err := imports.Process(fset.Position(file.Pos()).Filename, bf.Bytes(), nil)
	if err != nil {
		fmt.Println(fmt.Errorf("imports format err: %w", err))
		data = bf.Bytes()
	}

	err = os.WriteFile(fset.Position(file.Pos()).Filename, data, 0644)
	if err != nil {
		fmt.Println(fmt.Errorf("auto clean write err: %w", err))
	}
}
