package internal

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ignoreHashPath = func() string {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		return filepath.Join(usr.HomeDir, ".go_cleaner_ignore_hash")
	}()
	ignoreCommitWithMessage = "chore:"
)

var blameRegexp = regexp.MustCompile("^.*?\\((.*?)\\s+([\\d-]*?)\\s+(\\d*?)\\).*")

type Blame struct {
	data map[string]*BitMap
}

func MustNewBlame(beforeM int) *Blame {
	blame := &Blame{
		data: map[string]*BitMap{},
	}

	if beforeM <= 0 {
		return blame
	}

	before := time.Now().AddDate(0, -beforeM, 0)

	_ = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
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
			if err := blame.collectRecentlyModifiedLine(absPath, before); err != nil {
				panic("collect git info err: " + err.Error())
			}
		}
		return nil
	})

	return blame
}

func (b *Blame) collectRecentlyModifiedLine(filePath string, before time.Time) error {
	ignore, err := os.OpenFile(ignoreHashPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer ignore.Close()
	cmd := exec.Command("git", "log", "--all", "--pretty=format:%H", fmt.Sprintf("--grep=%s", ignoreCommitWithMessage))
	cmd.Stdout = ignore
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		fmt.Println("Run command failed: ", strings.Join(cmd.Args, " "))
		return err
	}

	if _, err = ignore.Seek(0, io.SeekStart); err != nil {
		fmt.Println("Seek file content failed: ", ignoreHashPath)
		return err
	}

	ignoreRaw, err := io.ReadAll(ignore)
	if err != nil {
		return fmt.Errorf("Blame.collectRecentlyModifiedLine read ignore hash err: %w", err)
	}
	ignoreHashs := map[string]bool{}
	for _, line := range bytes.Split(ignoreRaw, []byte{'\n'}) {
		if len(line) >= 8 {
			ignoreHashs[string(line[:8])] = true
		}
	}

	cmd = exec.Command(
		"git",
		"blame",
		"--date", "short",
		"--ignore-revs-file", ignoreHashPath,
		"--", filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Run command failed: ", strings.Join(cmd.Args, " "))
		return err
	}
	lines := bytes.Split(output, []byte{'\n'})
	bitmap := NewBitMap(len(lines) + 1)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// ^904b885 (
		if v := bytes.Split(line, []byte{' '}); len(v) > 0 {
			if hash := bytes.TrimPrefix(v[0], []byte{'^'}); ignoreHashs[string(hash)] {
				continue
			}
		}
		if re := blameRegexp.FindSubmatch(line); len(re) >= 4 {
			author := string(re[1])
			modified := string(re[2])
			number, err := strconv.ParseInt(string(re[3]), 10, 64)
			if err != nil {
				return fmt.Errorf("git blame invalid output: %s, err: %w", line, err)
			}
			if author == "Not Committed Yet" || modified == "" {
				continue
			}
			if t, err := time.Parse("2006-01-02", modified); err == nil && t.After(before) {
				bitmap.Add(int(number))
			}
		}
	}
	if _, ok := b.data[filePath]; !ok {
		b.data[filePath] = bitmap
	}
	return nil
}

func (b *Blame) RecentlyModifiedWithin(start, end token.Position) bool {
	file := start.Filename
	if _, ok := b.data[file]; !ok {
		return false
	}
	bs := b.data[file]
	for line := start.Line; line <= end.Line; line++ {
		if bs.Has(line) {
			return true
		}
	}
	return false
}
