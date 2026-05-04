package gmcore_crud

import _ "embed"

//go:embed runtime/designer_layout.js
var crudClientDesignerLayoutJS string

//go:embed runtime/designer_resize.js
var crudClientDesignerResizeJS string

//go:embed runtime/designer_inspector.js
var crudClientDesignerInspectorJS string

//go:embed runtime/designer_canvas.js
var crudClientDesignerCanvasJS string

func crudClientDesignerScript() string {
	return crudClientDesignerLayoutJS +
		crudClientDesignerResizeJS +
		crudClientDesignerInspectorJS +
		crudClientDesignerCanvasJS
}
