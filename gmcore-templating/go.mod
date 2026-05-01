module gmcore-templating

go 1.19

require (
	gmcore-debugbar v0.0.0
	gmcore-resolver v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace gmcore-debugbar => ../gmcore-debugbar

replace gmcore-resolver => ../gmcore-resolver
