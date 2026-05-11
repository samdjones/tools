# file-monitor

A Windows CLI tool written in Go that watches a source directory and automatically copies new files to a destination directory.

## Features

- Watch a directory for newly created files
- **Auto-detect and monitor removable volumes** (memory cards, USB drives) when mounted
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

### Automatic volume monitoring

```
file-monitor -volume-name <volume-label> -dst <destination-dir> [options]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-src` | *(optional)* | Directory to monitor (use `-src` or `-volume-name`, not both) |
| `-volume-name` | | Volume label to watch for; monitoring starts when volume is mounted |
| `-volume-path` | root | Subdirectory on the volume to monitor (e.g., `DCIM` for camera cards) |
| `-dst` | *(required)* | Directory to copy files into (created if it doesn't exist) |
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

### Running as a Windows service

You can wrap `file-monitor` with [NSSM](https://nssm.cc/) or the built-in `sc` command to run it as a background Windows service.

## Development

```
go test ./...
go build .
```

## License

MIT
