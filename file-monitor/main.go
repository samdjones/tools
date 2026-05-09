package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var version = "dev"

func main() {
	ver := flag.Bool("version", false, "Print version and exit")
	src := flag.String("src", "", "Source directory to monitor (required)")
	dst := flag.String("dst", "", "Destination directory for copied files (required)")
	ext := flag.String("ext", "", "Comma-separated extensions to watch, e.g. .txt,.jpg (empty = all files)")
	del := flag.Bool("delete", false, "Delete source file after successful copy")
	rename := flag.Bool("rename", false, "Rename copied file by appending a datetime suffix")
	pattern := flag.String("pattern", "20060102_150405", "Go time format string used for the datetime suffix")
	flag.Parse()

	if *ver {
		fmt.Println("file-monitor", version)
		os.Exit(0)
	}

	if *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "Error: -src and -dst are required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	if info, err := os.Stat(*src); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: source %q does not exist or is not a directory\n", *src)
		os.Exit(1)
	}

	if err := os.MkdirAll(*dst, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination directory: %v\n", err)
		os.Exit(1)
	}

	var exts []string
	for _, e := range strings.Split(*ext, ",") {
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
		Src:     *src,
		Dst:     *dst,
		Exts:    exts,
		Delete:  *del,
		Rename:  *rename,
		Pattern: *pattern,
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
