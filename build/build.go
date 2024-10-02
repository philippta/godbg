package build

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Package struct {
	ImportPath string
	Dir        string
	Name       string
}

func PackageInfo(path string) (Package, error) {
	if path == "" {
		path = "."
	}
	out, err := exec.Command("go", "list", "-json", path).Output()
	if err != nil {
		return Package{}, fmt.Errorf("run \"go list -json %s\": %w", path, err)
	}
	var pkg Package
	if err := json.Unmarshal(out, &pkg); err != nil {
		return Package{}, fmt.Errorf("decoding package info: %w", err)
	}
	return pkg, nil
}

func Build(path string) (string, error) {
	if path == "" {
		path = "."
	}
	cmd := exec.Command("go", "build", "-o", "godbg.bin", "-gcflags", "-N -l", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return filepath.Abs("godbg.bin")
}

func Test(path string) (string, error) {
	if path == "" {
		path = "."
	}
	cmd := exec.Command("go", "test", "-c", "-o", "godbg.test", path, "-args", "-gcflags", "all='-N -l'")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return filepath.Abs("godbg.test")
}

func TestFunctions(testBinPath string, funcExpr string) ([]string, error) {
	if funcExpr == "" {
		funcExpr = ".*"
	}
	abspath, err := filepath.Abs(testBinPath)
	if err != nil {
		return nil, fmt.Errorf("abs path of %q: %w", testBinPath, err)
	}

	out, err := exec.Command(abspath, "-test.list", funcExpr).Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}
