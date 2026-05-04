package gmcore_crud

func crudClientScript() string {
	return crudClientCoreScript() +
		crudClientFormScript() +
		crudClientDesignerScript() +
		crudClientActionsScript() +
		crudClientBootScript()
}
