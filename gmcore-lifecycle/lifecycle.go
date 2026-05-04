package gmcore_lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LifecycleEvent int

const (
	EventBoot        LifecycleEvent = iota
	EventReady
	EventShutdown
)

type AppLayout struct {
	Root         string
	ManifestRoot string
	BuildRoot    string
	EnvRoot      string
	Packaged     bool
}

func ResolveAppLayout(root string) (*AppLayout, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(absRoot, "app.yaml")
	_, err = os.Stat(manifestPath)
	packaged := os.IsNotExist(err)

	manifestRoot := absRoot
	buildRoot := absRoot

	if packaged {
		currentPath := filepath.Join(absRoot, "current")
		if _, err := os.Stat(filepath.Join(currentPath, "app.yaml")); err == nil {
			manifestRoot = currentPath
			buildRoot = currentPath
		}
	}

	layout := &AppLayout{
		Root:         absRoot,
		ManifestRoot: manifestRoot,
		BuildRoot:    buildRoot,
		EnvRoot:      absRoot,
		Packaged:     packaged,
	}
	return layout, nil
}

type LifecycleHook func(ctx context.Context) error

type LifecycleManager struct {
	hooks    map[LifecycleEvent][]LifecycleHook
	started  bool
	ready    bool
}

func NewManager() *LifecycleManager {
	return &LifecycleManager{
		hooks: make(map[LifecycleEvent][]LifecycleHook),
	}
}

func (m *LifecycleManager) AddHook(event LifecycleEvent, hook LifecycleHook) {
	m.hooks[event] = append(m.hooks[event], hook)
}

func (m *LifecycleManager) Boot(ctx context.Context) error {
	if m.started {
		return nil
	}
	for _, hook := range m.hooks[EventBoot] {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	m.started = true
	return nil
}

func (m *LifecycleManager) MarkReady(ctx context.Context) error {
	if !m.started {
		return errors.New("lifecycle: cannot mark ready before boot")
	}
	if m.ready {
		return nil
	}
	for _, hook := range m.hooks[EventReady] {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	m.ready = true
	return nil
}

func (m *LifecycleManager) Shutdown(ctx context.Context) error {
	if !m.started {
		return nil
	}
	for _, hook := range m.hooks[EventShutdown] {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	m.started = false
	m.ready = false
	return nil
}

func (m *LifecycleManager) IsStarted() bool   { return m.started }
func (m *LifecycleManager) IsReady() bool    { return m.ready }

type BootHook struct {
	priority int
	hook     LifecycleHook
}

func NewBootHook(priority int, hook LifecycleHook) *BootHook {
	return &BootHook{priority: priority, hook: hook}
}

type ShutdownHook struct {
	priority int
	hook     LifecycleHook
}

func NewShutdownHook(priority int, hook LifecycleHook) *ShutdownHook {
	return &ShutdownHook{priority: priority, hook: hook}
}

type Hooks struct {
	bootHooks     []*BootHook
	readyHooks    []*BootHook
	shutdownHooks []*ShutdownHook
}

func NewHooks() *Hooks {
	return &Hooks{
		bootHooks:     make([]*BootHook, 0),
		readyHooks:    make([]*BootHook, 0),
		shutdownHooks: make([]*ShutdownHook, 0),
	}
}

func (h *Hooks) AddBootHook(priority int, hook LifecycleHook) {
	h.bootHooks = append(h.bootHooks, NewBootHook(priority, hook))
}

func (h *Hooks) AddShutdownHook(priority int, hook LifecycleHook) {
	h.shutdownHooks = append(h.shutdownHooks, NewShutdownHook(priority, hook))
}

func (h *Hooks) ExecuteBoot(ctx context.Context) error {
	for _, bh := range h.bootHooks {
		if err := bh.hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hooks) ExecuteShutdown(ctx context.Context) error {
	for _, sh := range h.shutdownHooks {
		if err := sh.hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

type TimerHook struct {
	name     string
	interval time.Duration
	hook     LifecycleHook
}

func NewTimerHook(name string, interval time.Duration, hook LifecycleHook) *TimerHook {
	return &TimerHook{name: name, interval: interval, hook: hook}
}

func (t *TimerHook) Start(ctx context.Context, cancel func()) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	hookCtx := ctx
	for {
		select {
		case <-ticker.C:
			t.hook(hookCtx)
		case <-ctx.Done():
			return
		}
	}
}

type Paths struct {
	InstancePath string
	BinPath      string
}

func NewPaths(binPath string) *Paths {
	return &Paths{
		InstancePath: filepath.Join(filepath.Dir(binPath), "app.instance.json"),
		BinPath:      binPath,
	}
}

type InstallOptions struct {
	ArchivePath string
	TargetDir   string
}

func Install(opts InstallOptions) (string, string, error) {
	if opts.TargetDir == "" {
		return "", "", errors.New("target directory required")
	}
	return "", "", nil
}

func parseProcEnviron(data []byte) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(string(data), "\x00")
	for _, part := range parts {
		if idx := strings.Index(part, "="); idx != -1 {
			key := part[:idx]
			value := part[idx+1:]
			result[key] = value
		}
	}
	return result
}
