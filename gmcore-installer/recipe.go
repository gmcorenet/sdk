package gmcoreinstaller

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Recipe struct {
	Install   []Step `yaml:"install"`
	Reinstall []Step `yaml:"reinstall"`
	Remove    []Step `yaml:"remove"`
	Purge     []Step `yaml:"purge"`
	Uninstall []Step `yaml:"uninstall"`
}

type Step struct {
	Action               string            `yaml:"action"`
	From                 string            `yaml:"from"`
	To                   string            `yaml:"to"`
	Path                 string            `yaml:"path"`
	Content              string            `yaml:"content"`
	URL                  string            `yaml:"url"`
	SHA256               string            `yaml:"sha256"`
	Command              string            `yaml:"command"`
	Args                 []string          `yaml:"args"`
	Name                 string            `yaml:"name"`
	Manager              string            `yaml:"manager"`
	Optional             bool              `yaml:"optional"`
	Overwrite            bool              `yaml:"overwrite"`
	PromptOverwrite      bool              `yaml:"prompt_overwrite"`
	RequiresConfirmation bool              `yaml:"requires_confirmation"`
	Env                  map[string]string `yaml:"env"`
	StripComponents      int               `yaml:"strip_components"`
}

type Runner struct {
	SourceRoot      string
	TargetRoot      string
	Confirmed       bool
	AllowPrivileged bool
	HTTPClient      *http.Client
	Stdout          io.Writer
	Stderr          io.Writer
	Stdin           io.Reader
	DryRun          bool
	RollbackStack   []func() error
}

func LoadRecipe(path string) (Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, err
	}
	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return Recipe{}, err
	}
	return recipe, nil
}

func (r Runner) RunInstall(recipe Recipe) error {
	return r.RunSteps(recipe.Install)
}

func (r Runner) RunReinstall(recipe Recipe) error {
	steps := recipe.Reinstall
	if len(steps) == 0 {
		steps = recipe.Install
	}
	return r.RunSteps(steps)
}

func (r Runner) RunRemove(recipe Recipe) error {
	steps := recipe.Remove
	if len(steps) == 0 {
		steps = recipe.Uninstall
	}
	return r.RunSteps(steps)
}

func (r Runner) RunPurge(recipe Recipe) error {
	return r.RunSteps(recipe.Purge)
}

func (r Runner) RunSteps(steps []Step) error {
	if r.DryRun {
		for _, step := range steps {
			if err := r.dryRunStep(step); err != nil {
				return err
			}
		}
		return nil
	}
	for _, step := range steps {
		if err := r.RunStep(step); err != nil {
			if step.Optional {
				continue
			}
			r.Rollback()
			return err
		}
	}
	return nil
}

func (r Runner) dryRunStep(step Step) error {
	if step.RequiresConfirmation && !r.Confirmed {
		return fmt.Errorf("recipe action %q requires explicit confirmation", step.Action)
	}
	action := strings.TrimSpace(step.Action)
	if action == "" {
		return nil
	}
	if action == "os_package" || action == "apt_package" || action == "yum_package" || action == "dnf_package" || action == "apk_package" || action == "zypper_package" || action == "pacman_package" || action == "brew_package" {
		if !r.AllowPrivileged || os.Geteuid() != 0 {
			return fmt.Errorf("dry-run: os package action %q requires privileged installer mode", action)
		}
	}
	msg := fmt.Sprintf("[dry-run] Would execute: %s", action)
	if step.Name != "" {
		msg += fmt.Sprintf(" (%s)", step.Name)
	}
	fmt.Fprintln(r.Stdout, msg)
	return nil
}

func (r *Runner) Rollback() {
	if len(r.RollbackStack) == 0 {
		return
	}
	fmt.Fprintln(r.Stderr, "\nRolling back changes...")
	for i := len(r.RollbackStack) - 1; i >= 0; i-- {
		undo := r.RollbackStack[i]
		if err := undo(); err != nil {
			fmt.Fprintf(r.Stderr, "rollback error: %v\n", err)
		}
	}
	r.RollbackStack = nil
}

func (r *Runner) PushRollback(fn func() error) {
	r.RollbackStack = append(r.RollbackStack, fn)
}

func (r Runner) RunStep(step Step) error {
	if step.RequiresConfirmation && !r.Confirmed {
		return fmt.Errorf("recipe action %q requires explicit confirmation", step.Action)
	}
	switch strings.TrimSpace(step.Action) {
	case "":
		return nil
	case "copy_tree":
		source, err := safeJoin(r.SourceRoot, step.From)
		if err != nil {
			return err
		}
		target, err := safeJoin(r.TargetRoot, step.To)
		if err != nil {
			return err
		}
		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		return copyTree(source, target, step.Overwrite)
	case "copy_file":
		source, err := safeJoin(r.SourceRoot, step.From)
		if err != nil {
			return err
		}
		target, err := safeJoin(r.TargetRoot, step.To)
		if err != nil {
			return err
		}
		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		overwrite := step.Overwrite
		if !overwrite && step.PromptOverwrite && fileExists(target) {
			confirmed, err := r.confirm(fmt.Sprintf("%s already exists. Overwrite?", target), false)
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}
			overwrite = true
		}
		return copyFile(source, target, overwrite)
	case "ensure_dir":
		target, err := safeJoin(r.TargetRoot, step.Path)
		if err != nil {
			return err
		}
		return os.MkdirAll(target, 0o755)
	case "write_file":
		target, err := safeJoin(r.TargetRoot, step.Path)
		if err != nil {
			return err
		}
		if !step.Overwrite && fileExists(target) {
			return fmt.Errorf("%s already exists", target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(step.Content), 0o644)
	case "remove_tree":
		target, err := safeJoin(r.TargetRoot, step.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(target)
	case "remove_file":
		target, err := safeJoin(r.TargetRoot, step.Path)
		if err != nil {
			return err
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	case "download_file":
		return r.downloadFile(step)
	case "download_archive":
		return r.downloadArchive(step)
	case "extract_archive":
		return r.extractArchive(step)
	case "run":
		return r.runCommand(step)
	case "os_package", "apt_package", "yum_package", "dnf_package", "apk_package", "zypper_package", "pacman_package", "brew_package":
		return r.installOSPackage(step)
	default:
		return fmt.Errorf("unknown installer recipe action %q", step.Action)
	}
}

func (r Runner) confirm(message string, fallback bool) (bool, error) {
	if r.Stdin == nil {
		return fallback, nil
	}
	prompt := " [y/N] "
	if fallback {
		prompt = " [Y/n] "
	}
	if r.Stdout != nil {
		fmt.Fprint(r.Stdout, message+prompt)
	}
	var answer string
	if _, err := fmt.Fscanln(r.Stdin, &answer); err != nil {
		if errors.Is(err, io.EOF) {
			return fallback, nil
		}
		return fallback, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	switch answer {
	case "y", "yes", "s", "si", "sí":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return fallback, nil
	}
}

func (r Runner) downloadFile(step Step) error {
	if strings.TrimSpace(step.URL) == "" {
		return errors.New("download_file requires url")
	}
	target, err := safeJoin(r.TargetRoot, step.To)
	if err != nil {
		return err
	}
	if !step.Overwrite && fileExists(target) {
		return fmt.Errorf("%s already exists", target)
	}
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Get(step.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download %s: HTTP %d", step.URL, resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if strings.TrimSpace(step.SHA256) == "" {
		_, err = io.Copy(file, resp.Body)
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(step.SHA256)) {
		return fmt.Errorf("download %s sha256 mismatch", step.URL)
	}
	return nil
}

func (r Runner) downloadArchive(step Step) error {
	if strings.TrimSpace(step.URL) == "" {
		return errors.New("download_archive requires url")
	}
	temp, err := os.CreateTemp("", "gmcore-installer-archive-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	_ = temp.Close()
	defer os.Remove(tempPath)
	downloadStep := step
	downloadStep.To = tempPath
	if err := r.downloadFileToAbsolute(downloadStep, tempPath); err != nil {
		return err
	}
	extractStep := step
	extractStep.From = tempPath
	return r.extractArchive(extractStep)
}

func (r Runner) downloadFileToAbsolute(step Step, target string) error {
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Get(step.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download %s: HTTP %d", step.URL, resp.StatusCode)
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if strings.TrimSpace(step.SHA256) == "" {
		_, err = io.Copy(file, resp.Body)
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(step.SHA256)) {
		return fmt.Errorf("download %s sha256 mismatch", step.URL)
	}
	return nil
}

func (r Runner) extractArchive(step Step) error {
	archivePath := strings.TrimSpace(step.From)
	if archivePath == "" && strings.TrimSpace(step.Path) != "" {
		archivePath = strings.TrimSpace(step.Path)
	}
	if archivePath == "" {
		return errors.New("extract_archive requires from or path")
	}
	if !filepath.IsAbs(archivePath) {
		var err error
		archivePath, err = safeJoin(r.SourceRoot, archivePath)
		if err != nil {
			return err
		}
	}
	destination, err := safeJoin(r.TargetRoot, firstNonEmpty(step.To, "."))
	if err != nil {
		return err
	}
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, destination, step.StripComponents, step.Overwrite)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, destination, step.StripComponents, step.Overwrite)
	default:
		return fmt.Errorf("unsupported archive type %q", archivePath)
	}
}

func (r Runner) runCommand(step Step) error {
	commandName := strings.TrimSpace(step.Command)
	if commandName == "" && strings.TrimSpace(step.Path) != "" {
		path, err := safeJoin(r.SourceRoot, step.Path)
		if err != nil {
			return err
		}
		commandName = path
	}
	if commandName == "" {
		return errors.New("run requires command or path")
	}
	if filepath.IsAbs(commandName) || strings.Contains(commandName, string(os.PathSeparator)) {
		if err := validateExecutable(commandName); err != nil {
			return err
		}
	} else if _, err := exec.LookPath(commandName); err != nil {
		return fmt.Errorf("command %q not found in PATH", commandName)
	}
	cmd := exec.Command(commandName, step.Args...)
	cmd.Dir = r.TargetRoot
	cmd.Env = os.Environ()
	for key, value := range step.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Stdin = r.Stdin
	return cmd.Run()
}

func (r Runner) installOSPackage(step Step) error {
	if !r.AllowPrivileged {
		return fmt.Errorf("os package action %q requires privileged installer mode; rerun with privileged installer permissions or remove this step", firstNonEmpty(step.Name, step.Path))
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("os package action %q requires root privileges; current user is not root, stopping before running the package manager", firstNonEmpty(step.Name, step.Path))
	}
	name := strings.TrimSpace(step.Name)
	if name == "" {
		return errors.New("os_package requires name")
	}
	binary, args, err := packageInstallCommand(step.Action, step.Manager, name)
	if err != nil {
		return err
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Stdin = r.Stdin
	return cmd.Run()
}

func validateExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("executable %q is not accessible: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("executable %q is a directory", path)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("executable %q does not have execute permission; stopping before running it", path)
	}
	return nil
}

func packageInstallCommand(action, manager, name string) (string, []string, error) {
	manager = normalizePackageManager(action, manager)
	if manager == "auto" || manager == "" {
		manager = detectPackageManager()
	}
	switch manager {
	case "apt", "apt-get":
		return "apt-get", []string{"install", "-y", name}, nil
	case "dnf":
		return "dnf", []string{"install", "-y", name}, nil
	case "yum":
		return "yum", []string{"install", "-y", name}, nil
	case "apk":
		return "apk", []string{"add", name}, nil
	case "zypper":
		return "zypper", []string{"--non-interactive", "install", name}, nil
	case "pacman":
		return "pacman", []string{"-S", "--noconfirm", name}, nil
	case "brew", "homebrew":
		return "brew", []string{"install", name}, nil
	default:
		return "", nil, fmt.Errorf("unsupported os package manager %q", manager)
	}
}

func normalizePackageManager(action, manager string) string {
	manager = strings.TrimSpace(strings.ToLower(manager))
	if manager != "" {
		return manager
	}
	switch strings.TrimSpace(strings.ToLower(action)) {
	case "apt_package":
		return "apt"
	case "yum_package":
		return "yum"
	case "dnf_package":
		return "dnf"
	case "apk_package":
		return "apk"
	case "zypper_package":
		return "zypper"
	case "pacman_package":
		return "pacman"
	case "brew_package":
		return "brew"
	default:
		return "auto"
	}
}

func detectPackageManager() string {
	for _, candidate := range []string{"apt-get", "dnf", "yum", "apk", "zypper", "pacman", "brew"} {
		if _, err := exec.LookPath(candidate); err == nil {
			if candidate == "apt-get" {
				return "apt"
			}
			return candidate
		}
	}
	return "auto"
}

func safeJoin(base, relative string) (string, error) {
	base = filepath.Clean(strings.TrimSpace(base))
	if base == "" {
		return "", errors.New("missing base path")
	}
	relative = strings.TrimSpace(relative)
	relative = filepath.FromSlash(relative)
	target := filepath.Clean(filepath.Join(base, relative))
	if target == base || target == base+string(os.PathSeparator) {
		return target, nil
	}
	sep := string(os.PathSeparator)
	if !strings.HasPrefix(target, base+sep) {
		return "", fmt.Errorf("path %q escapes base %q", relative, base)
	}
	return target, nil
}

func copyTree(source, target string, overwrite bool) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", source)
	}
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, rel)
		if rel == "." {
			return os.MkdirAll(target, 0o755)
		}
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyFile(path, destination, overwrite)
	})
}

func copyFile(source, target string, overwrite bool) error {
	if !overwrite && fileExists(target) {
		return fmt.Errorf("%s already exists", target)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func extractTarGz(archivePath, destination string, stripComponents int, overwrite bool) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, ok, err := archiveTargetPath(destination, header.Name, stripComponents)
		if err != nil || !ok {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if !overwrite && fileExists(target) {
				return fmt.Errorf("%s already exists", target)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, reader); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func extractZip(archivePath, destination string, stripComponents int, overwrite bool) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		target, ok, err := archiveTargetPath(destination, file.Name, stripComponents)
		if err != nil || !ok {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if !overwrite && fileExists(target) {
			return fmt.Errorf("%s already exists", target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode()&0o777)
		if err != nil {
			_ = input.Close()
			return err
		}
		if _, err := io.Copy(out, input); err != nil {
			_ = input.Close()
			_ = out.Close()
			return err
		}
		if err := input.Close(); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func archiveTargetPath(destination, name string, stripComponents int) (string, bool, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" {
		return "", false, nil
	}
	parts := strings.Split(name, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		cleaned = append(cleaned, part)
	}
	if stripComponents > 0 {
		if len(cleaned) <= stripComponents {
			return "", false, nil
		}
		cleaned = cleaned[stripComponents:]
	}
	if len(cleaned) == 0 {
		return "", false, nil
	}
	relative := filepath.Join(cleaned...)
	target, err := safeJoin(destination, relative)
	if err != nil {
		return "", false, err
	}
	return target, true, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
