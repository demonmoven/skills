package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/updatecheck"
	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func TestBuildPostUpdateRestartCommandPrefersStableShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix shim path only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	shimDir := filepath.Join(home, "."+version.RepoName, "bin")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatalf("mkdir shim dir: %v", err)
	}
	shim := filepath.Join(shimDir, version.RepoName)
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho stub\n"), 0o755); err != nil {
		t.Fatalf("write shim: %v", err)
	}

	self := filepath.Join(shimDir, "versions", "v0.1.20", version.RepoName)
	gotPath, gotArgs := buildPostUpdateRestartCommand(self, "/tmp/gloop-data")

	if gotPath != shim {
		t.Fatalf("restart executable = %q, want stable shim %q", gotPath, shim)
	}
	wantArgs := []string{"restart", "--data-dir", "/tmp/gloop-data", "--no-open"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("restart args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestBuildNPMInstallArgsUsesBNPMAndPreferOnline(t *testing.T) {
	pkgSpec := updatecheck.PackageName + "@0.3.20"
	got := buildNPMInstallArgs(pkgSpec)
	want := []string{"install", "-g", "--registry", updatecheck.Registry, "--prefer-online", pkgSpec}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("npm args = %#v, want %#v", got, want)
	}
}

func TestManualInstallCommandIncludesRegistry(t *testing.T) {
	cmd := manualInstallCommand(updatecheck.PackageName + "@0.3.20")
	for _, want := range []string{"npm i -g", "--registry " + updatecheck.Registry, "--prefer-online", updatecheck.PackageName + "@0.3.20"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("manual install command %q does not contain %q", cmd, want)
		}
	}
}

func TestUpdateCheckHintDoesNotOfferDowngradeAsLatest(t *testing.T) {
	got := updateCheckHintFor("0.3.21", "0.3.20", "")
	if got != updateCheckHintCurrent {
		t.Fatalf("hint = %q, want %q", got, updateCheckHintCurrent)
	}
}

func TestUpdateCheckHintAllowsExplicitVersionSwitch(t *testing.T) {
	got := updateCheckHintFor("0.3.21", "0.3.20", "0.3.20")
	if got != updateCheckHintSwitchVersion {
		t.Fatalf("hint = %q, want %q", got, updateCheckHintSwitchVersion)
	}
}

func TestBuildNPMViewArgsUsesResolverPath(t *testing.T) {
	got := buildNPMViewArgs(updatecheck.PackageName + "@latest")
	want := []string{"view", updatecheck.PackageName + "@latest", "version", "dist.tarball", "--registry", updatecheck.Registry, "--prefer-online", "--json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("npm view args = %#v, want %#v", got, want)
	}
}

func TestParseNPMViewOutput(t *testing.T) {
	got, err := parseNPMViewOutput([]byte(`{"version":"0.3.20","dist.tarball":"https://bnpm.byted.org/pkg.tgz"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Version != "0.3.20" || got.Tarball != "https://bnpm.byted.org/pkg.tgz" {
		t.Fatalf("parsed = %#v", got)
	}
}

func TestCurrentNativeArchiveName(t *testing.T) {
	got, err := currentNativeArchiveName("0.3.20")
	if err != nil {
		t.Fatalf("archive name: %v", err)
	}
	expectedPlatform := runtime.GOOS
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		expectedPlatform = "windows"
		ext = ".zip"
	}
	want := "gloop-0.3.20-" + expectedPlatform + "-" + runtime.GOARCH + ext
	if got != want {
		t.Fatalf("archive = %q, want %q", got, want)
	}
}

func TestPackedPackageInstallabilityRequiresChecksumAndCurrentArchive(t *testing.T) {
	archive, err := currentNativeArchiveName("0.3.20")
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{
		"checksums.txt":          true,
		"dist/" + archive:        true,
		"scripts/npm-install.js": true,
	}
	checksums := "abc123  " + archive + "\n"

	bundled, err := checkPackedPackageInstallability("0.3.20", paths, checksums)
	if err != nil {
		t.Fatalf("check installability: %v", err)
	}
	if !bundled {
		t.Fatal("bundled = false, want true")
	}

	delete(paths, "dist/"+archive)
	bundled, err = checkPackedPackageInstallability("0.3.20", paths, checksums)
	if err != nil {
		t.Fatalf("remote fallback should be allowed when checksum exists: %v", err)
	}
	if bundled {
		t.Fatal("bundled = true, want false when archive is not packed")
	}

	_, err = checkPackedPackageInstallability("0.3.20", paths, "")
	if err == nil {
		t.Fatal("expected checksum entry error")
	}
}

func TestParseNPMPackOutputFindsTarballFilename(t *testing.T) {
	raw, _ := json.Marshal([]npmPackResult{{Filename: "bytedance-dev-gloop-0.3.20.tgz"}})
	got, err := parseNPMPackOutput(raw)
	if err != nil {
		t.Fatalf("parse pack output: %v", err)
	}
	if got.Filename != "bytedance-dev-gloop-0.3.20.tgz" {
		t.Fatalf("filename = %q", got.Filename)
	}
}
