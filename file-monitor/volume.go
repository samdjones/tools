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
func getVolumeLabel(mountPoint string) string {
	if _, err := os.Stat(mountPoint); err == nil {
		return filepath.VolumeName(mountPoint)
	}
	return ""
}

// runMonitor handles standard directory monitoring with optional destination volume.
func runMonitor(src, dst, destVolumeName, destVolumePath, ext string, del, rename bool, pattern string) {
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		log.Fatalf("Error: source %q does not exist or is not a directory\n", src)
	}

	var actualDst string
	if destVolumeName != "" {
		actualDst = resolveDest(destVolumeName, destVolumePath)
	} else {
		actualDst = dst
		if err := ensureDestDir(actualDst); err != nil {
			log.Fatalf("Error preparing destination: %v\n", err)
		}
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
		Src:              src,
		Dst:              actualDst,
		DestVolumeName:   destVolumeName,
		DestVolumePath:   destVolumePath,
		Exts:             exts,
		Delete:           del,
		Rename:           rename,
		Pattern:          pattern,
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
	if destVolumeName != "" {
		log.Printf("(destination volume %q - waiting if needed)", destVolumeName)
	}
	log.Printf("Filter     : %s", filterDesc)
	if cfg.Delete {
		log.Printf("Mode       : move (delete after copy)")
	} else {
		log.Printf("Mode       : copy")
	}
	if cfg.Rename {
		log.Printf("Rename     : enabled (pattern: %s)", cfg.Pattern)
	}

	monitorWithDestVolume(watcher, cfg)
}

// resolveDest waits for a destination volume and returns its path.
func resolveDest(volumeName, volumePath string) string {
	log.Printf("Waiting for destination volume %q to be mounted...", volumeName)

	for {
		mountPoint := findVolumeByLabel(volumeName)
		if mountPoint != "" {
			log.Printf("Destination volume %q found at %s", volumeName, mountPoint)
			dst := mountPoint
			if volumePath != "" {
				dst = filepath.Join(mountPoint, volumePath)
			}
			return dst
		}
		time.Sleep(2 * time.Second)
	}
}

// ensureDestDir creates the destination directory if it doesn't exist.
func ensureDestDir(dst string) error {
	return os.MkdirAll(dst, 0o755)
}

// runWithVolume waits for a source volume to be mounted, then starts monitoring.
func runWithVolume(volumeName, volumePath, dst, destVolumeName, destVolumePath, ext string, del, rename bool, pattern string) {
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

	var actualDst string
	if destVolumeName != "" {
		actualDst = resolveDest(destVolumeName, destVolumePath)
	} else {
		actualDst = dst
		if err := ensureDestDir(actualDst); err != nil {
			log.Fatalf("Error creating destination directory: %v\n", err)
		}
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
		Src:              src,
		Dst:              actualDst,
		DestVolumeName:   destVolumeName,
		DestVolumePath:   destVolumePath,
		Exts:             exts,
		Delete:           del,
		Rename:           rename,
		Pattern:          pattern,
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
	if destVolumeName != "" {
		log.Printf("(destination volume %q - waiting if needed)", destVolumeName)
	}
	log.Printf("Filter     : %s", filterDesc)
	if cfg.Delete {
		log.Printf("Mode       : move (delete after copy)")
	} else {
		log.Printf("Mode       : copy")
	}
	if cfg.Rename {
		log.Printf("Rename     : enabled (pattern: %s)", cfg.Pattern)
	}

	monitorWithDestVolume(watcher, cfg)
}
