package util

import (
	"runtime"
	"strings"
	"sync"
)

const (
	unknownModVerion = "@unknown"
	mainModVersion   = "@main"
)

var (
	knownMods sync.Map
)

type modVersion struct {
	modVersion string
	mod        string
}

type ModVersion interface{ Get() string }

func NewModVersion(mod string) ModVersion {
	v := &modVersion{mod: strings.Trim(mod, "/")}
	defer func() {
		if !isUnknown(v.modVersion) {
			knownMods.Store(mod, v.modVersion)
		}
	}()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		v.modVersion = mod + unknownModVerion
		return v
	}
	v.modVersion = v.analyzeModeVersion(file)
	return v
}

func (a *modVersion) Get() string { return a.modVersion }
func (a *modVersion) analyzeModeVersion(file string) string {
	if strings.Contains(file, "/"+a.mod+"/") {
		return a.mod + mainModVersion
	} else if idx := strings.Index(file, "/"+a.mod+"@"); idx > 0 {
		for endIdx := idx + len(a.mod) + 2; endIdx < len(file); endIdx++ {
			if file[endIdx] == '/' {
				return file[idx+1 : endIdx]
			}
		}
	}
	return a.mod + unknownModVerion
}

func isUnknown(ver string) bool { return strings.HasSuffix(ver, unknownModVerion) }

func RangeKnownModVersions(f func(mod, ver string) bool) {
	knownMods.Range(func(key, value any) bool { return f(key.(string), value.(string)) })
}
