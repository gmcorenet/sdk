package gmcoreconsole

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Console struct {
	appRoot  string
	stdout   io.Writer
	stderr   io.Writer
	registry *Registry
}

type Option func(*Console)

func New(appRoot string, options ...Option) *Console {
	console := &Console{
		appRoot:  filepath.Clean(appRoot),
		stdout:   os.Stdout,
		stderr:   os.Stderr,
		registry: NewRegistry(),
	}
	for _, option := range options {
		option(console)
	}
	console.registerDefaultCommands()
	return console
}

func WithOutput(stdout, stderr io.Writer) Option {
	return func(console *Console) {
		if stdout != nil {
			console.stdout = stdout
		}
		if stderr != nil {
			console.stderr = stderr
		}
	}
}

func (console *Console) Register(command Command) {
	console.registry.Register(command)
}

func (console *Console) Run(args []string) error {
	if len(args) == 0 {
		return console.Help()
	}
	name := strings.TrimSpace(args[0])
	if name == "" || name == "--help" || name == "-h" {
		return console.Help()
	}
	if name == "help" {
		if len(args) > 1 {
			return console.CommandHelp(args[1])
		}
		return console.Help()
	}
	command, ok := console.registry.Resolve(name)
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}
	if hasHelpFlag(args[1:]) {
		return console.printCommandHelp(command)
	}
	return command.Run(Context{AppRoot: console.appRoot, Stdout: console.stdout, Stderr: console.stderr}, args[1:])
}

func (console *Console) Help() error {
	fmt.Fprintln(console.stdout, "GMCore app console")
	fmt.Fprintln(console.stdout, "")
	fmt.Fprintln(console.stdout, "Usage:")
	fmt.Fprintln(console.stdout, "  bin/console <command> [args...]")
	fmt.Fprintln(console.stdout, "")
	fmt.Fprintln(console.stdout, "Commands:")
	for _, command := range console.registry.Commands() {
		line := "  " + command.Name
		if command.Description != "" {
			padding := 22 - len(command.Name)
			if padding < 1 {
				padding = 1
			}
			line += strings.Repeat(" ", padding) + command.Description
		}
		fmt.Fprintln(console.stdout, line)
	}
	return nil
}

func (console *Console) CommandHelp(name string) error {
	command, ok := console.registry.Resolve(name)
	if !ok {
		return fmt.Errorf("unknown command %q", strings.TrimSpace(name))
	}
	return console.printCommandHelp(command)
}

func (console *Console) printCommandHelp(command Command) error {
	fmt.Fprintf(console.stdout, "GMCore app console: %s\n", command.Name)
	if command.Description != "" {
		fmt.Fprintln(console.stdout, "")
		fmt.Fprintln(console.stdout, command.Description)
	}
	fmt.Fprintln(console.stdout, "")
	fmt.Fprintln(console.stdout, "Usage:")
	if strings.TrimSpace(command.Usage) != "" {
		for _, line := range strings.Split(command.Usage, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Fprintln(console.stdout, "  "+line)
		}
	} else {
		fmt.Fprintf(console.stdout, "  bin/console %s [args...]\n", command.Name)
	}
	return nil
}

func (console *Console) registerDefaultCommands() {
	console.Register(Command{
		Name:        "make",
		Description: "Run an app maker",
		Usage:       "bin/console make <maker> <name> [args...]\nbin/console makers",
		Run: func(ctx Context, args []string) error {
			return executeMaker(ctx, args)
		},
	})
	console.Register(Command{
		Name:        "run",
		Description: "Run an app command from bin/gmcore/commands",
		Usage:       "bin/console run <command> [args...]\nbin/console commands",
		Run: func(ctx Context, args []string) error {
			return executeTool(ctx, "commands", "command", args)
		},
	})
	console.Register(Command{
		Name:        "commands",
		Description: "List app-local commands",
		Usage:       "bin/console commands",
		Run: func(ctx Context, args []string) error {
			return listTools(ctx, "commands")
		},
	})
	console.Register(Command{
		Name:        "makers",
		Description: "List app-local makers",
		Usage:       "bin/console makers",
		Run: func(ctx Context, args []string) error {
			return listTools(ctx, "makers")
		},
	})
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

type toolManifest struct {
	Name        string `yaml:"name"`
	Executable  string `yaml:"executable"`
	Usage       string `yaml:"usage"`
	Description string `yaml:"description"`
}

type tool struct {
	Name        string
	Scope       string
	Path        string
	Usage       string
	Description string
}

func executeTool(ctx Context, scope, singular string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bin/console %s <name> [args...]", singular)
	}
	name := strings.TrimSpace(args[0])
	resolved, ok := resolveTool(ctx.AppRoot, scope, name)
	if !ok {
		available := availableToolNames(ctx.AppRoot, scope)
		if len(available) == 0 {
			return fmt.Errorf("%s %q not found", singular, name)
		}
		return fmt.Errorf("%s %q not found (available: %s)", singular, name, strings.Join(available, ", "))
	}
	return runTool(ctx, resolved, args[1:])
}

func listTools(ctx Context, scope string) error {
	names := availableToolNames(ctx.AppRoot, scope)
	if len(names) == 0 {
		fmt.Fprintf(ctx.Stdout, "no %s found\n", scope)
		return nil
	}
	for _, name := range names {
		fmt.Fprintln(ctx.Stdout, name)
	}
	return nil
}

func resolveTool(appRoot, scope, name string) (tool, bool) {
	for _, root := range toolRoots(appRoot, scope) {
		if resolved, ok := resolveToolInRoot(root, scope, name); ok {
			return resolved, true
		}
	}
	return tool{}, false
}

func availableToolNames(appRoot, scope string) []string {
	seen := map[string]struct{}{}
	for _, root := range toolRoots(appRoot, scope) {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := strings.TrimSpace(entry.Name())
			if name == "" || strings.HasPrefix(name, ".") {
				continue
			}
			if !entry.IsDir() && strings.HasSuffix(name, ".sh") {
				name = strings.TrimSuffix(name, ".sh")
			}
			if _, ok := resolveToolInRoot(root, scope, name); ok {
				seen[name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func toolRoots(appRoot, scope string) []string {
	candidates := []string{
		filepath.Join(appRoot, "bin", "gmcore", scope),
		filepath.Join(appRoot, "vendor", "gmcore", "framework", "bin", "gmcore", scope),
		filepath.Join(appRoot, "vendor", "gmcore", "system-sdk", "bin", "gmcore", scope),
	}
	for _, pattern := range []string{
		filepath.Join(appRoot, "vendor", "gmcore", "sdk", "*", "bin", "gmcore", scope),
		filepath.Join(appRoot, "vendor", "gmcore", "bundles", "*", "bin", "gmcore", scope),
	} {
		matches, _ := filepath.Glob(pattern)
		candidates = append(candidates, matches...)
	}
	return uniqueCleanPaths(candidates)
}

func resolveToolInRoot(root, scope, name string) (tool, bool) {
	for _, candidateName := range nameCandidates(name) {
		if resolved, ok := resolveManifestTool(root, scope, candidateName); ok {
			return resolved, true
		}
		for _, candidate := range []string{
			filepath.Join(root, candidateName),
			filepath.Join(root, candidateName+".sh"),
		} {
			info, err := os.Stat(candidate)
			if err == nil && isExecutableFile(info) {
				return tool{Name: name, Scope: scope, Path: candidate}, true
			}
		}
	}
	return tool{}, false
}

func resolveManifestTool(root, scope, name string) (tool, bool) {
	base := filepath.Join(root, name)
	for _, manifestName := range []string{"maker.yaml", "command.yaml", "tool.yaml", "manifest.yaml"} {
		manifest, err := loadToolManifest(filepath.Join(base, manifestName))
		if err != nil {
			continue
		}
		executable := strings.TrimSpace(manifest.Executable)
		if executable == "" {
			executable = "run"
		}
		target := filepath.Clean(filepath.Join(base, executable))
		if !strings.HasPrefix(target, filepath.Clean(base)+string(os.PathSeparator)) && target != filepath.Clean(base) {
			continue
		}
		info, err := os.Stat(target)
		if err == nil && isExecutableFile(info) {
			toolName := strings.TrimSpace(manifest.Name)
			if toolName == "" {
				toolName = name
			}
			return tool{Name: toolName, Scope: scope, Path: target, Usage: manifest.Usage, Description: manifest.Description}, true
		}
	}
	return tool{}, false
}

func isExecutableFile(info os.FileInfo) bool {
	return !info.IsDir() && info.Mode().Perm()&0o111 != 0
}

func loadToolManifest(path string) (toolManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return toolManifest{}, err
	}
	var manifest toolManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return toolManifest{}, err
	}
	return manifest, nil
}

func runTool(ctx Context, tool tool, args []string) error {
	if strings.TrimSpace(tool.Path) == "" {
		return fmt.Errorf("tool path cannot be empty")
	}
	command := exec.Command(tool.Path, args...)
	command.Dir = ctx.AppRoot
	command.Env = append(os.Environ(),
		"GMCORE_APP_ROOT="+ctx.AppRoot,
		"GMCORE_TOOL_SCOPE="+tool.Scope,
		"GMCORE_TOOL_NAME="+tool.Name,
	)
	command.Stdout = ctx.Stdout
	command.Stderr = ctx.Stderr
	command.Stdin = os.Stdin
	return command.Run()
}

func nameCandidates(name string) []string {
	lower := strings.ToLower(name)
	if lower == name {
		return []string{name}
	}
	return []string{name, lower}
}

func uniqueCleanPaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if path == "." {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
