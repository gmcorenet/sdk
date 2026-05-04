package gmcore_crud

import (
	"strings"
	"testing"
)

func TestRuntimeUsesFormalCrudPortalAPI(t *testing.T) {
	core := crudClientCoreScript()
	if !strings.Contains(core, "function openCrudPortal(options)") {
		t.Fatalf("expected shared openCrudPortal API in runtime core")
	}
	if !strings.Contains(core, "themeForNode(anchor)") {
		t.Fatalf("expected openCrudPortal to derive portal theme from its anchor")
	}
	if !strings.Contains(core, "restoreScrollTop") {
		t.Fatalf("expected openCrudPortal to own scroll retention")
	}
	if !strings.Contains(core, "themeName != null") {
		t.Fatalf("ensureFloatingPortal should not reset portal theme when callers only read portal state")
	}

	form := crudClientFormScript()
	for _, expected := range []string{
		"name: 'crud-floating-relation'",
		"anchorAttr: 'data-crud-relation-anchor'",
		"portalName: 'crud-floating-choice'",
		"anchorAttr: 'data-crud-choice-anchor'",
		"openCrudPortal({",
	} {
		if !strings.Contains(form, expected) {
			t.Fatalf("expected form runtime to use openCrudPortal with %q", expected)
		}
	}
	if strings.Contains(form, "positionFloatingPortal('crud-floating-relation'") {
		t.Fatalf("relation runtime should not position relation portal outside openCrudPortal")
	}

	actions := crudClientActionsScript()
	if !strings.Contains(actions, "openCrudPortal({") || !strings.Contains(actions, "name: 'crud-floating-menu'") {
		t.Fatalf("actions runtime should open row action menus through openCrudPortal")
	}
	for _, expected := range []string{
		"data-row-action-config",
		"crud-action-group",
		"crud-action-separator",
		"window.confirm(confirmText)",
		"function isServerAction(item)",
		"fetchJSON(apiURL('/api/action')",
		"dispatchMutation('server_action'",
		"window.location.href = url",
		"window.open(url, target || '_blank')",
	} {
		if !strings.Contains(actions, expected) {
			t.Fatalf("actions runtime missing custom action support %q", expected)
		}
	}
	for _, expected := range []string{
		"crudIconRegistry",
		"function crudIcon(name, fallback)",
		"GMCoreCrudIconRegistry",
	} {
		if !strings.Contains(core, expected) {
			t.Fatalf("core runtime missing formal icon registry support %q", expected)
		}
	}
}
