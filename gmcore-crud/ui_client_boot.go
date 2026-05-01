package gmcorecrud

import _ "embed"

//go:embed runtime/boot.js
var crudClientBootJS string

func crudClientBootScript() string {
	return crudClientBootJS
}
