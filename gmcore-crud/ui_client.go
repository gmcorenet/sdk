package gmcorecrud

func crudClientScript() string {
	return crudClientCoreScript() +
		crudClientFormScript() +
		crudClientDesignerScript() +
		crudClientActionsScript() +
		crudClientBootScript()
}
