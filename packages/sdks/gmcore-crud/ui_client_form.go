package gmcore_crud

import _ "embed"

//go:embed runtime/choice_picker.js
var crudClientChoicePickerJS string

//go:embed runtime/form.js
var crudClientFormJS string

func crudClientFormScript() string {
	return crudClientChoicePickerJS + crudClientFormJS
}
