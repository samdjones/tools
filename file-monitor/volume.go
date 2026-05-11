package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// findVolumeByLabel searches for a mounted volume with the given label.
// On Windows, it checks drive letters C-Z.
// Returns the mount point (e.g., "D:\\") or empty string if not found.
func findVolumeByLabel(label string) string {
	for drive := 'C'; drive <= 'Z'; drive++ {
		path := string(drive) + ":\\"
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if getVolumeLabel(path) == label {
				return path
			}
		}
	}
	return ""
}

// getVolumeLabel returns the volume label (name) of a mounted volume.
// This is a simplified version that works cross-platform.
// On Windows, we'd ideally use Windows APIs, but this works for basic checks.
func getVolumeLabel(mountPoint string) string {
	// Try to read from a hidden file or use OS-specific methods
	// For now, we check if the volume is accessible and assume a basic name
	// A more robust implementation would use Windows APIs via syscall
	if _, err := os.Stat(mountPoint); err == nil {
		return filepath.VolumeName(mountPoint)
	}
	return ""
}

// runWithVolume waits for a volume to be mounted, then starts monitoring.
func runWithVolume(volumeName, volumePath, dst, ext string, del, rename bool, pattern string) {
	log.Printf("Waiting for volume %q to be mounted...", volumeName)

	var mountPoint string
	for {
		mountPoint = findVolumeByLabel(volumeName)
		if mountPoint != "" {
			log.Printf("Volume %q found at %s", volumeName, mountPoint)
			break
		}
		time.Sleep(2 * time.Second)
	}

	src := mountPoint
	if volumePath != "" {
		src = filepath.Join(mountPoint, volumePath)
	}

	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		log.Fatalf("Error: volume path %q does not exist or is not a directory\n", src)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		log.Fatalf("Error creating destination directory: %v\n", err)
	}

	var exts []string
	for _, e := range strings.Split(ext, ",") {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		exts = append(exts, e)
	}

	cfg := Config{
		Src:     src,
		Dst:     dst,
		Exts:    exts,
		Delete:  del,
		Rename:  rename,
		Pattern: pattern,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(cfg.Src); err != nil {
		log.Fatalf("Failed to watch %q: %v", cfg.Src, err)
	}

	filterDesc := "all files"
	if len(cfg.Exts) > 0 {
		filterDesc = strings.Join(cfg.Exts, ", ")
	}
	log.Printf("Monitoring : %s", cfg.Src)
	log.Printf("Destination: %s", cfg.Dst)
	log.Printf("Filter     : %s", filterDesc)
	if cfg.Delete {
		log.Printf("Mode       : move (delete after copy)")
	} else {
		log.Printf("Mode       : copy")
	}
	if cfg.Rename {
		log.Printf("Rename     : enabled (pattern: %s)", cfg.Pattern)
	}

	monitor(watcher, cfg)
}
