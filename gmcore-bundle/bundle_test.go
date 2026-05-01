package gmcorebundle

import "testing"

func TestBootstrapImportPathUsesExplicitModule(t *testing.T) {
	manifest := Manifest{
		Name:    "acme/maps-bundle",
		Package: "maps-bundle",
		Module:  "github.com/acme/gmcore-maps-bundle",
		Bootstrap: BootstrapSpec{
			Import: "src",
		},
	}

	if got := manifest.BootstrapImportPath(); got != "github.com/acme/gmcore-maps-bundle/src" {
		t.Fatalf("unexpected import path: %q", got)
	}
}

func TestBootstrapImportPathKeepsLegacyModuleFallback(t *testing.T) {
	manifest := Manifest{
		Name:    "gmcore/crud-bundle",
		Package: "crud-bundle",
		Bootstrap: BootstrapSpec{
			Import: "src",
		},
	}

	if got := manifest.BootstrapImportPath(); got != "gmcore-crud-bundle/src" {
		t.Fatalf("unexpected import path: %q", got)
	}
}
