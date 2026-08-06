package storage

import (
	"encoding/json"

	"code.byted.org/analyzers/go_cleaner/pkg/core"
	"code.byted.org/analyzers/go_cleaner/pkg/core/repoinfo"
	"code.byted.org/analyzers/go_cleaner/pkg/out"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type StorageCleaner struct {
	config *StorageConfig
	module *repoinfo.Module
	prog   *ssa.Program
	pkgs   []*ssa.Package

	matchers []StorageMatcher
	result   *core.CleanResult
}

func (c *StorageCleaner) Load() {
	psms := make(map[string]bool)

	for _, psm := range c.config.PSMs {
		psms[psm] = true
	}

	c.prog, c.pkgs = ssautil.AllPackages(c.module.Packages, ssa.InstantiateGenerics)
	c.prog.Build()
	return
}

func (c *StorageCleaner) CollectDef() {
	return
}

func (c *StorageCleaner) CollectUsed() {
	c.matchers = append(c.matchers, NewConfigMatcher(c.module, c.config.PSMs), NewRedisMatcher(c.prog, c.pkgs, c.module.Packages))

	matched := make(map[string][]UserInfo)

	for _, m := range c.matchers {
		for _, psm := range c.config.PSMs {
			if m.Match(psm) {
				matched[psm] = append(matched[psm], m.ListUsers(psm)...)
			}
		}
	}

	for _, info := range matched {
		data, _ := json.Marshal(info)
		out.SetOutput("storage_clean", string(data))
		c.result.NumOfRemoved += len(info)
	}
}

func (c *StorageCleaner) CleanPrepare() {
	for _, matcher := range c.matchers {
		for _, psm := range c.config.PSMs {
			c.result.TextEdits = append(c.result.TextEdits, matcher.GetEdits(psm)...)
		}
	}
}

func (c *StorageCleaner) Clean() *core.CleanResult {
	c.result.Flush(c.prog.Fset)
	return c.result
}

func runStorageCleaner(config *StorageConfig, m *repoinfo.Module) *core.CleanResult {
	c := &StorageCleaner{
		config:   config,
		module:   m,
		result:   &core.CleanResult{},
		matchers: []StorageMatcher{},
	}

	c.Load()
	c.CollectDef()
	c.CollectUsed()
	c.CleanPrepare()
	return c.Clean()
}
