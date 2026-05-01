package gmcorelifecycle

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type ReleaseManifest struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	InstallTarget  string   `json:"install_target"`
	PersistentDirs []string `json:"persistent_dirs"`
}

type AppManifest struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Runtime struct {
		Mode        string `yaml:"mode"`
		Transport   string `yaml:"transport"`
		Entrypoint  string `yaml:"entrypoint"`
		Healthcheck string `yaml:"healthcheck"`
	} `yaml:"runtime"`
}

type AppLayout struct {
	InstallRoot  string
	ManifestRoot string
	ManifestPath string
	BuildRoot    string
	EnvRoot      string
	Packaged     bool
}

type RuntimePaths struct {
	Root            string
	CurrentRoot     string
	RunDir          string
	LogDir          string
	DataDir         string
	PIDPath         string
	InstancePath    string
	HTTPSocketPath  string
	ControlSockPath string
	StdoutLogPath   string
	StderrLogPath   string
}

type AppStatus struct {
	Name            string
	Root            string
	RuntimeMode     string
	EffectiveUser   string
	EffectiveGroup  string
	PID             int
	Running         bool
	Ready           bool
	Entrypoint      string
	PIDPath         string
	InstancePath    string
	HTTPSocketPath  string
	ControlSockPath string
	RunDir          string
	LogDir          string
	DataDir         string
	HTTPReady       bool
	ControlReady    bool
}

type InstallOptions struct {
	ArchivePath string
	Reinstall   bool
	TargetPath  string
}

type ExecutionIdentity struct {
	User     string
	Group    string
	UID      int
	GID      int
	GroupIDs []int
}

type instanceMetadata struct {
	Name       string `json:"name"`
	PID        int    `json:"pid"`
	Root       string `json:"root"`
	Entrypoint string `json:"entrypoint"`
	StartedAt  string `json:"started_at"`
}

const sharedRuntimeGroup = "gmcore"

var (
	managedRoots = []string{
		"/opt/gmcore",
		"/var/lib/gmcore",
		"C:\\ProgramData\\gmcore",
	}
)

func getManagedRoots() []string {
	if runtime.GOOS == "windows" {
		return []string{"C:\\ProgramData\\gmcore"}
	}
	return managedRoots
}

func ReadManifest(archivePath string) (ReleaseManifest, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return ReleaseManifest{}, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return ReleaseManifest{}, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ReleaseManifest{}, err
		}
		if filepath.ToSlash(header.Name) != "release.json" {
			continue
		}
		var manifest ReleaseManifest
		if err := json.NewDecoder(tr).Decode(&manifest); err != nil {
			return ReleaseManifest{}, err
		}
		if manifest.Name == "" || manifest.Version == "" {
			return ReleaseManifest{}, fmt.Errorf("incomplete release manifest in %s", archivePath)
		}
		return manifest, nil
	}

	return ReleaseManifest{}, fmt.Errorf("release.json not found in %s", archivePath)
}

func Install(opts InstallOptions) (ReleaseManifest, string, error) {
	manifest, err := ReadManifest(opts.ArchivePath)
	if err != nil {
		return ReleaseManifest{}, "", err
	}

	target := opts.TargetPath
	if target == "" {
		target = manifest.InstallTarget
	}
	if target == "" {
		return ReleaseManifest{}, "", errors.New("missing install target")
	}

	releaseDir := filepath.Join(target, "releases", manifest.Version)
	if _, err := os.Stat(releaseDir); err == nil && !opts.Reinstall {
		return ReleaseManifest{}, "", fmt.Errorf("release already exists at %s", releaseDir)
	}

	if opts.Reinstall {
		_ = os.RemoveAll(releaseDir)
	}

	if err := os.MkdirAll(filepath.Join(target, "releases"), 0o755); err != nil {
		return ReleaseManifest{}, "", err
	}

	if err := extractArchive(opts.ArchivePath, releaseDir); err != nil {
		return ReleaseManifest{}, "", err
	}

	if err := ensurePersistentDirs(target, releaseDir, manifest.PersistentDirs); err != nil {
		return ReleaseManifest{}, "", err
	}
	if err := ensureRuntimeEnvFile(target, releaseDir); err != nil {
		return ReleaseManifest{}, "", err
	}

	current := filepath.Join(target, "current")
	_ = os.Remove(current)
	if err := os.Symlink(releaseDir, current); err != nil {
		return ReleaseManifest{}, "", err
	}
	if err := ensureManagedOperatorIdentity(target); err != nil {
		return ReleaseManifest{}, "", err
	}
	if err := normalizePackagedRuntimeSecurity(target, releaseDir, manifest.Name); err != nil {
		return ReleaseManifest{}, "", err
	}
	if err := reconcileGlobalCLI(manifest.Name, target); err != nil {
		return ReleaseManifest{}, "", err
	}

	return manifest, target, nil
}

func PrepareManagedApp(root, appName string) error {
	root = filepath.Clean(strings.TrimSpace(root))
	appName = strings.TrimSpace(appName)
	if os.Geteuid() != 0 || !isManagedInstall(root) || appName == "" {
		return nil
	}
	if err := ensureManagedOperatorIdentity(root); err != nil {
		return err
	}
	identity, err := ensureManagedRuntimeIdentity(appName, root)
	if err != nil {
		return err
	}
	for _, dir := range []string{
		filepath.Join(root, "data"),
		filepath.Join(root, "var"),
		filepath.Join(root, "var", "run"),
		filepath.Join(root, "var", "log"),
		filepath.Join(root, "current"),
		filepath.Join(root, "current", "bin"),
	} {
		if err := os.MkdirAll(dir, 0o2750); err != nil {
			return err
		}
		if err := os.Chown(dir, identity.UID, identity.GID); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o2750); err != nil {
			return err
		}
	}
	for _, path := range []string{
		filepath.Join(root, ".env"),
		filepath.Join(root, ".env.local"),
		filepath.Join(root, ".env.example"),
		filepath.Join(root, "config"),
		filepath.Join(root, "config", "keys"),
	} {
		if err := normalizeSecurityPath(path, identity, false); err != nil {
			return err
		}
	}
	return nil
}

func Remove(target string) error {
	if err := removeGlobalCLIIfMatches(target); err != nil {
		return err
	}
	current := filepath.Join(target, "current")
	_ = os.Remove(current)
	return os.RemoveAll(filepath.Join(target, "releases"))
}

func Purge(target string) error {
	if err := removeGlobalCLIIfMatches(target); err != nil {
		return err
	}
	return os.RemoveAll(target)
}

func LoadAppManifest(root string) (AppManifest, error) {
	layout, err := ResolveAppLayout(root)
	if err != nil {
		return AppManifest{}, err
	}
	data, err := os.ReadFile(layout.ManifestPath)
	if err != nil {
		return AppManifest{}, err
	}
	var manifest AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return AppManifest{}, err
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return AppManifest{}, errors.New("app.yaml missing name")
	}
	if strings.TrimSpace(manifest.Runtime.Entrypoint) == "" {
		return AppManifest{}, errors.New("app.yaml missing runtime.entrypoint")
	}
	return manifest, nil
}

func ResolveAppLayout(root string) (AppLayout, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	currentRoot := filepath.Join(root, "current")
	currentManifest := filepath.Join(currentRoot, "app.yaml")
	layout := AppLayout{
		InstallRoot:  root,
		ManifestRoot: root,
		ManifestPath: filepath.Join(root, "app.yaml"),
		BuildRoot:    root,
		EnvRoot:      root,
	}
	if fileExists(currentManifest) {
		layout.ManifestRoot = currentRoot
		layout.ManifestPath = currentManifest
		layout.EnvRoot = root
		layout.Packaged = true
		if fileExists(filepath.Join(currentRoot, "go.mod")) {
			layout.BuildRoot = currentRoot
		}
		return layout, nil
	}
	if fileExists(layout.ManifestPath) {
		if fileExists(filepath.Join(root, "go.mod")) {
			layout.BuildRoot = root
		}
		return layout, nil
	}
	return AppLayout{}, fmt.Errorf("app.yaml not found in %s or %s", filepath.Join(root, "app.yaml"), currentManifest)
}

func Paths(root string) RuntimePaths {
	root = filepath.Clean(strings.TrimSpace(root))
	runDir := filepath.Join(root, "var", "run")
	logDir := filepath.Join(root, "var", "log")
	return RuntimePaths{
		Root:            root,
		CurrentRoot:     filepath.Join(root, "current"),
		RunDir:          runDir,
		LogDir:          logDir,
		DataDir:         filepath.Join(root, "data"),
		PIDPath:         filepath.Join(runDir, "app.pid"),
		InstancePath:    filepath.Join(runDir, "app.instance.json"),
		HTTPSocketPath:  filepath.Join(runDir, "http.sock"),
		ControlSockPath: filepath.Join(runDir, "control.sock"),
		StdoutLogPath:   filepath.Join(logDir, "stdout.log"),
		StderrLogPath:   filepath.Join(logDir, "stderr.log"),
	}
}

func Status(root string) (AppStatus, error) {
	manifest, err := LoadAppManifest(root)
	if err != nil {
		return AppStatus{}, err
	}
	paths := Paths(root)
	identity, err := resolveExecutionIdentity(paths.Root, manifest.Name)
	if err != nil {
		identity = currentExecutionIdentity()
	}
	status := AppStatus{
		Name:            manifest.Name,
		Root:            filepath.Clean(root),
		RuntimeMode:     runtimeMode(manifest),
		EffectiveUser:   fallbackIdentityValue(identity.User, strconv.Itoa(identity.UID)),
		EffectiveGroup:  fallbackIdentityValue(identity.Group, strconv.Itoa(identity.GID)),
		Entrypoint:      resolveEntrypoint(paths.Root, manifest.Runtime.Entrypoint),
		PIDPath:         paths.PIDPath,
		InstancePath:    paths.InstancePath,
		HTTPSocketPath:  paths.HTTPSocketPath,
		ControlSockPath: paths.ControlSockPath,
		RunDir:          paths.RunDir,
		LogDir:          paths.LogDir,
		DataDir:         paths.DataDir,
	}

	pid, running, err := readRunningPID(paths.PIDPath)
	if err != nil {
		return status, err
	}
	status.PID = pid
	status.Running = running
	status.HTTPReady = socketExists(paths.HTTPSocketPath)
	status.ControlReady = socketExists(paths.ControlSockPath)
	status.Ready = computeReady(status)
	return status, nil
}

func Start(root string, extraEnv []string) (AppStatus, error) {
	layout, err := ResolveAppLayout(root)
	if err != nil {
		return AppStatus{}, err
	}
	manifest, err := LoadAppManifest(root)
	if err != nil {
		return AppStatus{}, err
	}
	paths := Paths(layout.InstallRoot)
	if err := os.MkdirAll(paths.RunDir, 0o755); err != nil {
		return AppStatus{}, err
	}
	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return AppStatus{}, err
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return AppStatus{}, err
	}
	identity, err := resolveExecutionIdentity(layout.InstallRoot, manifest.Name)
	if err != nil {
		return AppStatus{}, err
	}
	if err := PrepareManagedApp(layout.InstallRoot, manifest.Name); err != nil {
		return AppStatus{}, err
	}
	identity, err = resolveExecutionIdentity(layout.InstallRoot, manifest.Name)
	if err != nil {
		return AppStatus{}, err
	}

	current, err := Status(root)
	if err == nil && current.Running {
		return current, fmt.Errorf("app %s is already running with pid %d", current.Name, current.PID)
	}
	entrypoint := resolveEntrypoint(layout.InstallRoot, manifest.Runtime.Entrypoint)
	if err := stopDuplicateManagedProcesses(paths, 3*time.Second); err != nil {
		return AppStatus{}, err
	}
	_ = os.Remove(paths.PIDPath)
	_ = os.Remove(paths.InstancePath)
	removeSocket(paths.HTTPSocketPath)
	removeSocket(paths.ControlSockPath)
	if _, err := os.Stat(entrypoint); err != nil {
		return AppStatus{}, fmt.Errorf("entrypoint %s not found", entrypoint)
	}
	if err := grantEdgeEntrypointCapability(manifest, entrypoint); err != nil {
		return AppStatus{}, err
	}
	if err := validateStartupPermissions(entrypoint, paths, identity); err != nil {
		return AppStatus{}, err
	}

	stdoutFile, err := os.OpenFile(paths.StdoutLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return AppStatus{}, err
	}
	defer stdoutFile.Close()

	stderrFile, err := os.OpenFile(paths.StderrLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return AppStatus{}, err
	}
	defer stderrFile.Close()

	cmd := exec.Command(entrypoint)
	cmd.Dir = layout.InstallRoot
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if shouldDropPrivileges(layout, identity) {
		cmd.SysProcAttr.Credential = &syscall.Credential{
			Uid:    uint32(identity.UID),
			Gid:    uint32(identity.GID),
			Groups: uint32Groups(identity.GroupIDs),
		}
	}
	cmd.Env = os.Environ()
	for _, item := range extraEnv {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			cmd.Env = withEnv(cmd.Env, parts[0], parts[1])
		}
	}
	cmd.Env = withEnv(cmd.Env, "APP_DATA_DIR", paths.DataDir)
	cmd.Env = withEnv(cmd.Env, "APP_HTTP_SOCKET", paths.HTTPSocketPath)
	cmd.Env = withEnv(cmd.Env, "APP_CONTROL_SOCKET", paths.ControlSockPath)
	cmd.Env = withEnv(cmd.Env, "APP_PID_FILE", paths.PIDPath)
	cmd.Env = withEnv(cmd.Env, "GMCORE_MANAGED_LAUNCH", "1")

	if err := cmd.Start(); err != nil {
		return AppStatus{}, err
	}
	if err := os.WriteFile(paths.PIDPath, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		return AppStatus{}, err
	}
	if err := writeInstanceMetadata(paths.InstancePath, instanceMetadata{
		Name:       manifest.Name,
		PID:        cmd.Process.Pid,
		Root:       layout.InstallRoot,
		Entrypoint: entrypoint,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		_ = cmd.Process.Kill()
		_ = os.Remove(paths.PIDPath)
		return AppStatus{}, err
	}
	if shouldDropPrivileges(layout, identity) {
		_ = os.Chown(paths.PIDPath, identity.UID, identity.GID)
		_ = os.Chmod(paths.PIDPath, 0o640)
		_ = os.Chown(paths.InstancePath, identity.UID, identity.GID)
		_ = os.Chmod(paths.InstancePath, 0o640)
	}

	time.Sleep(200 * time.Millisecond)
	if !pidRunning(cmd.Process.Pid) {
		return AppStatus{}, fmt.Errorf("app exited during startup: %s", tailFile(paths.StderrLogPath, 8))
	}
	return Status(layout.InstallRoot)
}

func Stop(root string, timeout time.Duration) (AppStatus, error) {
	status, err := Status(root)
	if err != nil {
		return AppStatus{}, err
	}
	if !status.Running {
		_ = os.Remove(status.PIDPath)
		_ = os.Remove(status.InstancePath)
		removeSocket(status.HTTPSocketPath)
		removeSocket(status.ControlSockPath)
		return status, nil
	}

	if err := syscall.Kill(status.PID, syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return status, err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !pidRunning(status.PID) {
			_ = os.Remove(status.PIDPath)
			_ = os.Remove(status.InstancePath)
			removeSocket(status.HTTPSocketPath)
			removeSocket(status.ControlSockPath)
			return Status(root)
		}
		time.Sleep(150 * time.Millisecond)
	}
	if err := syscall.Kill(status.PID, syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return status, err
	}
	time.Sleep(100 * time.Millisecond)
	_ = os.Remove(status.PIDPath)
	_ = os.Remove(status.InstancePath)
	removeSocket(status.HTTPSocketPath)
	removeSocket(status.ControlSockPath)
	return Status(root)
}

func Reload(root string) (AppStatus, error) {
	status, err := Status(root)
	if err != nil {
		return AppStatus{}, err
	}
	if !status.Running {
		return status, fmt.Errorf("app %s is not running", status.Name)
	}
	if err := syscall.Kill(status.PID, syscall.SIGHUP); err != nil {
		return status, err
	}
	time.Sleep(300 * time.Millisecond)
	return Status(root)
}

func Restart(root string, extraEnv []string, timeout time.Duration) (AppStatus, error) {
	if _, err := Stop(root, timeout); err != nil {
		return AppStatus{}, err
	}
	return Start(root, extraEnv)
}

func extractArchive(archivePath, target string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(target, filepath.Clean(header.Name))
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) && path != filepath.Clean(target) {
			return fmt.Errorf("invalid archive path %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensurePersistentDirs(target, releaseDir string, dirs []string) error {
	for _, dir := range dirs {
		dst := filepath.Join(target, dir)
		if _, err := os.Stat(dst); err == nil {
			continue
		}

		src := filepath.Join(releaseDir, dir)
		if _, err := os.Stat(src); err == nil {
			if err := copyTree(src, dst); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func ensureRuntimeEnvFile(target, releaseDir string) error {
	targetEnv := filepath.Join(target, ".env")
	if fileExists(targetEnv) {
		return nil
	}
	sourceEnv := filepath.Join(releaseDir, ".env.example")
	data, err := os.ReadFile(sourceEnv)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetEnv), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetEnv, data, 0o640)
}

func copyTree(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, info.Mode())
	})
}

func resolveEntrypoint(root, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return ""
	}
	if filepath.IsAbs(configured) {
		return configured
	}
	return filepath.Clean(filepath.Join(root, configured))
}

func validateStartupPermissions(entrypoint string, paths RuntimePaths, identity ExecutionIdentity) error {
	if err := requireExecutable(entrypoint, identity); err != nil {
		return err
	}
	if err := requireDirectoryAccess(paths.Root, "install root", false, identity); err != nil {
		return err
	}
	for _, check := range []struct {
		path    string
		purpose string
	}{
		{path: paths.RunDir, purpose: "runtime run directory"},
		{path: paths.LogDir, purpose: "runtime log directory"},
		{path: paths.DataDir, purpose: "runtime data directory"},
	} {
		if err := requireDirectoryAccess(check.path, check.purpose, true, identity); err != nil {
			return err
		}
	}
	return nil
}

func requireExecutable(path string, identity ExecutionIdentity) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("entrypoint %s is a directory", path)
	}
	ownerUID, ownerGID := fileOwnership(info)
	if userCan(identity, info.Mode(), ownerUID, ownerGID, false, true) {
		return nil
	}
	return fmt.Errorf(
		"entrypoint %s is not executable for uid=%d gid=%d (mode=%#o owner=%d:%d)",
		path,
		identity.UID,
		identity.GID,
		info.Mode().Perm(),
		ownerUID,
		ownerGID,
	)
}

func requireDirectoryAccess(path, purpose string, write bool, identity ExecutionIdentity) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %s is not a directory", purpose, path)
	}
	ownerUID, ownerGID := fileOwnership(info)
	if userCan(identity, info.Mode(), ownerUID, ownerGID, write, true) {
		return nil
	}
	action := "accessible"
	if write {
		action = "writable"
	}
	return fmt.Errorf(
		"%s %s is not %s for uid=%d gid=%d (mode=%#o owner=%d:%d)",
		purpose,
		path,
		action,
		identity.UID,
		identity.GID,
		info.Mode().Perm(),
		ownerUID,
		ownerGID,
	)
}

func userCan(identity ExecutionIdentity, mode os.FileMode, ownerUID, ownerGID int, write, exec bool) bool {
	if identity.UID == 0 {
		return true
	}
	bits := 0
	if exec {
		bits |= 0o1
	}
	if write {
		bits |= 0o2
	}
	perm := int(mode.Perm())
	if ownerUID == identity.UID && perm&(bits<<6) == (bits<<6) {
		return true
	}
	for _, gid := range identity.GroupIDs {
		if ownerGID == gid && perm&(bits<<3) == (bits<<3) {
			return true
		}
	}
	return perm&bits == bits
}

func fileOwnership(info os.FileInfo) (int, int) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, -1
	}
	return int(stat.Uid), int(stat.Gid)
}

func tailFile(path string, maxLines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "no stderr output"
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, " | ")
}

func currentExecutionIdentity() ExecutionIdentity {
	groupIDs := []int{os.Getegid()}
	if extraGroups, err := os.Getgroups(); err == nil {
		for _, gid := range extraGroups {
			if gid == os.Getegid() {
				continue
			}
			groupIDs = append(groupIDs, gid)
		}
	}
	return ExecutionIdentity{
		UID:      os.Geteuid(),
		GID:      os.Getegid(),
		GroupIDs: groupIDs,
	}
}

func resolveExecutionIdentity(root, appName string) (ExecutionIdentity, error) {
	identity := currentExecutionIdentity()
	if !isManagedInstall(root) || os.Geteuid() != 0 {
		return identity, nil
	}
	username := runtimeUserName(appName)
	account, err := user.Lookup(username)
	if err != nil {
		return identity, nil
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return identity, err
	}
	gid, err := strconv.Atoi(account.Gid)
	if err != nil {
		return identity, err
	}
	groupIDs := []int{gid}
	if values, err := account.GroupIds(); err == nil {
		groupIDs = groupIDs[:0]
		for _, value := range values {
			parsed, parseErr := strconv.Atoi(value)
			if parseErr == nil {
				groupIDs = append(groupIDs, parsed)
			}
		}
		if len(groupIDs) == 0 {
			groupIDs = []int{gid}
		}
	}
	return ExecutionIdentity{
		User:     account.Username,
		Group:    sharedRuntimeGroup,
		UID:      uid,
		GID:      gid,
		GroupIDs: groupIDs,
	}, nil
}

func shouldDropPrivileges(layout AppLayout, identity ExecutionIdentity) bool {
	return layout.Packaged && os.Geteuid() == 0 && identity.UID > 0
}

func fallbackIdentityValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func uint32Groups(values []int) []uint32 {
	out := make([]uint32, 0, len(values))
	for _, value := range values {
		if value < 0 {
			continue
		}
		out = append(out, uint32(value))
	}
	return out
}

func isManagedInstall(root string) bool {
	root = filepath.Clean(strings.TrimSpace(root))
	roots := getManagedRoots()
	sep := string(os.PathSeparator)
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	for _, managedRoot := range roots {
		cleanRoot := filepath.Clean(managedRoot)
		if root == cleanRoot || strings.HasPrefix(root, cleanRoot+sep) {
			return true
		}
	}
	return false
}

func normalizePackagedRuntimeSecurity(target, releaseDir, appName string) error {
	if os.Geteuid() != 0 || !isManagedInstall(target) {
		return nil
	}
	identity, err := ensureManagedRuntimeIdentity(appName, target)
	if err != nil {
		return err
	}
	for _, dir := range []string{
		filepath.Join(target, "data"),
		filepath.Join(target, "var"),
		filepath.Join(target, "var", "run"),
		filepath.Join(target, "var", "log"),
	} {
		if err := os.MkdirAll(dir, 0o2750); err != nil {
			return err
		}
		if err := os.Chown(dir, identity.UID, identity.GID); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o2750); err != nil {
			return err
		}
	}
	for _, path := range []string{
		filepath.Join(target, ".env"),
		filepath.Join(target, ".env.local"),
		filepath.Join(target, ".env.example"),
		filepath.Join(releaseDir, ".env"),
		filepath.Join(releaseDir, ".env.local"),
		filepath.Join(releaseDir, ".env.example"),
		filepath.Join(target, "config"),
		filepath.Join(target, "config", "keys"),
	} {
		if err := normalizeSecurityPath(path, identity, false); err != nil {
			return err
		}
	}
	for _, root := range []string{filepath.Join(target, "data"), filepath.Join(target, "var"), filepath.Join(target, "config", "keys")} {
		if _, err := os.Stat(root); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			return normalizeSecurityEntry(path, info, identity)
		}); err != nil {
			return err
		}
	}
	if err := grantEdgeRuntimeCapabilities(target, releaseDir); err != nil {
		return err
	}
	return nil
}

func normalizeSecurityPath(path string, identity ExecutionIdentity, writable bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return normalizeSecurityEntryWithMode(path, info, identity, writable)
}

func normalizeSecurityEntry(path string, info os.FileInfo, identity ExecutionIdentity) error {
	return normalizeSecurityEntryWithMode(path, info, identity, true)
}

func normalizeSecurityEntryWithMode(path string, info os.FileInfo, identity ExecutionIdentity, writable bool) error {
	mode := os.FileMode(0o640)
	if info.IsDir() {
		if writable {
			mode = 0o2750
		} else {
			mode = 0o2750
		}
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") || base == "users.json" {
		mode = 0o600
		if info.IsDir() {
			mode = 0o2750
		}
	}
	if info.Mode()&0o111 != 0 && !info.IsDir() {
		mode = 0o750
	}
	if err := os.Chown(path, identity.UID, identity.GID); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func ensureManagedRuntimeIdentity(appName, homeDir string) (ExecutionIdentity, error) {
	if err := ensureGroupExists(sharedRuntimeGroup); err != nil {
		return ExecutionIdentity{}, err
	}
	username := runtimeUserName(appName)
	if err := migrateLegacyRuntimeUser(appName, username, homeDir); err != nil {
		return ExecutionIdentity{}, err
	}
	if _, err := user.Lookup(username); err != nil {
		if err := exec.Command("useradd", "--system", "--home-dir", homeDir, "--no-create-home", "--gid", sharedRuntimeGroup, "--shell", "/usr/sbin/nologin", username).Run(); err != nil {
			return ExecutionIdentity{}, fmt.Errorf("create runtime user %s: %w", username, err)
		}
	}
	identity, err := resolveExecutionIdentity(homeDir, appName)
	if err != nil {
		return ExecutionIdentity{}, err
	}
	if identity.User == "" {
		return ExecutionIdentity{}, fmt.Errorf("runtime user %s not available", username)
	}
	if err := cleanupLegacyRuntimeUser(appName, username, homeDir); err != nil {
		return ExecutionIdentity{}, err
	}
	return identity, nil
}

func ensureManagedOperatorIdentity(homeDir string) error {
	if os.Geteuid() != 0 || !isManagedInstall(homeDir) {
		return nil
	}
	if err := ensureGroupExists(sharedRuntimeGroup); err != nil {
		return err
	}
	if _, err := user.Lookup(sharedRuntimeGroup); err == nil {
		return nil
	}
	if err := exec.Command("useradd", "--system", "--home-dir", filepath.Clean("/opt/gmcore"), "--no-create-home", "--gid", sharedRuntimeGroup, "--shell", "/bin/bash", sharedRuntimeGroup).Run(); err != nil {
		return fmt.Errorf("create operator user %s: %w", sharedRuntimeGroup, err)
	}
	return nil
}

func ensureGroupExists(name string) error {
	if _, err := user.LookupGroup(name); err == nil {
		return nil
	}
	if err := exec.Command("groupadd", "--system", name).Run(); err != nil {
		return fmt.Errorf("create group %s: %w", name, err)
	}
	return nil
}

func runtimeUserName(appName string) string {
	name := strings.ToLower(strings.TrimSpace(appName))
	replacer := regexp.MustCompile(`[^a-z0-9]+`)
	name = replacer.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	name = strings.TrimPrefix(name, "gmcore-")
	if name == "" {
		name = "app"
	}
	name = "gmcore-" + name
	if len(name) > 31 {
		name = name[:31]
	}
	return strings.TrimRight(name, "-")
}

func legacyRuntimeUserName(appName string) string {
	name := strings.ToLower(strings.TrimSpace(appName))
	replacer := regexp.MustCompile(`[^a-z0-9]+`)
	name = replacer.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "app"
	}
	name = "gmcore-" + name
	if len(name) > 31 {
		name = name[:31]
	}
	return strings.TrimRight(name, "-")
}

func migrateLegacyRuntimeUser(appName, username, homeDir string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if _, err := user.Lookup(username); err == nil {
		return nil
	}
	legacy := "gmcore-" + legacyRuntimeUserName(appName)
	legacy = strings.TrimRight(legacy, "-")
	if legacy == username {
		return nil
	}
	if _, err := user.Lookup(legacy); err != nil {
		return nil
	}
	if err := exec.Command("usermod", "-l", username, "-d", homeDir, legacy).Run(); err != nil {
		return fmt.Errorf("rename runtime user %s to %s: %w", legacy, username, err)
	}
	return nil
}

func grantEdgeRuntimeCapabilities(target, releaseDir string) error {
	manifest, err := LoadAppManifest(target)
	if err != nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(manifest.Runtime.Mode), "edge") {
		return nil
	}
	setcapPath, err := exec.LookPath("setcap")
	if err != nil {
		return nil
	}
	entrypoint := resolveEntrypoint(target, manifest.Runtime.Entrypoint)
	if strings.TrimSpace(entrypoint) == "" {
		return nil
	}
	binaryPath := filepath.Join(releaseDir, "bin", filepath.Base(entrypoint))
	if !fileExists(binaryPath) {
		binaryPath = entrypoint
	}
	if !fileExists(binaryPath) {
		return nil
	}
	if err := exec.Command(setcapPath, "cap_net_bind_service=+ep", binaryPath).Run(); err != nil {
		return fmt.Errorf("setcap on %s: %w", binaryPath, err)
	}
	return nil
}

func grantEdgeEntrypointCapability(manifest AppManifest, entrypoint string) error {
	if os.Geteuid() != 0 || !strings.EqualFold(strings.TrimSpace(manifest.Runtime.Mode), "edge") {
		return nil
	}
	setcapPath, err := exec.LookPath("setcap")
	if err != nil || !fileExists(entrypoint) {
		return nil
	}
	if err := exec.Command(setcapPath, "cap_net_bind_service=+ep", entrypoint).Run(); err != nil {
		return fmt.Errorf("setcap on %s: %w", entrypoint, err)
	}
	return nil
}

func reconcileGlobalCLI(name, target string) error {
	if strings.TrimSpace(name) != "gmcore-cli" {
		return nil
	}
	linkPath := filepath.Clean("/usr/local/bin/gmcore-cli")
	targetPath := filepath.Join(filepath.Clean(target), "current", "bin", "gmcore-cli")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}
	if existing, err := os.Readlink(linkPath); err == nil {
		if filepath.Clean(existing) == filepath.Clean(targetPath) {
			return nil
		}
	}
	_ = os.Remove(linkPath)
	return os.Symlink(targetPath, linkPath)
}

func removeGlobalCLIIfMatches(target string) error {
	linkPath := filepath.Clean("/usr/local/bin/gmcore-cli")
	existing, err := os.Readlink(linkPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return nil
	}
	expectedPrefix := filepath.Join(filepath.Clean(target), "current", "bin", "gmcore-cli")
	if filepath.Clean(existing) != filepath.Clean(expectedPrefix) {
		return nil
	}
	return os.Remove(linkPath)
}

func cleanupLegacyRuntimeUser(appName, username, homeDir string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	legacy := "gmcore-" + legacyRuntimeUserName(appName)
	legacy = strings.TrimRight(legacy, "-")
	if legacy == username {
		return nil
	}
	desiredUser, err := user.Lookup(username)
	if err != nil {
		return nil
	}
	legacyUser, err := user.Lookup(legacy)
	if err != nil {
		return nil
	}
	desiredUID, err := strconv.Atoi(desiredUser.Uid)
	if err != nil {
		return err
	}
	desiredGID, err := strconv.Atoi(desiredUser.Gid)
	if err != nil {
		return err
	}
	legacyUID, err := strconv.Atoi(legacyUser.Uid)
	if err != nil {
		return err
	}
	legacyGID, err := strconv.Atoi(legacyUser.Gid)
	if err != nil {
		return err
	}
	if err := filepath.Walk(homeDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		ownerUID, ownerGID := fileOwnership(info)
		if ownerUID == legacyUID || ownerGID == legacyGID {
			if chownErr := os.Chown(path, desiredUID, desiredGID); chownErr != nil {
				return chownErr
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := exec.Command("userdel", legacy).Run(); err != nil {
		return fmt.Errorf("remove legacy runtime user %s: %w", legacy, err)
	}
	return nil
}

func readRunningPID(path string) (int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, false, nil
		}
		return 0, false, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false, err
	}
	return pid, pidRunning(pid), nil
}

func writeInstanceMetadata(path string, meta instanceMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

func stopDuplicateManagedProcesses(paths RuntimePaths, timeout time.Duration) error {
	pids, err := findDuplicateManagedProcesses(paths)
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		return nil
	}
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range pids {
			if pidRunning(pid) {
				alive = true
				break
			}
		}
		if !alive {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	for _, pid := range pids {
		if pidRunning(pid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	return nil
}

func findDuplicateManagedProcesses(paths RuntimePaths) ([]int, error) {
	if runtime.GOOS == "windows" {
		return findDuplicateManagedProcessesWindows(paths)
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, nil
	}
	seen := map[int]struct{}{}
	var out []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == os.Getpid() {
			continue
		}
		environ, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
		if err != nil || len(environ) == 0 {
			continue
		}
		values := parseProcEnviron(environ)
		if values["GMCORE_MANAGED_LAUNCH"] != "1" {
			continue
		}
		if values["APP_PID_FILE"] == paths.PIDPath || values["APP_DATA_DIR"] == paths.DataDir || values["APP_HTTP_SOCKET"] == paths.HTTPSocketPath {
			if _, ok := seen[pid]; !ok {
				seen[pid] = struct{}{}
				out = append(out, pid)
			}
		}
	}
	return out, nil
}

func findDuplicateManagedProcessesWindows(paths RuntimePaths) ([]int, error) {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	seen := map[int]struct{}{}
	var out []int
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		pidStr := strings.Trim(parts[1], "\" ")
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 || pid == os.Getpid() {
			continue
		}
		cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "CommandLine")
		cmdOut, err := cmd.Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(cmdOut), "GMCORE_MANAGED_LAUNCH=1") {
			if _, ok := seen[pid]; !ok {
				seen[pid] = struct{}{}
				out = append(out, pid)
			}
		}
	}
	return out, nil
}

func parseProcEnviron(data []byte) map[string]string {
	out := map[string]string{}
	for _, item := range strings.Split(string(data), "\x00") {
		if strings.TrimSpace(item) == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}

func pidRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func socketExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func removeSocket(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSocket != 0 {
		_ = os.Remove(path)
	}
}

func withEnv(base []string, key, value string) []string {
	key = strings.TrimSpace(key)
	if key == "" {
		return base
	}
	prefix := key + "="
	out := make([]string, 0, len(base)+1)
	replaced := false
	for _, item := range base {
		if strings.HasPrefix(item, prefix) {
			out = append(out, prefix+value)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, prefix+value)
	}
	return out
}

func runtimeMode(manifest AppManifest) string {
	mode := strings.TrimSpace(manifest.Runtime.Mode)
	if mode == "" {
		return "edge"
	}
	return mode
}

func computeReady(status AppStatus) bool {
	switch status.RuntimeMode {
	case "uds":
		return status.Running && status.HTTPReady
	case "edge":
		return status.Running
	default:
		return status.Running
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func envValue(values []string, key string) string {
	prefix := strings.TrimSpace(key) + "="
	for _, item := range values {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}
