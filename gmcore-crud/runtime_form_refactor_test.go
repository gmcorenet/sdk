package gmcore_crud

import (
	"strings"
	"testing"
)

func TestRuntimeUsesSharedCrudChoicePicker(t *testing.T) {
	form := crudClientFormScript()
	for _, expected := range []string{
		"function createCrudChoicePicker(config)",
		"const picker = createCrudChoicePicker({",
		"aria-haspopup",
		"role=\"option\"",
		"portalName: 'crud-floating-relation'",
		"portalName: 'crud-floating-choice'",
		"picker.visibleOptions(",
		"picker.bindOptions(",
		"picker.setActiveIndex(",
	} {
		if !strings.Contains(form, expected) {
			t.Fatalf("expected form runtime to contain %q", expected)
		}
	}
	if strings.Contains(form, "function renderChoicePickerPortal(") {
		t.Fatalf("expected relation and local choices to use createCrudChoicePicker portal API")
	}
}

func TestDesignerRuntimeIsSplitIntoFocusedModules(t *testing.T) {
	designer := crudClientDesignerScript()
	for _, expected := range []string{
		"function normalizeDesignerLayout",
		"function clearDesignerResizeState",
		"function renderDesignerModeHeader",
		"function renderDesignerToolbar",
		"function renderDesignerCanvas",
	} {
		if !strings.Contains(designer, expected) {
			t.Fatalf("expected designer runtime module to contain %q", expected)
		}
	}
	if strings.Count(designer, "function renderDesigner") < 3 {
		t.Fatalf("expected designer runtime to be assembled from multiple designer modules")
	}
}

func TestWysiwygRuntimeDoesNotUseExecCommand(t *testing.T) {
	form := crudClientFormScript()
	if strings.Contains(form, "execCommand") {
		t.Fatalf("wysiwyg runtime must not use deprecated document.execCommand")
	}
	for _, expected := range []string{
		"function wrapSelection(",
		"function toggleInline(",
		"function setBlockTag(",
		"function toggleList(",
		"data-crud-wysiwyg-mode=\"' + esc(wysiwygMode) + '\"",
	} {
		if !strings.Contains(form, expected) {
			t.Fatalf("expected modern wysiwyg helper %q", expected)
		}
	}
}
