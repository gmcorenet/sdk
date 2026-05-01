package gmcoreconsole

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func executeMaker(ctx Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bin/console make <maker> <name> [args...]")
	}
	makerName := strings.TrimSpace(args[0])
	if makerName == "" {
		return fmt.Errorf("missing maker name")
	}

	if resolved, ok := resolveTool(ctx.AppRoot, "makers", makerName); ok {
		toolArgs := append([]string{makerName}, args[1:]...)
		return runTool(ctx, resolved, toolArgs)
	}

	if _, ok := resolveTool(ctx.AppRoot, "makers", "make:"+makerName); ok {
		toolArgs := append([]string{"make:" + makerName}, args[1:]...)
		return runTool(ctx, tool{Name: "make:" + makerName, Scope: "makers"}, toolArgs)
	}

	maker, ok := builtInMakers()[makerName]
	if !ok {
		externalNames := availableToolNames(ctx.AppRoot, "makers")
		names := make([]string, 0, len(builtInMakers())+len(externalNames))
		names = append(names, externalNames...)
		for name := range builtInMakers() {
			names = append(names, name)
		}
		return fmt.Errorf("unknown maker %q (available: %s)", makerName, strings.Join(names, ", "))
	}
	return maker.Run(ctx.AppRoot, args[1:])
}

func readGoModule(appPath string) string {
	data, err := os.ReadFile(filepath.Join(appPath, "go.mod"))
	if err != nil {
		return filepath.Base(appPath)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return filepath.Base(appPath)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func printJSON(value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
