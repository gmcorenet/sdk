package gmcore_console

import (
	"io"
	"sort"
	"strings"
)

type Context struct {
	AppRoot string
	Stdout  io.Writer
	Stderr  io.Writer
}

type Handler func(Context, []string) error

type Command struct {
	Name        string
	Description string
	Usage       string
	Run         Handler
}

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{commands: map[string]Command{}}
}

func (registry *Registry) Register(command Command) {
	name := strings.TrimSpace(command.Name)
	if name == "" || command.Run == nil {
		return
	}
	command.Name = name
	registry.commands[name] = command
}

func (registry *Registry) Resolve(name string) (Command, bool) {
	command, ok := registry.commands[strings.TrimSpace(name)]
	return command, ok
}

func (registry *Registry) Commands() []Command {
	commands := make([]Command, 0, len(registry.commands))
	for _, command := range registry.commands {
		commands = append(commands, command)
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})
	return commands
}
