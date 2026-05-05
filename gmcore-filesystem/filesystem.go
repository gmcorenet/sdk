package gmcore_filesystem

// Package gmcore_filesystem provides file system utilities for common operations
// like reading, writing, copying, moving, and walking directories.
//
// Examples:
//
//	// Read and write files
//	data, _ := ReadFile("file.txt")
//	WriteFile("new.txt", data, 0644)
//
//	// Copy files and directories
//	CopyFile("src.txt", "dst.txt")
//	CopyDir("src_dir/", "dst_dir/")
//
//	// Walk directory tree
//	Walk("/path", func(path string, info *FileInfo, err error) error {
//	    fmt.Println(path)
//	    return nil
//	})

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type FileInfo struct {
	Path     string
	Name     string
	Size     int64
	IsDir    bool
	IsSymlink bool
	Mode     os.FileMode
	ModTime  int64
}

type WalkFunc func(path string, info *FileInfo, err error) error

func EnsureDir(path string, mode os.FileMode) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path %q exists but is not a directory", path)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	parent := filepath.Dir(path)
	if parent != path {
		if err := EnsureDir(parent, mode); err != nil {
			return err
		}
	}

	if err := os.Mkdir(path, mode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func WriteFile(path string, data []byte, mode os.FileMode) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	dir := filepath.Dir(path)
	if err := EnsureDir(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure directory: %w", err)
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func ReadFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

func ReadFileLines(path string) ([]string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := splitLines(data)
	return lines, nil
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := string(data[start:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(data) {
		line := string(data[start:])
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}

func CopyFile(src, dst string) error {
	if src == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if dst == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("source is a directory")
	}

	dir := filepath.Dir(dst)
	if err := EnsureDir(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure destination directory: %w", err)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

func CopyDir(src, dst string) error {
	if src == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if dst == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	if err := EnsureDir(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else if entry.Type()&os.ModeSymlink == 0 {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			link, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read symlink: %w", err)
			}
			if err := os.Symlink(link, dstPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		}
	}

	return nil
}

func MoveFile(src, dst string) error {
	if src == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if dst == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	if err := os.Rename(src, dst); err != nil {
		if linkErr, ok := err.(*os.LinkError); ok {
			if perrm, ok := linkErr.Err.(syscall.Errno); ok && perrm == syscall.EXDEV {
				if err := CopyFile(src, dst); err != nil {
					return err
				}
				if err := os.Remove(src); err != nil {
					return fmt.Errorf("failed to remove source after copy: %w", err)
				}
				return nil
			}
		}
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func RemoveFile(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

func RemoveDir(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove directory: %w", err)
	}

	return nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func GetFileInfo(path string) (*FileInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Path:     path,
		Name:     info.Name(),
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		IsSymlink: info.Mode()&os.ModeSymlink != 0,
		Mode:     info.Mode(),
		ModTime:  info.ModTime().Unix(),
	}, nil
}

func Walk(root string, fn WalkFunc) error {
	if root == "" {
		return fmt.Errorf("root path cannot be empty")
	}

	info, err := os.Lstat(root)
	if err != nil {
		return fn(root, nil, fmt.Errorf("failed to lstat root: %w", err))
	}

	if err := walkRecursive(root, info, fn); err != nil {
		return err
	}

	return nil
}

func walkRecursive(path string, info os.FileInfo, fn WalkFunc) error {
	fileInfo := &FileInfo{
		Path:     path,
		Name:     info.Name(),
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		IsSymlink: info.Mode()&os.ModeSymlink != 0,
		Mode:     info.Mode(),
		ModTime:  info.ModTime().Unix(),
	}

	if err := fn(path, fileInfo, nil); err != nil {
		return err
	}

	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fn(path, fileInfo, fmt.Errorf("failed to read directory: %w", err))
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		entryInfo, err := entry.Info()
		if err != nil {
			if err := fn(entryPath, nil, fmt.Errorf("failed to get entry info: %w", err)); err != nil {
				return err
			}
			continue
		}

		if err := walkRecursive(entryPath, entryInfo, fn); err != nil {
			return err
		}
	}

	return nil
}

func ListFiles(dir string) ([]string, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func ListDirs(dir string) ([]string, error) {
	if dir == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(dir, entry.Name()))
		}
	}

	return dirs, nil
}

func Glob(root, pattern string) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	pattern = filepath.Join(root, pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob failed: %w", err)
	}

	return matches, nil
}

func GlobRecursive(root, pattern string) ([]string, error) {
	if root == "" {
		return nil, fmt.Errorf("root path cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	fullPattern := filepath.Join(root, "**", pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("glob recursive failed: %w", err)
	}

	return matches, nil
}
