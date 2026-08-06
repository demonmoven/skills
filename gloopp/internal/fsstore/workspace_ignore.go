package fsstore

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// isGloopDir 判断目录名是否是 .gloop 系统目录。
func isGloopDir(name string) bool {
	return name == GloopDir
}

type gloopIgnore []string

func loadGloopIgnore(root string) gloopIgnore {
	raw, err := os.ReadFile(filepath.Join(root, ".gloopignore"))
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = filepath.ToSlash(line)
		line = strings.TrimPrefix(line, "./")
		line = strings.TrimPrefix(line, "/")
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func shouldSkipWorkspacePath(name, rel string, isDir bool, ignore gloopIgnore) bool {
	if name == ".gloopignore" && !isDir {
		return true
	}
	if name == ".git" && isDir {
		return true
	}
	if isSystemWorkspaceDir(name) && isDir {
		return true
	}
	if isGloopDir(name) && isDir {
		return true
	}
	if isDefaultIgnoredDir(name) && isDir {
		return true
	}
	if isDefaultIgnoredFile(name) && !isDir {
		return true
	}
	return ignore.matches(rel, isDir)
}

func isDefaultIgnoredDir(name string) bool {
	switch name {
	case "node_modules", "vendor", "dist", "build", "__pycache__",
		".tox", ".venv", "venv", ".mypy_cache", ".pytest_cache",
		".next", ".nuxt", "target":
		return true
	}
	return false
}

func isDefaultIgnoredFile(name string) bool {
	for _, ext := range defaultIgnoredExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

var defaultIgnoredExts = []string{
	".exe", ".dll", ".so", ".dylib", ".a", ".o", ".obj",
	".pyc", ".pyo", ".class", ".jar",
	".out",
}

func isSystemWorkspaceDir(name string) bool {
	switch name {
	case ".Trash", ".Trashes", ".Spotlight-V100", ".fseventsd":
		return true
	default:
		return false
	}
}

func (g gloopIgnore) matches(rel string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range g {
		dirPattern := strings.HasSuffix(pattern, "/")
		p := strings.TrimSuffix(pattern, "/")
		if p == "" {
			continue
		}
		if rel == p || (isDir && dirPattern && rel == p) {
			return true
		}
		if strings.HasPrefix(rel, p+"/") {
			return true
		}
		if ok, _ := path.Match(p, rel); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(rel)); ok {
			return true
		}
	}
	return false
}
