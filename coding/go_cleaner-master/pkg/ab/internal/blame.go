package internal

import (
	"bytes"
	"fmt"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

var blameRegexp = regexp.MustCompile("^.*?\\((.*?)\\s+([\\d-]*?)\\s+(\\d*?)\\).*")

type BlameInfo struct {
	author   string
	modifyAt string
}

type Blame struct {
	data map[string][]BlameInfo
}

func NewBlame() *Blame {
	blame := &Blame{
		data: map[string][]BlameInfo{},
	}

	filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == "vendor" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		absPath, _ := filepath.Abs(path)
		if filepath.Ext(path) == ".go" {
			_ = blame.collect(absPath)
		}
		return nil
	})

	return blame
}

func (b *Blame) Search(pos token.Position) BlameInfo {
	file := pos.Filename
	line := pos.Line
	if v, ok := b.data[file]; !ok || len(v) <= line {
		return BlameInfo{}
	}
	blameInfo := b.data[file][line]
	return blameInfo
}

func (b *Blame) collect(filePath string) error {
	cmd := exec.Command(
		"git",
		"blame",
		"--date", "short",
		"--", filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return err
	}
	if _, ok := b.data[filePath]; !ok {
		b.data[filePath] = []BlameInfo{}
	}
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		if re := blameRegexp.FindSubmatch(line); len(re) >= 4 {
			author := string(re[1])
			time := string(re[2])
			number, err := strconv.ParseInt(string(re[3]), 10, 64)
			if err != nil {
				fmt.Println(err, string(re[1]), string(re[2]), string(re[3]))
			}
			for len(b.data[filePath]) <= int(number) {
				b.data[filePath] = append(b.data[filePath], BlameInfo{})
			}
			b.data[filePath][number].author = author
			b.data[filePath][number].modifyAt = time
		}
	}
	return nil
}

func (b *Blame) Within(start, end token.Position, beforeM int) bool {
	threshold := time.Now().AddDate(0, -beforeM, 0)
	file := start.Filename
	if _, ok := b.data[file]; !ok {
		return false
	}
	bs := b.data[file]
	for line := start.Line; line <= end.Line && line < len(bs); line++ {
		// 未提交的修改不会触发新提交保护机制，避免本地多次执行时可能产生的问题
		// bs[line].modifyAt可能为空, git blame --ignore-revs-file 可能导致为空
		if bs[line].author == "Not Committed Yet" || bs[line].modifyAt == "" {
			continue
		}
		if t, err := time.Parse("2006-01-02", bs[line].modifyAt); err == nil && t.After(threshold) {
			return true
		}
	}
	return false
}
