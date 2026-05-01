package gmcorecrud

import _ "embed"

//go:embed runtime/core.js
var crudClientCoreJS string

func crudClientCoreScript() string {
	return crudClientCoreJS
}
