package gmcore_options_resolver

// Package gmcore_options_resolver provides command-line argument parsing with support
// for flags, options, and positional arguments.
//
// Examples:
//
//	opts := New().
//		AddString("name", "Your name", "John").
//		AddInt("port", "Port number", 8080).
//		AddFlag("verbose", "v", "Verbose mode")
//
//	err := opts.Parse(os.Args[1:])
//	if opts.IsSet("name") {
//		val, _ := opts.GetString("name")
//		fmt.Println("Name:", val)
//	}

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Option struct {
	Name        string
	Short      string
	Description string
	Default    interface{}
	Value      interface{}
	Required   bool
}

type Options struct {
	options    map[string]*Option
	positional []string
	resolved   bool
	strictMode bool
}

func New() *Options {
	return &Options{
		options:  make(map[string]*Option),
		resolved: false,
	}
}

func (o *Options) AddString(name, description string, defaultVal string) *Options {
	o.options[name] = &Option{
		Name:        name,
		Description: description,
		Default:     defaultVal,
		Value:       defaultVal,
	}
	return o
}

func (o *Options) AddInt(name, description string, defaultVal int) *Options {
	o.options[name] = &Option{
		Name:        name,
		Description: description,
		Default:     defaultVal,
		Value:       defaultVal,
	}
	return o
}

func (o *Options) AddBool(name, description string, defaultVal bool) *Options {
	o.options[name] = &Option{
		Name:        name,
		Description: description,
		Default:     defaultVal,
		Value:       defaultVal,
	}
	return o
}

func (o *Options) AddStringRequired(name, description string) *Options {
	o.options[name] = &Option{
		Name:        name,
		Description: description,
		Required:    true,
	}
	return o
}

func (o *Options) AddFlag(name, short, description string) *Options {
	o.options[name] = &Option{
		Name:        name,
		Short:       short,
		Description: description,
		Default:     false,
		Value:       false,
	}
	return o
}

func (o *Options) SetStrictMode(enabled bool) *Options {
	o.strictMode = enabled
	return o
}

func (o *Options) findByShort(short string) *Option {
	for _, opt := range o.options {
		if opt.Short == short {
			return opt
		}
	}
	return nil
}

func (o *Options) Parse(args []string) error {
	o.resolved = true

	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			name := parts[0]

			opt, ok := o.options[name]
			if !ok {
				if o.strictMode {
					return fmt.Errorf("unknown option: --%s", name)
				}
				continue
			}

			if _, isBool := opt.Default.(bool); isBool && len(parts) == 1 {
				opt.Value = true
			} else if len(parts) == 2 {
				if err := o.setValue(opt, parts[1]); err != nil {
					return fmt.Errorf("invalid value for --%s: %w", name, err)
				}
			} else {
				i++
				if i >= len(args) {
					return fmt.Errorf("option --%s requires a value", name)
				}
				if err := o.setValue(opt, args[i]); err != nil {
					return fmt.Errorf("invalid value for --%s: %w", name, err)
				}
			}
		} else if strings.HasPrefix(arg, "-") {
			name := arg[1:]

			opt, ok := o.options[name]
			if !ok {
				opt = o.findByShort(name)
			}
			if opt == nil {
				if o.strictMode {
					return fmt.Errorf("unknown option: -%s", name)
				}
				continue
			}

			if _, isBool := opt.Default.(bool); isBool {
				opt.Value = true
			} else {
				i++
				if i >= len(args) {
					return fmt.Errorf("option -%s requires a value", name)
				}
				if err := o.setValue(opt, args[i]); err != nil {
					return fmt.Errorf("invalid value for -%s: %w", name, err)
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}

	for name, opt := range o.options {
		if opt.Required && opt.Value == nil {
			return fmt.Errorf("required option --%s is missing", name)
		}
	}

	o.positional = positional
	return nil
}

func (o *Options) setValue(opt *Option, value string) error {
	switch opt.Default.(type) {
	case string:
		opt.Value = value
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		opt.Value = v
	case bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		opt.Value = v
	default:
		opt.Value = value
	}
	return nil
}

func (o *Options) GetString(name string) (string, error) {
	opt, ok := o.options[name]
	if !ok {
		return "", fmt.Errorf("option --%s not found", name)
	}
	if str, ok := opt.Value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("option --%s is not a string", name)
}

func (o *Options) GetInt(name string) (int, error) {
	opt, ok := o.options[name]
	if !ok {
		return 0, fmt.Errorf("option --%s not found", name)
	}
	if v, ok := opt.Value.(int); ok {
		return v, nil
	}
	return 0, fmt.Errorf("option --%s is not an int", name)
}

func (o *Options) GetBool(name string) (bool, error) {
	opt, ok := o.options[name]
	if !ok {
		return false, fmt.Errorf("option --%s not found", name)
	}
	if v, ok := opt.Value.(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("option --%s is not a bool", name)
}

func (o *Options) IsSet(name string) bool {
	opt, ok := o.options[name]
	if !ok {
		return false
	}
	return opt.Value != nil && opt.Value != opt.Default
}

func (o *Options) GetPositional() []string {
	return o.positional
}

func (o *Options) GetPositionalAt(index int) (string, error) {
	if index < 0 || index >= len(o.positional) {
		return "", fmt.Errorf("positional index %d out of bounds", index)
	}
	return o.positional[index], nil
}

func (o *Options) ShowHelp() {
	fmt.Println("Options:")
	for _, opt := range o.options {
		defaultStr := ""
		if opt.Default != nil {
			defaultStr = fmt.Sprintf(" (default: %v)", opt.Default)
		}
		requiredStr := ""
		if opt.Required {
			requiredStr = " [required]"
		}
		if opt.Short != "" {
			fmt.Printf("  -%s, --%s%s%s%s\n", opt.Short, opt.Name, defaultStr, requiredStr, opt.Description)
		} else {
			fmt.Printf("  --%s%s%s%s\n", opt.Name, defaultStr, requiredStr, opt.Description)
		}
	}
}

func (o *Options) ShowUsage(progName string) {
	fmt.Printf("Usage: %s [options]", progName)
	if len(o.positional) > 0 {
		fmt.Printf(" %s", strings.Join(o.positional, " "))
	}
	fmt.Println()
	o.ShowHelp()
}

func ParseEnv(prefix string) map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}

		if prefix != "" {
			key = strings.TrimPrefix(key, prefix)
		}
		env[key] = value
	}
	return env
}
