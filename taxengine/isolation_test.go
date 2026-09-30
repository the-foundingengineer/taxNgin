package taxengine_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTaxengineAndReturnsImportOnlyStdlib(t *testing.T) {
	mod := moduleRoot(t)
	for _, pkg := range []string{"taxsoftware/taxengine", "taxsoftware/internal/returns"} {
		cmd := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", pkg)
		cmd.Dir = mod
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v\n%s", pkg, err, out.String())
		}
		for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line == pkg {
				continue
			}
			if strings.HasPrefix(line, "taxsoftware/") {
				t.Errorf("%s depends on non-stdlib module package %s", pkg, line)
			}
			if !strings.HasPrefix(line, "taxsoftware") && line != pkg {
				t.Errorf("%s imports %s (not stdlib)", pkg, line)
			}
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
