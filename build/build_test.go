package build_test

import (
	"os"
	"testing"

	"github.com/philippta/godbg/build"
)

func TestTest(t *testing.T) {
	path, err := build.Test("")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer os.Remove(path)
	t.Logf("File: %s", path)
}

func TestPackageInfo(t *testing.T) {
	pkg, err := build.PackageInfo(".")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	t.Logf("ImportPath: %s", pkg.ImportPath)
	t.Logf("Dir: %s", pkg.Dir)
}

func TestTestFunctions(t *testing.T) {
	path, err := build.Test("")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer os.Remove(path)

	funcs, err := build.TestFunctions(path, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	t.Logf("Funcs: %v", funcs)
}
