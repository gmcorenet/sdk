package gmcoreconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Env        map[string]string
	Parameters map[string]string
	Strict     bool
}

func LoadYAML(path string, out interface{}, options Options) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rendered, err := ResolveString(string(data), withFileParameters(data, options))
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(rendered), out)
}

func ResolveString(content string, options Options) (string, error) {
	resolved, err := resolveToken(content, "%env(", ")%", options.Strict, func(key string) (string, bool) {
		value, ok := options.Env[strings.TrimSpace(key)]
		return value, ok
	})
	if err != nil {
		return "", err
	}
	resolved, err = resolveToken(resolved, "%parameter.", "%", options.Strict, func(key string) (string, bool) {
		value, ok := options.Parameters[strings.TrimSpace(key)]
		return value, ok
	})
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func ParseEnvFile(path string) map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}

func LoadEnvFiles(paths ...string) map[string]string {
	out := map[string]string{}
	for _, path := range paths {
		for key, value := range ParseEnvFile(path) {
			out[key] = value
		}
	}
	return out
}

func LoadAppEnv(appPath string) map[string]string {
	appName := filepath.Base(filepath.Clean(appPath))
	candidates := []string{
		filepath.Join(appPath, ".env"),
		filepath.Join(appPath, ".env.local"),
		filepath.Join(appPath, "config", appName+".env"),
	}
	if _, err := os.Stat(candidates[0]); err != nil {
		candidates = append(candidates, filepath.Join(appPath, ".env.example"))
	}
	values := LoadEnvFiles(candidates...)
	appPrefix := strings.ToUpper(strings.ReplaceAll(appName, "-", "_")) + "_"
	prefixes := []string{appPrefix, "GMCORE_" + appPrefix}
	for key, value := range values {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				values["APP_"+strings.TrimPrefix(key, prefix)] = value
				break
			}
		}
	}
	return values
}

func EnvList(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func withFileParameters(data []byte, options Options) Options {
	root := struct {
		Parameters map[string]interface{} `yaml:"parameters"`
	}{}
	if err := yaml.Unmarshal(data, &root); err != nil || len(root.Parameters) == 0 {
		return options
	}
	merged := map[string]string{}
	for key, value := range options.Parameters {
		merged[key] = value
	}
	for key, value := range root.Parameters {
		raw := fmt.Sprint(value)
		if resolved, err := ResolveString(raw, Options{Env: options.Env, Parameters: merged, Strict: options.Strict}); err == nil {
			raw = resolved
		}
		merged[strings.TrimSpace(key)] = raw
	}
	options.Parameters = merged
	return options
}

func resolveToken(content, prefix, suffix string, strict bool, lookup func(string) (string, bool)) (string, error) {
	for {
		start := strings.Index(content, prefix)
		if start < 0 {
			return content, nil
		}
		end := strings.Index(content[start+len(prefix):], suffix)
		if end < 0 {
			return content, nil
		}
		tokenEnd := start + len(prefix) + end
		key := strings.TrimSpace(content[start+len(prefix) : tokenEnd])
		value, ok := lookup(key)
		if !ok && strict {
			return "", fmt.Errorf("missing config placeholder %s%s%s", prefix, key, suffix)
		}
		content = content[:start] + value + content[tokenEnd+len(suffix):]
	}
}
