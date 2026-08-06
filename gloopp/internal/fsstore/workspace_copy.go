package fsstore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// PrepareSandboxCopy creates a discardable copy of the quest workspace for a
// single agent session. It intentionally reuses the same ignore rules as normal
// workspace copy so .git, .gloop and local trash directories do not leak into
// the sandbox.
func (qs *QuestStore) PrepareSandboxCopy(qid, sid, sourceDir string) (string, error) {
	if qid == "" {
		return "", fmt.Errorf("qid 不能为空")
	}
	if sid == "" {
		return "", fmt.Errorf("session id 不能为空")
	}
	if sourceDir == "" {
		return "", fmt.Errorf("sourceDir 不能为空")
	}
	dst := filepath.Join(qs.dir(qid), "sandboxes", sid)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	if err := copyDir(sourceDir, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func (qs *QuestStore) RemoveSandbox(qid, sid string) error {
	if qid == "" || sid == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(qs.dir(qid), "sandboxes", sid))
}

func SandboxDiff(baseDir, sandboxDir string) ([]DiffFileStat, int, int, error) {
	return diffDirs(baseDir, sandboxDir)
}

// --- 目录拷贝 ---

func copyDir(src, dst string) error {
	ignore := loadGloopIgnore(src)
	return copyDirWithIgnore(src, dst, "", ignore)
}

func copyDirWithIgnore(src, dst, rel string, ignore gloopIgnore) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s 不是目录", src)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childRel := filepath.Join(rel, entry.Name())
		if shouldSkipWorkspacePath(entry.Name(), childRel, entry.IsDir(), ignore) {
			continue
		}
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyDirWithIgnore(srcPath, dstPath, childRel, ignore); err != nil {
				return err
			}
		} else {
			// 跳过符号链接（避免循环）
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		// ETXTBSY: 目标是正在执行的二进制。先 unlink 再创建新文件。
		if errors.Is(err, syscall.ETXTBSY) {
			if rmErr := os.Remove(dst); rmErr != nil {
				return fmt.Errorf("remove busy file: %w (original: %w)", rmErr, err)
			}
			out, err = os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		}
		if err != nil {
			return err
		}
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// --- 目录 diff（copy 模式用）---

// diffDirs 对比两个目录的文件差异。返回变更文件列表、总新增行数、总删除行数。
// 行数统计对文本文件有效，二进制文件按 1 行计算。
func diffDirs(baseDir, workDir string) ([]DiffFileStat, int, int, error) {
	var files []DiffFileStat
	totalAdd := 0
	totalDel := 0

	baseFiles := map[string]os.FileInfo{}
	workFiles := map[string]os.FileInfo{}

	ignore := loadGloopIgnore(baseDir)
	if err := listFilesWithIgnore(baseDir, "", baseFiles, ignore); err != nil {
		return nil, 0, 0, err
	}
	if err := listFilesWithIgnore(workDir, "", workFiles, ignore); err != nil {
		return nil, 0, 0, err
	}

	// 检查新增和修改
	for path, wInfo := range workFiles {
		if _, ok := baseFiles[path]; !ok {
			add := countLines(filepath.Join(workDir, path))
			files = append(files, DiffFileStat{
				Path:      path,
				Status:    "added",
				Additions: add,
				Deletions: 0,
			})
			totalAdd += add
		} else {
			bInfo := baseFiles[path]
			same, err := filesEqual(filepath.Join(baseDir, path), filepath.Join(workDir, path), bInfo, wInfo)
			if err != nil {
				return nil, 0, 0, err
			}
			if !same {
				add, del := diffFileLines(filepath.Join(baseDir, path), filepath.Join(workDir, path))
				files = append(files, DiffFileStat{
					Path:      path,
					Status:    "modified",
					Additions: add,
					Deletions: del,
				})
				totalAdd += add
				totalDel += del
			}
		}
	}

	// 检查删除
	for path := range baseFiles {
		if _, ok := workFiles[path]; !ok {
			del := countLines(filepath.Join(baseDir, path))
			files = append(files, DiffFileStat{
				Path:      path,
				Status:    "deleted",
				Additions: 0,
				Deletions: del,
			})
			totalDel += del
		}
	}

	return files, totalAdd, totalDel, nil
}

func filesEqual(basePath, workPath string, baseInfo, workInfo os.FileInfo) (bool, error) {
	if baseInfo.Size() != workInfo.Size() {
		return false, nil
	}
	baseFile, err := os.Open(basePath)
	if err != nil {
		return false, err
	}
	defer baseFile.Close()

	workFile, err := os.Open(workPath)
	if err != nil {
		return false, err
	}
	defer workFile.Close()

	const bufSize = 32 * 1024
	baseBuf := make([]byte, bufSize)
	workBuf := make([]byte, bufSize)
	for {
		baseN, baseErr := baseFile.Read(baseBuf)
		workN, workErr := workFile.Read(workBuf)
		if baseN != workN || !bytes.Equal(baseBuf[:baseN], workBuf[:workN]) {
			return false, nil
		}
		if baseErr == io.EOF && workErr == io.EOF {
			return true, nil
		}
		if baseErr != nil && baseErr != io.EOF {
			return false, baseErr
		}
		if workErr != nil && workErr != io.EOF {
			return false, workErr
		}
	}
}

func listFiles(root, rel string, out map[string]os.FileInfo) error {
	return listFilesWithIgnore(root, rel, out, loadGloopIgnore(root))
}

func listFilesWithIgnore(root, rel string, out map[string]os.FileInfo, ignore gloopIgnore) error {
	dir := filepath.Join(root, rel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		childRel := filepath.Join(rel, name)
		if shouldSkipWorkspacePath(name, childRel, entry.IsDir(), ignore) {
			continue
		}
		if entry.IsDir() {
			if err := listFilesWithIgnore(root, childRel, out, ignore); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			// 跳过符号链接
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			out[childRel] = info
		}
	}
	return nil
}

func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 1
	}
	if len(data) == 0 {
		return 0
	}
	n := 0
	for _, b := range data {
		if b == '\n' {
			n++
		}
	}
	// 最后一行没有换行也算
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func diffFileLines(basePath, workPath string) (int, int) {
	baseData, err1 := os.ReadFile(basePath)
	workData, err2 := os.ReadFile(workPath)
	if err1 != nil || err2 != nil {
		return countLines(workPath), countLines(basePath)
	}
	// 简单行 diff：统计各自行数做近似
	// 精确 diff 算法太重了，MVP 用行数差近似
	baseLines := countLinesFromBytes(baseData)
	workLines := countLinesFromBytes(workData)
	add := workLines - baseLines
	if add < 0 {
		add = 0
	}
	del := baseLines - workLines
	if del < 0 {
		del = 0
	}
	return add, del
}

func countLinesFromBytes(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := 0
	for _, b := range data {
		if b == '\n' {
			n++
		}
	}
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}
