package thrift

import (
	"os"
	"path/filepath"
	"testing"

	"code.byted.org/analyzers/go_cleaner/pkg/idl_cleaner"
)

func TestThrift(t *testing.T) {
	repoRoot := getRepoRoot()
	t.Log(repoRoot)
	result := Clean(&idl_cleaner.Params{
		Root:    repoRoot,
		File:    "test/idls/srv1.thrift",
		Methods: []string{"Srv3Fn2", "GetSrv1Func4", "GetSrv1Func5", "GetSrv1Func2", "GetSrv1Func7", "Srv2Func2"},
		Remove:  true,
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
}

func TestWdev(t *testing.T) {
	repoRoot := getRepoRoot()
	t.Log(repoRoot)
	result := Clean(&idl_cleaner.Params{
		Root: filepath.Join(repoRoot, "../rpc_idl"),
		File: "ies_live/platform/webcast_wdev_base.thrift",
		// Write:   writeToFile(filepath.Join(repoRoot, "test/idls/webcast_wdev_base.thrift")),
		Methods: []string{"TriggerPathTopology", "UpsertClientEvent"},
		Remove:  true,
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
}

func TestProfitCore(t *testing.T) {
	// webcast/profit/profit_core.thrift
	repoRoot := getRepoRoot()
	t.Log(repoRoot)
	result := Clean(&idl_cleaner.Params{
		Root: filepath.Join(repoRoot, "../rpc_idl"),
		File: "webcast/profit/profit_core.thrift",
		// Write:   writeToFile(filepath.Join(repoRoot, "test/idls/webcast_wdev_base.thrift")),
		Methods: []string{"GetAssetsEffect", "PinMostlyUsedPageTop"},
		Remove:  true,
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
}

func getRepoRoot() string {
	wd, _ := os.Getwd()
	abs, _ := filepath.Abs(filepath.Join(wd, "../../.."))
	return abs
}
