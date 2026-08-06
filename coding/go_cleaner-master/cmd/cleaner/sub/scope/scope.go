package scope

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.byted.org/analyzers/go_cleaner/pkg/out"
	"code.byted.org/analyzers/go_cleaner/pkg/util"
	"code.byted.org/lang/gg/gmap"
	"code.byted.org/lang/gg/gslice"
	"github.com/sirupsen/logrus"
)

// https://bytedance.larkoffice.com/docx/Ey1pdFe3goJwRKxj5itcaDbRnwg
type scoper struct {
	repoAppMains []string // 代码仓库下包括的PSM程序入口
	mainDir      string   // 程序入口目录
	repoRoot     string   // 仓库根目录
}

func (s *scoper) Eval() ([]string, error) {
	if mainDir == "." {
		return nil, nil // 如果main-pkg在仓库根目录下，判定为非monorepo仓库，不进行分析
	}
	// 1. 仅清理当前module下的代码，不清理跨项目引用的情况（提供拓展参数：include-modules，仅支持仓库下module，清理了不一定生效，可能不是local replace方式的引用）
	modDir, modName, err := util.ModuleDirAndNameOf(s.mainDir) // 找到main-dir所在的module目录
	if modDir == "" {
		return nil, fmt.Errorf("failed find go.mod: %v", err)
	}
	appScopes, err := s.loadAppsCompilePkgs(modDir, modName)
	if err != nil {
		return nil, fmt.Errorf("failed load apps compile pkgs: %v", err)
	}
	scope := s.excludeCommonPkgs(appScopes)
	return gslice.Map(scope.Pkgs, func(pkg *appPkg) string { return pkg.Path }), nil
}

func (s *scoper) runModTidy(modDir string) bool {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = modDir
	errBuf := strings.Builder{}
	cmd.Stdout, cmd.Stderr = io.Discard, &errBuf
	if err := cmd.Run(); err != nil {
		out.Warnf("[scope]failed run mod tidy: %v, %s", err, errBuf.String())
		return false
	}
	return true
}

func (s *scoper) loadAppsCompilePkgs(modDir, modName string) (map[string]*appScope, error) {
	if !s.runModTidy(modDir) {
		return nil, fmt.Errorf("failed run mod tidy")
	}
	var (
		wg         sync.WaitGroup
		appActions sync.Map
	)
	for _, appMain := range s.repoAppMains {
		appModDir, _, _ := util.ModuleDirAndNameOf(appMain)
		if appModDir != modDir {
			fmt.Printf("[scope]skip load app(%s) compile pkgs, not in same module\n", appMain)
			continue
		}
		appMain := appMain
		wg.Add(1)
		go func() {
			defer wg.Done()
			actions, err := s.getMainCompilePkgs(appMain)
			if err != nil {
				appActions.Store(appMain, err)
			} else {
				appActions.Store(appMain, actions)
			}
		}()
	}

	wg.Wait()
	appActionMap := map[string][]*ActionJSON{}
	failApp := []string{}
	appActions.Range(func(key, value interface{}) bool {
		switch v := value.(type) {
		case error:
			failApp = append(failApp, key.(string))
			fmt.Println("failed get compile pkgs for app: ", key, "; err: ", v)
		case []*ActionJSON:
			appActionMap[key.(string)] = v
		}
		return true
	})
	if len(failApp) > 0 {
		return nil, fmt.Errorf("failed get compile pkgs for apps: %v", failApp)
	}
	return s.arrangeAppsCompile(modDir, modName, appActionMap)
}

func (s *scoper) arrangeAppsCompile(curModDir, curModName string, appActions map[string][]*ActionJSON) (map[string]*appScope, error) {
	appScopes := map[string]*appScope{}
	unknownPkg := []string{}
	for appMain, actions := range appActions {
		scope := &appScope{MainDir: appMain}
		for _, a := range actions {
			if a.Mode == "link-install" {
				scope.MainPkgPath = a.Package
			} else if a.Mode == "build" && strings.HasPrefix(a.Package, curModName) {
				_, pkgName := filepath.Split(a.Package) // 只处理当前module下的package
				var relPath string
				if a.Package == curModName {
					relPath, _ = filepath.Rel(s.repoRoot, curModDir)
				} else {
					pkgDir := filepath.Join(curModDir, strings.TrimPrefix(a.Package, curModName+"/"))
					relPath, _ = filepath.Rel(s.repoRoot, pkgDir)
				}
				if relPath == "" {
					fmt.Println("failed get rel path for pkg: ", a.Package, "; repo root: ", s.repoRoot, "; cur mod dir: ", curModDir)
					unknownPkg = append(unknownPkg, "#"+a.Package)
					continue
				}
				scope.Pkgs = append(scope.Pkgs, &appPkg{
					PkgName: pkgName,
					PkgPath: a.Package,
					Path:    relPath,
				})
			}
		}
		if err := s.includeMainSubDirs(scope); err != nil {
			logrus.Warnf("[scope]failed include main sub dirs: %v", err)
			return nil, fmt.Errorf("failed include main sub dirs: %v", err)
		}
		appScopes[appMain] = scope
	}
	if len(unknownPkg) > 0 {
		logrus.Warnf("[scope]unknown pkg: %v", unknownPkg)
		return nil, fmt.Errorf("unknown pkg: %v", unknownPkg)
	}
	return appScopes, nil
}

func (s *scoper) includeMainSubDirs(app *appScope) error {
	mainDir := filepath.Join(s.repoRoot, app.MainDir)
	dirs := map[string]*dirInfo{}
	err := filepath.WalkDir(mainDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if mainDir == path {
			return nil
		} else if d.IsDir() {
			_, err := os.Stat(filepath.Join(path, "go.mod"))
			if err == nil {
				logrus.Infof("[scope]skip module dir: %s", path)
				return filepath.SkipDir
			} else if os.IsNotExist(err) {
				return nil
			} else { // 别的不确定有什么错误，先直接跳过
				return nil
			}
		}
		dir, file := filepath.Split(path)
		dir = filepath.Clean(dir)
		if strings.HasSuffix(file, ".go") {
			if dirs[dir] == nil {
				dirs[dir] = &dirInfo{Dir: dir}
			}
			dirs[dir].GoFiles = append(dirs[dir].GoFiles, &fileInfo{
				Name:    file,
				Path:    strings.TrimPrefix(path, s.repoRoot),
				PkgPath: strings.Replace(dir, mainDir, app.MainPkgPath, 1),
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed walk dir: %s, err: %v", mainDir, err)
	}
	compilePkgs := gslice.ToMapValues(app.Pkgs, func(p *appPkg) string { return filepath.Join(s.repoRoot, p.Path) })
	for _, dir := range dirs {
		if compilePkgs[dir.Dir] != nil {
			compilePkgs[dir.Dir].UnderMain = true
			continue
		}
		relDir, _ := filepath.Rel(s.repoRoot, dir.Dir)
		_, pkgName := filepath.Split(dir.Dir)
		app.Pkgs = append(app.Pkgs, &appPkg{
			PkgName:   pkgName,
			PkgPath:   strings.Replace(dir.Dir, mainDir, app.MainPkgPath, 1),
			Path:      strings.Trim(relDir, "/"),
			UnderMain: true,
		})
	}
	return nil
}

func (s *scoper) getMainCompilePkgs(mainDir string) ([]*ActionJSON, error) {
	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("action_%d_%s.json", time.Now().Unix(), util.RandString(10)))
	cmd := exec.Command("go", "list", "-compiled", "-debug-actiongraph="+outputFile)
	cmd.Dir = filepath.Join(s.repoRoot, mainDir)
	errBuf := strings.Builder{}
	cmd.Stderr, cmd.Stdout = &errBuf, io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed run list go compiled for main: %s, err:%v, %s", mainDir, err, errBuf.String())
	}
	cnt, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed read action graph file(%s): %v", outputFile, err)
	}
	return parseActions(cnt)
}

func (s *scoper) excludeCommonPkgs(apps map[string]*appScope) *appScope {
	reused := map[string]bool{}
	for _, app := range apps {
		if app.MainDir == s.mainDir {
			continue
		}
		for _, pkg := range app.Pkgs {
			reused[pkg.PkgPath] = true // 被其他app-main复用的pkg，认定为公共pkg，不做清理
		}
	}
	targetApp := apps[s.mainDir]
	if targetApp == nil {
		panic(fmt.Errorf("main app not found: %s in %v", s.mainDir, gmap.Keys(apps)))
	}
	excludeDirs := []string{}
	excluded := appScope{MainDir: targetApp.MainDir, MainPkgPath: targetApp.MainPkgPath}
	for _, pkg := range targetApp.Pkgs {
		if pkg.UnderMain { // main-pkg下的，直接圈定
			excluded.Pkgs = append(excluded.Pkgs, pkg)
		} else if !reused[pkg.PkgPath] { // 被其他app使用的，不做圈定
			excluded.Pkgs = append(excluded.Pkgs, pkg)
		} else {
			excludeDirs = append(excludeDirs, pkg.Path)
		}
	}
	out.SetInfof("[scope]exclude dirs: %v", excludeDirs)
	return &excluded
}

type dirInfo struct {
	Dir     string
	GoFiles []*fileInfo
}

type fileInfo struct {
	Name    string
	Path    string
	PkgPath string
}

type appScope struct {
	MainDir     string
	MainPkgPath string
	Pkgs        []*appPkg
}

type appPkg struct {
	PkgName   string
	PkgPath   string
	Path      string
	UnderMain bool // 目录在main-pkg的目录下
}
