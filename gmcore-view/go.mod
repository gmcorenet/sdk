module gmcore-view

go 1.19

require (
	gmcore-i18n v0.0.0
	gmcore-resolver v0.0.0
	gmcore-templating v0.0.0
)

require (
	gmcore-debugbar v0.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace gmcore-debugbar => ../gmcore-debugbar

replace gmcore-i18n => ../gmcore-i18n

replace gmcore-resolver => ../gmcore-resolver

replace gmcore-templating => ../gmcore-templating
