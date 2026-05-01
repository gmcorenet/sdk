package gmcorecrud

import _ "embed"

//go:embed runtime/actions.js
var crudClientActionsJS string

func crudClientActionsScript() string {
	return crudClientActionsJS
}
