package portalmap_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"taxsoftware/internal/portalmap"
	"taxsoftware/internal/rulesyaml"
)

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	// internal/portalmap/portalmap_test.go → ../../ is the repo root.
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func loadRaw(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(repoRoot(), "rules", "portals", "ng-la-etax.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read portal map: %v", err)
	}
	return raw
}

// TestLoad_DraftRejected verifies that a DRAFT portal map is refused when
// AllowDraft is not set. This gate mirrors the rule-set gate: a map nobody
// has verified against a live portal session must not reach production.
func TestLoad_DraftRejected(t *testing.T) {
	raw := loadRaw(t)
	_, err := portalmap.Load(raw, rulesyaml.Unmarshal, portalmap.LoadOptions{})
	if err == nil {
		t.Fatal("expected error loading a DRAFT portal map without AllowDraft")
	}
	if !strings.Contains(err.Error(), "DRAFT") {
		t.Errorf("expected error to mention DRAFT, got: %v", err)
	}
}

// TestLoad_AllowDraft verifies that a DRAFT map loads successfully when
// AllowDraft is set, as required in tests and the authoring tool.
func TestLoad_AllowDraft(t *testing.T) {
	raw := loadRaw(t)
	pm, err := portalmap.Load(raw, rulesyaml.Unmarshal, portalmap.LoadOptions{AllowDraft: true})
	if err != nil {
		t.Fatalf("Load with AllowDraft: %v", err)
	}
	if pm.Portal.MapVersion != "0.1.0-draft" {
		t.Errorf("MapVersion: want %q, got %q", "0.1.0-draft", pm.Portal.MapVersion)
	}
	if pm.Portal.Code != "NG-LA-ETAX" {
		t.Errorf("Code: want %q, got %q", "NG-LA-ETAX", pm.Portal.Code)
	}
	if len(pm.Pages) == 0 {
		t.Error("expected at least one page")
	}
	if len(pm.Preconditions) == 0 {
		t.Error("expected at least one precondition")
	}
	// Verify PC01 is the Payer ID gate and blocks the pack.
	pc01 := findPrecondition(t, pm, "PC01")
	if !pc01.BlocksPack {
		t.Error("PC01 (Payer ID) must have blocks_pack: true")
	}
	// Verify PC03 blocks the TCC but not the pack.
	pc03 := findPrecondition(t, pm, "PC03")
	if pc03.BlocksPack {
		t.Error("PC03 (passport photograph) must not block the pack")
	}
	if !pc03.BlocksTCC {
		t.Error("PC03 (passport photograph) must block the TCC")
	}
	// Verify TCC conditions are present.
	if len(pm.TCC.Conditions) == 0 {
		t.Error("expected TCC conditions")
	}
}

func findPrecondition(t *testing.T, pm portalmap.PortalMap, id string) portalmap.Precondition {
	t.Helper()
	for _, pc := range pm.Preconditions {
		if pc.ID == id {
			return pc
		}
	}
	t.Fatalf("precondition %q not found", id)
	return portalmap.Precondition{}
}
