package endpoint

import (
	"path/filepath"
	"testing"
)

func TestKitexSimple(t *testing.T) { // 最基础的kitex项目布局
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "ToClean1,ToClean2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/kitex-simple"),
	}, checkExpectedFiles)
}

func TestKitexNeedMain(t *testing.T) { // 单module多main文件，且main.go不在go.mod所在目录
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "ToClean1,ToClean2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/kitex-need_main"),
		MainDir:     "rpc_service",
	}, checkExpectedFiles)
}

func TestKitexHandlerDir(t *testing.T) { // kitex项目，handler.go文件在handler目录下
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "ToClean1,ToClean2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/kitex-handler_dir"),
	}, checkExpectedFiles)
}

func TestKitexSpecifyHandler(t *testing.T) { // kitex项目，handler.go不在main.go所在目录，且不在handler目录下
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "ToClean1,ToClean2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/kitex-specify_handler"),
		HandlerPath: "impl",
	}, checkExpectedFiles)
}
