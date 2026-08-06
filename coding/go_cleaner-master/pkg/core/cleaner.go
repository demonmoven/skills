package core

import (
	"bufio"
	"go/token"
	"os"

	"code.byted.org/analyzers/go_cleaner/pkg/core/imports"
)

type Cleaner interface {
	Load()
	CollectDef()
	CollectUsed()
	CleanPrepare()
	Clean() *CleanResult
}

type CleanResult struct {
	NumOfCodeLines int
	NumOfRemoved   int
	TextEdits      []TextEdit
}

func (c *CleanResult) Flush(fset *token.FileSet) bool {
	if len(c.TextEdits) == 0 {
		return false
	}

	type Replacement struct {
		Beg     int
		End     int
		NewText []byte
	}

	changes := make(map[string]map[int]Replacement)
	for _, edit := range c.TextEdits {
		start := fset.Position(edit.Beg).Line
		stop := start
		if edit.End != token.NoPos {
			stop = fset.Position(edit.End).Line
		}
		filename := fset.Position(edit.Beg).Filename
		if _, ok := changes[filename]; !ok {
			changes[filename] = make(map[int]Replacement)
		}

		changes[filename][start] = Replacement{
			Beg:     start,
			End:     stop,
			NewText: edit.NewText,
		}
	}

	applyChanges := func(filename string, targets map[int]Replacement) error {
		srcFile, _ := os.Open(filename)
		scanner := bufio.NewScanner(srcFile)

		var buf []byte
		currentLine := 1
		replacement := Replacement{}
		var ok bool
		for scanner.Scan() {
			if currentLine < replacement.End {
				currentLine += 1
				continue
			}

			if currentLine == replacement.End {
				replacement.NewText = append(replacement.NewText, '\n')
				buf = append(buf, replacement.NewText...)
				replacement = Replacement{}
				currentLine += 1
				continue
			}

			if replacement, ok = targets[currentLine]; ok {
				currentLine += 1
				continue
			}

			buf = append(buf, scanner.Bytes()...)
			buf = append(buf, '\n')
			currentLine++
		}

		if err := srcFile.Close(); err != nil {
			return err
		}

		return writeBufferToFile(buf, filename)
	}

	for filename, targets := range changes {
		if err := applyChanges(filename, targets); err != nil {
			panic(err)
		}
	}

	return true
}

func writeBufferToFile(buffer []byte, filename string) error {
	buffer, err := imports.Process(filename, buffer, nil)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := file.Write(buffer); err != nil {
		return err
	}

	return file.Sync()
}

type TextEdit struct {
	Beg     token.Pos
	End     token.Pos
	NewText []byte
}
