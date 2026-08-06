package endpoint

import (
	"path/filepath"
	"testing"
)

func TestHertzSimple(t *testing.T) {
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "/toclean/m1,/toclean/m2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/hertz-simple"),
	}, checkExpectedFiles)
}

func TestHertzWithMainDir(t *testing.T) {
	runCase(t, CleanEndpointParams{
		CleanerFlag: "-f --before=0",
		Endpoint:    "/toclean/m1,/toclean/m2",
		MeegoID:     testMeegoID,
		RepoDir:     filepath.Join(endpointTestDir, "cases/hertz-simple"),
		MainDir:     "anywhere",
	})
}
