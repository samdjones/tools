package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleFile(path string, cfg Config) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil // file may have already been removed
	}
	if info.IsDir() {
		return nil
	}
	if !matchesFilter(path, cfg.Exts) {
		return nil
	}

	dstName := buildDstName(filepath.Base(path), cfg)
	dstPath := filepath.Join(cfg.Dst, dstName)

	if err := copyFile(path, dstPath); err != nil {
		return fmt.Errorf("copy %q to %q: %w", path, dstPath, err)
	}
	log.Printf("Copied  %q to %q", filepath.Base(path), dstPath)

	if cfg.Delete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("delete %q: %w", path, err)
		}
		log.Printf("Deleted %q", path)
	}
	return nil
}

func buildDstName(name string, cfg Config) string {
	if !cfg.Rename {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	ts := time.Now().Format(cfg.Pattern)
	return base + "_" + ts + ext
}

func matchesFilter(path string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	fileExt := strings.ToLower(filepath.Ext(path))
	for _, e := range exts {
		if fileExt == e {
			return true
		}
	}
	return false
}

func copyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); retErr == nil {
			retErr = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
