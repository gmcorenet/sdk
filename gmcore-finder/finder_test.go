package gmcore_finder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	opts := &FindOptions{Recursive: true, IncludeFiles: true, IncludeDirs: false}
	files, err := FindFiles(tmpDir, opts)
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFindDirs(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	subSubDir := filepath.Join(subDir, "subSubDir")
	err = os.Mkdir(subSubDir, 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	dirs, err := FindDirs(tmpDir, nil)
	if err != nil {
		t.Fatalf("FindDirs failed: %v", err)
	}

	if len(dirs) < 2 {
		t.Errorf("expected at least 2 dirs, got %d", len(dirs))
	}
}

func TestSearchInFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	content := "line1 with pattern\nline2 without\nline3 with pattern again"
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	matches, err := SearchInFile(path, "pattern", nil)
	if err != nil {
		t.Fatalf("SearchInFile failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestSearch(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	file2 := filepath.Join(tmpDir, "file2.txt")
	err = os.WriteFile(file2, []byte("hello golang"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	matches, err := Search(tmpDir, "hello", nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestSearchWithCaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(path, []byte("Hello hello HELLO"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	opts := &SearchOptions{CaseSensitive: true}
	matches, err := SearchInFile(path, "hello", opts)
	if err != nil {
		t.Fatalf("SearchInFile failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match with case sensitive, got %d", len(matches))
	}

	opts = &SearchOptions{CaseSensitive: false}
	matches, err = SearchInFile(path, "hello", opts)
	if err != nil {
		t.Fatalf("SearchInFile failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 line match without case sensitive, got %d", len(matches))
	}
}

func TestSearchWithWholeWord(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(path, []byte("hello world helloWorld hello"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	opts := &SearchOptions{WholeWord: true}
	matches, err := SearchInFile(path, "hello", opts)
	if err != nil {
		t.Fatalf("SearchInFile failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 line match with whole word, got %d", len(matches))
	}
}

func TestFindByName(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	matches, err := FindByName(tmpDir, "test", nil)
	if err != nil {
		t.Fatalf("FindByName failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
}

func TestFindByExtension(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "file2.log"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	matches, err := FindByExtension(tmpDir, ".txt", nil)
	if err != nil {
		t.Fatalf("FindByExtension failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
}

func TestCountMatches(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("hello golang"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	count, err := CountMatches(tmpDir, "hello", nil)
	if err != nil {
		t.Fatalf("CountMatches failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 matches, got %d", count)
	}
}
