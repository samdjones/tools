package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		path string
		exts []string
		want bool
	}{
		{"file.txt", []string{".txt"}, true},
		{"file.TXT", []string{".txt"}, true},
		{"file.jpg", []string{".txt", ".jpg"}, true},
		{"file.png", []string{".txt", ".jpg"}, false},
		{"file.png", nil, true},
		{"noext", []string{".txt"}, false},
	}
	for _, tc := range tests {
		got := matchesFilter(tc.path, tc.exts)
		if got != tc.want {
			t.Errorf("matchesFilter(%q, %v) = %v, want %v", tc.path, tc.exts, got, tc.want)
		}
	}
}

func TestBuildDstName_NoRename(t *testing.T) {
	cfg := Config{Rename: false}
	if got := buildDstName("report.pdf", cfg); got != "report.pdf" {
		t.Errorf("got %q, want %q", got, "report.pdf")
	}
}

func TestBuildDstName_Rename(t *testing.T) {
	cfg := Config{Rename: true, Pattern: "20060102"}
	got := buildDstName("report.pdf", cfg)
	if !strings.HasPrefix(got, "report_") || !strings.HasSuffix(got, ".pdf") {
		t.Errorf("unexpected renamed filename: %q", got)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", string(data), "hello")
	}
}

func TestHandleFile_ExtFilter(t *testing.T) {
	dir := t.TempDir()
	dst := t.TempDir()

	src := filepath.Join(dir, "image.png")
	os.WriteFile(src, []byte("data"), 0o644)

	cfg := Config{Src: dir, Dst: dst, Exts: []string{".txt"}}
	if err := handleFile(src, cfg); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dst)
	if len(entries) != 0 {
		t.Errorf("expected no files copied for filtered extension, got %d", len(entries))
	}
}

func TestHandleFile_Copy(t *testing.T) {
	dir := t.TempDir()
	dst := t.TempDir()

	src := filepath.Join(dir, "doc.txt")
	os.WriteFile(src, []byte("content"), 0o644)

	cfg := Config{Src: dir, Dst: dst, Exts: []string{".txt"}}
	if err := handleFile(src, cfg); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dst)
	if len(entries) != 1 {
		t.Errorf("expected 1 file copied, got %d", len(entries))
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source file should still exist: %v", err)
	}
}

func TestHandleFile_Delete(t *testing.T) {
	dir := t.TempDir()
	dst := t.TempDir()

	src := filepath.Join(dir, "doc.txt")
	os.WriteFile(src, []byte("content"), 0o644)

	cfg := Config{Src: dir, Dst: dst, Delete: true}
	if err := handleFile(src, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file should have been deleted")
	}
}
