package core

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
)

type FileSystem struct {
	Overlay map[string][]byte

	// 需要被保持为原有内容的文件
	Preserving map[string][]byte
}

func NewFileSystem() *FileSystem {
	return &FileSystem{
		Overlay:    make(map[string][]byte),
		Preserving: make(map[string][]byte),
	}
}

func (f *FileSystem) read(filename string) ([]byte, error) {
	filename = getAbsPathWithBestEfforts(filename)

	if b, ok := f.Overlay[filename]; ok {
		return b, nil
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	f.Overlay[filename] = b
	return b, nil
}

func (f *FileSystem) backup(filename string, data []byte) {
	filename = getAbsPathWithBestEfforts(filename)

	f.Preserving[filename] = data
}

func (f *FileSystem) write(filename string, data []byte) {
	filename = getAbsPathWithBestEfforts(filename)

	f.Overlay[filename] = data
}

func (f *FileSystem) Cached(filename string) bool {
	filename = getAbsPathWithBestEfforts(filename)

	if _, ok := f.Overlay[filename]; ok {
		return true
	}
	return false
}

func (f *FileSystem) Flush() error {
	for path, data := range f.Overlay {
		if _, ok := f.Preserving[path]; ok {
			data = f.Preserving[path]
		}

		err := os.WriteFile(path, data, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileSystem) FlushToStdout() error {
	for path, data := range f.Overlay {
		if _, ok := f.Preserving[path]; ok {
			data = f.Preserving[path]
		}

		if _, err := os.Stdout.Write([]byte(path + "\n")); err != nil {
			return err
		}

		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileSystem) BrieflyShortenToFirstLine(path string) error {
	data, err := f.read(path)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(bytes.NewReader(data))
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return err
	}

	// 用第一行代码来代替测试文件内容
	f.write(path, line)

	// 在写回时，我们将测试文件恢复到原来的状态
	f.backup(path, data)

	return nil
}

// 在使用overlay时，如果传入的filename是相对路径，有可能导致老版本的golang.org/x/tools loader
// 报告一个panic错误: paths xxx and yyy both canonicalize to xxx in overlay file Replace map
func getAbsPathWithBestEfforts(filename string) string {
	abspath, err := filepath.Abs(filename)
	if err != nil {
		return filename
	}

	return abspath
}
