# file-monitor

A Windows CLI tool written in Go that watches a source directory and automatically copies new files to a destination directory.

## Features

- Watch a directory for newly created files
- **Auto-detect and monitor removable volumes** (memory cards, USB drives) when mounted
- **Tolerate removable destination volumes** — pauses and resumes when destination is unavailable
- Filter by one or more file extensions
- Optionally delete the source file after copying (move behaviour)
- Optionally rename copied files with a configurable datetime suffix

## Installation

```
go install github.com/samdjones/file-monitor@latest
```

Or build from source:

```
git clone https://github.com/samdjones/file-monitor
cd file-monitor
go build -o file-monitor.exe .
```

## Usage

### Direct directory monitoring

```
file-monitor -src <source-dir> -dst <destination-dir> [options]
```

### Automatic source volume monitoring

```
file-monitor -volume-name <volume-label> -dst <destination-dir> [options]
```

### Automatic destination volume monitoring

```
file-monitor -src <source-dir> -dest-volume-name <volume-label> [options]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-src` | *(optional)* | Directory to monitor (use `-src` or `-volume-name`, not both) |
| `-volume-name` | | Volume label to watch for as source; monitoring starts when volume is mounted |
| `-volume-path` | root | Subdirectory on the source volume to monitor (e.g., `DCIM` for camera cards) |
| `-dst` | *(optional)* | Destination directory (use `-dst` or `-dest-volume-name`, not both) |
| `-dest-volume-name` | | Volume label to watch for as destination; syncing waits if volume is unmounted |
| `-dest-volume-path` | root | Subdirectory on the destination volume |
| `-ext` | *(all files)* | Comma-separated extensions to watch, e.g. `.txt,.jpg` |
| `-delete` | `false` | Delete source file after a successful copy |
| `-rename` | `false` | Append a datetime suffix to copied filenames |
| `-pattern` | `20060102_150405` | Go time format string used for the datetime suffix |
| `-version` | | Print version and exit |

### Examples

Copy every new `.log` file from `C:\\logs\\incoming` to `C:\\logs\\archive`:

```
file-monitor -src C:\logs\incoming -dst C:\logs\archive -ext .log
```

Move new `.jpg` and `.png` photos, renaming them with a timestamp:

```
file-monitor -src D:\camera -dst D:\photos -ext .jpg,.png -delete -rename
```

Resulting filename: `photo_20240315_143022.jpg`

Use a custom datetime pattern (year-month-day only):

```
file-monitor -src D:\camera -dst D:\photos -rename -pattern 2006-01-02
```

Auto-monitor a memory card: wait for volume "MCARD" to be mounted, then copy `.jpg` files from its `DCIM` folder:

```
file-monitor -volume-name MCARD -volume-path DCIM -dst C:\photos -ext .jpg
```

Auto-monitor a USB drive: start copying as soon as a drive labeled "BACKUP" appears:

```
file-monitor -volume-name BACKUP -dst C:\backups -delete -rename
```

Monitor local folder but wait for a backup drive: copy files only when the destination "BACKUP" drive is mounted:

```
file-monitor -src C:\documents -dest-volume-name BACKUP -dest-volume-path backups
```

If the backup drive is ejected while files are waiting, file-monitor automatically pauses and resumes when the drive remounts.

Monitor both source and destination volumes: watch camera's SD card and copy to external backup drive (both removable):

```
file-monitor -volume-name CAMERA -volume-path DCIM -dest-volume-name BACKUP -ext .jpg
```

### Running as a Windows service

You can wrap `file-monitor` with [NSSM](https://nssm.cc/) or the built-in `sc` command to run it as a background Windows service.

## Development

```
go test ./...
go build .
```

## License

MIT
