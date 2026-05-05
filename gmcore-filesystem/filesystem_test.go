package gmcore_filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	data := []byte("hello world")
	err := WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readData, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(readData) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(readData))
	}
}

func TestReadFileLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	data := []byte("line1\nline2\nline3")
	err := WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	lines, err := ReadFileLines(path)
	if err != nil {
		t.Fatalf("ReadFileLines failed: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	err := WriteFile(src, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = CopyFile(src, dst)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	if !Exists(dst) {
		t.Error("destination file should exist")
	}

	readData, err := ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readData) != "test content" {
		t.Errorf("expected 'test content', got %q", string(readData))
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	err := os.Mkdir(src, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = CopyDir(src, dst)
	if err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	if !IsDir(dst) {
		t.Error("destination should be a directory")
	}

	copiedFile := filepath.Join(dst, "file.txt")
	if !Exists(copiedFile) {
		t.Error("copied file should exist")
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	err := WriteFile(src, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = MoveFile(src, dst)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	if Exists(src) {
		t.Error("source file should not exist after move")
	}

	if !Exists(dst) {
		t.Error("destination file should exist after move")
	}
}

func TestRemoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	err := WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = RemoveFile(path)
	if err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	if Exists(path) {
		t.Error("file should not exist after removal")
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	if Exists(path) {
		t.Error("file should not exist initially")
	}

	err := WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !Exists(path) {
		t.Error("file should exist after creation")
	}
}

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()

	if !IsDir(tmpDir) {
		t.Error("temp dir should be a directory")
	}

	path := filepath.Join(tmpDir, "test.txt")
	err := WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if IsDir(path) {
		t.Error("file should not be a directory")
	}
}

func TestIsFile(t *testing.T) {
	tmpDir := t.TempDir()

	if IsFile(tmpDir) {
		t.Error("directory should not be a file")
	}

	path := filepath.Join(tmpDir, "test.txt")
	err := WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !IsFile(path) {
		t.Error("file should be a file")
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	err := WriteFile(path, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	info, err := GetFileInfo(path)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got %q", info.Name)
	}

	if info.Size != 4 {
		t.Errorf("expected size 4, got %d", info.Size)
	}

	if info.IsDir {
		t.Error("should not be a directory")
	}
}

func TestWalk(t *testing.T) {
	tmpDir := t.TempDir()

	err := WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test1"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = WriteFile(filepath.Join(subDir, "file2.txt"), []byte("test2"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var visited []string
	err = Walk(tmpDir, func(path string, info *FileInfo, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if len(visited) < 4 {
		t.Errorf("expected at least 4 entries, got %d: %v", len(visited), visited)
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test1"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test2"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	files, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestListDirs(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	dirs, err := ListDirs(tmpDir)
	if err != nil {
		t.Fatalf("ListDirs failed: %v", err)
	}

	if len(dirs) != 1 {
		t.Errorf("expected 1 directory, got %d", len(dirs))
	}
}

func TestGlob(t *testing.T) {
	tmpDir := t.TempDir()

	err := WriteFile(filepath.Join(tmpDir, "test1.txt"), []byte("test1"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = WriteFile(filepath.Join(tmpDir, "test2.txt"), []byte("test2"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("other"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	matches, err := Glob(tmpDir, "test*.txt")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}
