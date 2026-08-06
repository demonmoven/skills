package storage

import (
	"bufio"
	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"code.byted.org/lang/gg/gslice"
	"fmt"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type PackageInfo struct {
	Name    string
	Version string
	Module  string
}

type ConfigMatcher struct {
	storageNames []string
	pkgToConfigs map[PackageInfo]map[string]bool
	users        []UserInfo
}

func NewConfigMatcher(module *repoinfo.Module, names []string) *ConfigMatcher {
	pkgToConfigs := make(map[PackageInfo]map[string]bool)
	packages.Visit(module.Packages, nil, func(p *packages.Package) {
		configures := make(map[string]bool)
		pkginfo := PackageInfo{
			Name: p.Name,
		}

		if p.Module == nil {
			return
		}

		pkginfo.Version = p.Module.Version
		pkginfo.Module = p.Module.Path
		err := filepath.Walk(p.Module.Dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || strings.HasSuffix(path, ".go") {
				return nil
			}

			configures[path] = true

			return nil
		})

		if err != nil {
			panic(err)
		}

		pkgToConfigs[pkginfo] = configures

	})

	return &ConfigMatcher{
		pkgToConfigs: pkgToConfigs,
		storageNames: names,
	}
}

func (c *ConfigMatcher) Match(psm string) bool {
	for pkginfo, configures := range c.pkgToConfigs {
		for config := range configures {
			file, err := os.Open(config)
			if err != nil {
				panic(err)
			}

			scanner := bufio.NewScanner(file)
			line := 1
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), psm) {
					c.users = append(c.users, UserInfo{
						Position: fmt.Sprintf("%s:%d", config, line),
						Code:     strings.TrimSpace(scanner.Text()),
						ModName:  pkginfo.Module,
						Verison:  pkginfo.Version,
					})
				}
				line += 1
			}
		}
	}

	c.users = gslice.Uniq(c.users)
	return len(c.users) > 0
}

func (c *ConfigMatcher) GetInitializers(psm string) (result []ssa.CallInstruction) {
	return
}

func (c *ConfigMatcher) ListUsers(psm string) (result []UserInfo) {
	return c.users
}

func (c *ConfigMatcher) GetEdits(psm string) (result []core.TextEdit) {
	return
}
