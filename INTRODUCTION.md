# imgdup — Image & Video Duplicate Detector

> Find visually similar photos and videos on your drive — in seconds, from your terminal.

---

## The Problem

You have thousands of photos from phone backups, camera imports, and shared drives. Some are exact copies. Others are near-identical — same shot, slightly different crop or compression. Sorting through them manually wastes hours. `imgdup` does it in one command.

---

## Demo

```
$ imgdup -f ~/Photos -s 90

  Scanning files... [████████████████████░░░░░] 840/1000

  ┌─────────────────────────────────────────────────────────────────┐
  │  3 duplicate groups found  ·  14 files  ·  scanned in 2.4s     │
  │  Recoverable space: 218 MB                                      │
  └─────────────────────────────────────────────────────────────────┘

  Group 1  ·  similarity 98.4%  ·  saves 87 MB
  ┌──────┬──────────────────────────────────────┬────────┬──────────┐
  │      │ File                                 │   Size │ Modified │
  ├──────┼──────────────────────────────────────┼────────┼──────────┤
  │ KEEP │ ~/Photos/2024/IMG_4021.jpg           │ 9.1 MB │ Jan 2024 │
  │      │ ~/Photos/backup/IMG_4021_copy.jpg    │ 4.3 MB │ Mar 2024 │
  │      │ ~/Downloads/IMG_4021_compressed.jpg  │ 2.8 MB │ Apr 2024 │
  └──────┴──────────────────────────────────────┴────────┴──────────┘
  ...
```

---

## Features

| Feature | Details |
|---|---|
| **Perceptual similarity** | Detects visually similar images even after resize, crop, or re-encode |
| **Video support** | Compares videos frame-by-frame (requires `ffmpeg`) |
| **Exact duplicates** | MD5 matching catches byte-for-byte copies instantly |
| **Interactive mode** | Guided prompts when no flags are provided — no manual needed |
| **Multiple output formats** | `table` (default), `json`, `csv` — pipe into other tools |
| **Export to file** | `--out-file report.csv` saves results for later review |
| **Hash cache** | Re-scans skip already-processed files — fast on repeated runs |
| **Recursive scan** | Optionally walk all subdirectories with `-r` |
| **Parallel processing** | Uses all CPU cores for images; up to 4 workers for video |
| **Graceful interruption** | `Ctrl+C` stops cleanly without corrupting the cache |

---

## Supported Formats

**Images:** `.jpg` `.jpeg` `.png` `.gif` `.webp` `.tiff` `.tif` `.bmp`

**Videos:** `.mp4` `.mov` `.avi` `.mkv` `.m4v` `.wmv` `.webm` `.3gp` *(ffmpeg required)*

---

## Quick Start

### Install

```bash
# Build from source (requires Go 1.21+)
git clone https://github.com/imgdup/image-dupl-detector
cd image-dupl-detector
make build          # produces ./imgdup

# Move to your PATH
mv imgdup /usr/local/bin/
```

### Run (interactive)

```bash
imgdup
# → Prompts you for folder and similarity threshold
```

### Run (with flags)

```bash
imgdup -f ~/Pictures -s 90
```

---

## CLI Reference

```
Usage:
  imgdup [flags]

Flags:
  -f, --folder string       Folder to scan
  -s, --similarity int      Similarity threshold 1–100 (default: prompts)
  -r, --recursive           Scan subfolders recursively
  -o, --output string       Output format: table, json, csv  (default: table)
      --out-file string      Write results to a file
  -q, --quiet               Suppress progress bar, print results only
      --no-color            Disable ANSI colours
  -v, --version             Show version
  -h, --help                Show help
```

### Examples

```bash
# Interactive — guided prompts for folder & threshold
imgdup

# Scan ~/Photos at 90% similarity, table output
imgdup -f ~/Photos -s 90

# Recursive scan, export CSV report
imgdup -f ~/Pictures -s 85 -r -o csv --out-file duplicates.csv

# Quiet mode, JSON output (useful for scripts)
imgdup -f ~/Downloads -s 95 -q -o json

# Disable colours (e.g. in CI or log files)
imgdup -f /mnt/media -s 90 --no-color
```

---

## How It Works

`imgdup` runs a three-phase pipeline:

```
Scan  →  Hash  →  Compare  →  Output
```

1. **Scan** — walks the folder and collects supported media files (skips hidden files and anything under 1 KB).

2. **Hash** — computes two fingerprints per file in parallel:
   - **MD5** — for exact byte-level duplicates.
   - **pHash** (perceptual hash) — a 64-bit fingerprint of visual content. Two images can look identical to the human eye while having different MD5s (different compression, resolution, or metadata). pHash catches these.
   - **Frame hashes** — for videos, key frames are extracted via `ffmpeg` and hashed individually.

3. **Compare** — groups files using a BK-tree for fast Hamming-distance search on pHashes. Similarity is `(1 − hamming_distance / 64) × 100`. Groups are sorted by recoverable space so the biggest wins appear first.

4. **Output** — renders the results as a table, JSON, or CSV. The `KEEP` label marks the largest file in each group as the recommended copy to keep.

**Hash cache** — results are stored in `~/.imgdup/cache.db` (bbolt). On the next run, unchanged files are read from cache — making repeated scans of large libraries fast.

---

## Requirements

| Dependency | Required for | Install |
|---|---|---|
| Go 1.21+ | Building from source | [go.dev/dl](https://go.dev/dl/) |
| ffmpeg | Video duplicate detection | `brew install ffmpeg` |

> Video support is optional. If `ffmpeg` is not found, `imgdup` skips video files and processes images only.

---

## Why imgdup?

Most duplicate-finder tools either:
- Only detect **exact** copies (byte-identical), missing compressed or resized versions.
- Require a GUI, making them hard to script or automate.
- Process files sequentially, making them slow on large libraries.

`imgdup` uses perceptual hashing to catch near-duplicates, runs fully in the terminal, and parallelises across all CPU cores — so it stays fast even on libraries with tens of thousands of files.

---

## License

MIT
