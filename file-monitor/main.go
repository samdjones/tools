package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	ver := flag.Bool("version", false, "Print version and exit")
	src := flag.String("src", "", "Source directory to monitor (required unless -volume-name is used)")
	dst := flag.String("dst", "", "Destination directory for copied files (required unless -dest-volume-name is used)")
	ext := flag.String("ext", "", "Comma-separated extensions to watch, e.g. .txt,.jpg (empty = all files)")
	del := flag.Bool("delete", false, "Delete source file after successful copy")
	rename := flag.Bool("rename", false, "Rename copied file by appending a datetime suffix")
	pattern := flag.String("pattern", "20060102_150405", "Go time format string used for the datetime suffix")
	volumeName := flag.String("volume-name", "", "Watch for a volume with this label and monitor it when mounted")
	volumePath := flag.String("volume-path", "", "Subdirectory on the volume to monitor (default: root)")
	destVolumeName := flag.String("dest-volume-name", "", "Wait for destination volume with this label; syncing starts when mounted")
	destVolumePath := flag.String("dest-volume-path", "", "Subdirectory on the destination volume (default: root)")
	flag.Parse()

	if *ver {
		fmt.Println("file-monitor", version)
		os.Exit(0)
	}

	if *dst == "" && *destVolumeName == "" {
		fmt.Fprintln(os.Stderr, "Error: -dst or -dest-volume-name is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	if *dst != "" && *destVolumeName != "" {
		fmt.Fprintln(os.Stderr, "Error: cannot use -dst with -dest-volume-name")
		os.Exit(1)
	}

	if *volumeName != "" {
		if *src != "" {
			fmt.Fprintln(os.Stderr, "Error: cannot use -src with -volume-name")
			os.Exit(1)
		}
		runWithVolume(*volumeName, *volumePath, *dst, *destVolumeName, *destVolumePath, *ext, *del, *rename, *pattern)
		return
	}

	if *src == "" {
		fmt.Fprintln(os.Stderr, "Error: -src or -volume-name is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	runMonitor(*src, *dst, *destVolumeName, *destVolumePath, *ext, *del, *rename, *pattern)
}
