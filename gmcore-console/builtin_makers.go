package gmcoreconsole

import (
	"fmt"
)

type appMaker struct {
	Name  string
	Usage string
	Run   func(appPath string, args []string) error
}

func builtInMakers() map[string]appMaker {
	return map[string]appMaker{
		"crud": {
			Name:  "crud",
			Usage: "bin/console make crud <Name>",
			Run:   makeCrud,
		},
		"entity": {
			Name:  "entity",
			Usage: "bin/console make entity <Name>",
			Run:   makeEntity,
		},
		"form": {
			Name:  "form",
			Usage: "bin/console make form <Name>",
			Run:   makeForm,
		},
		"controller": {
			Name:  "controller",
			Usage: "bin/console make controller <Name>",
			Run:   makeController,
		},
		"command": {
			Name:  "command",
			Usage: "bin/console make command <name>",
			Run:   makeCommand,
		},
		"service": {
			Name:  "service",
			Usage: "bin/console make service <Name>",
			Run:   makeService,
		},
	}
}

func makeCrud(appPath string, args []string) error {
	return delegateToFramework("make:crud", appPath, args)
}

func makeEntity(appPath string, args []string) error {
	return delegateToFramework("make:entity", appPath, args)
}

func makeForm(appPath string, args []string) error {
	return delegateToFramework("make:form", appPath, args)
}

func makeController(appPath string, args []string) error {
	return delegateToFramework("make:controller", appPath, args)
}

func makeCommand(appPath string, args []string) error {
	return delegateToFramework("make:command", appPath, args)
}

func makeService(appPath string, args []string) error {
	return delegateToFramework("make:service", appPath, args)
}

func delegateToFramework(makerName string, appPath string, args []string) error {
	resolved, ok := resolveTool(appPath, "makers", makerName)
	if !ok {
		return fmt.Errorf("maker %q not found in framework", makerName)
	}
	toolArgs := append([]string{makerName}, args...)
	return runTool(Context{AppRoot: appPath, Stdout: nil, Stderr: nil}, resolved, toolArgs)
}
