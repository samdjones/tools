package main

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Config holds the runtime configuration for the monitor.
type Config struct {
	Src     string
	Dst     string
	Exts    []string
	Delete  bool
	Rename  bool
	Pattern string
}

func monitor(watcher *fsnotify.Watcher, cfg Config) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				// Brief pause so the OS finishes writing the file before we copy it.
				time.Sleep(150 * time.Millisecond)
				if err := handleFile(event.Name, cfg); err != nil {
					log.Printf("Error: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
