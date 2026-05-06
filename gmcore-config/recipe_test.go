package gmcore_config

import (
	"strings"
	"testing"
)

func TestRecipeCatalogIncludesLifecycleAndRateLimit(t *testing.T) {
	names := map[string]bool{}
	for _, recipe := range AllRecipes {
		names[recipe.Name] = true
	}

	if !names["gmcore-lifecycle"] {
		t.Fatalf("gmcore-lifecycle recipe missing")
	}
	if !names["gmcore-ratelimit"] {
		t.Fatalf("gmcore-ratelimit recipe missing")
	}
}

func TestTransportRecipeContainsExposureDefaults(t *testing.T) {
	var transport Recipe
	found := false
	for _, recipe := range AllRecipes {
		if recipe.Name == "gmcore-transport" {
			transport = recipe
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("transport recipe not found")
	}
	if len(transport.ConfigFiles) == 0 {
		t.Fatalf("transport recipe has no config files")
	}

	content := transport.ConfigFiles[0].Content
	if !strings.Contains(content, "exposure:") {
		t.Fatalf("transport recipe must include exposure defaults")
	}
	if !strings.Contains(content, "mode: internal") {
		t.Fatalf("transport exposure defaults must set internal mode")
	}
}
