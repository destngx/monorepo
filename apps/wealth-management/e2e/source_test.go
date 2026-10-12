package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGivenWorkbookPathWhenResolvingThenExpandsHomeAndRequiresTheFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "Money Manager.xlsx"), []byte("PK"), 0o600); err != nil {
		t.Fatal(err)
	}
	if path, err := sourceWorkbook(" ~/Money Manager.xlsx "); err != nil || path != filepath.Join(home, "Money Manager.xlsx") {
		t.Fatalf("path = %q, err = %v", path, err)
	}
	for _, missing := range []string{"", "~/missing.xlsx"} {
		if _, err := sourceWorkbook(missing); err == nil {
			t.Fatalf("%q must be refused", missing)
		}
	}
}
