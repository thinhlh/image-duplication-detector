package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imgdup/image-dupl-detector/internal/config"
)

// MediaType classifies a file as image or video.
type MediaType string

const (
	MediaImage MediaType = "image"
	MediaVideo MediaType = "video"
)

// FileInfo holds metadata about a discovered media file.
type FileInfo struct {
	Path      string
	MediaType MediaType
	Size      int64
	ModTime   time.Time
	Width     int
	Height    int
	Duration  time.Duration
}

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".tiff": true, ".tif": true, ".bmp": true,
}

var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
	".m4v": true, ".wmv": true, ".webm": true, ".3gp": true,
}

// Scan walks the folder described in cfg and emits FileInfo for each
// supported media file. Returns channels for results and non-fatal errors.
// The result channel is closed when the walk completes or ctx is cancelled.
func Scan(ctx context.Context, cfg *config.ScanConfig) (<-chan FileInfo, <-chan error) {
	out := make(chan FileInfo, 256)
	errCh := make(chan error, 64)

	go func() {
		defer close(out)
		defer close(errCh)

		walkFn := func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				// Non-fatal: log and continue
				select {
				case errCh <- err:
				default:
				}
				return nil
			}

			if d.IsDir() {
				// Skip hidden directories
				if strings.HasPrefix(d.Name(), ".") && path != cfg.Folder {
					return filepath.SkipDir
				}
				// If not recursive, skip subdirectories
				if !cfg.Recursive && path != cfg.Folder {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip hidden files
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(d.Name()))
			var mediaType MediaType

			if imageExts[ext] {
				mediaType = MediaImage
			} else if videoExts[ext] && cfg.VideoSupported {
				mediaType = MediaVideo
			} else {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return nil
			}

			// Skip very small files (< 1KB)
			if info.Size() < 1024 {
				return nil
			}

			fi := FileInfo{
				Path:      path,
				MediaType: mediaType,
				Size:      info.Size(),
				ModTime:   info.ModTime(),
			}

			select {
			case out <- fi:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}

		_ = filepath.WalkDir(cfg.Folder, walkFn)
	}()

	return out, errCh
}

// CountFiles does a quick walk to count total files for the progress bar.
// Must mirror the same skip logic as Scan() to produce an accurate count.
func CountFiles(ctx context.Context, cfg *config.ScanConfig) int {
	count := 0
	_ = filepath.WalkDir(cfg.Folder, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip hidden directories (same as Scan)
			if strings.HasPrefix(d.Name(), ".") && path != cfg.Folder {
				return filepath.SkipDir
			}
			// Skip subdirectories when not recursive (same as Scan)
			if !cfg.Recursive && path != cfg.Folder {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if imageExts[ext] || (videoExts[ext] && cfg.VideoSupported) {
			info, err := d.Info()
			if err == nil && info.Size() >= 1024 {
				count++
			}
		}
		return nil
	})
	return count
}

// IsImageExt returns true if the extension belongs to a supported image format.
func IsImageExt(ext string) bool {
	return imageExts[strings.ToLower(ext)]
}

// IsVideoExt returns true if the extension belongs to a supported video format.
func IsVideoExt(ext string) bool {
	return videoExts[strings.ToLower(ext)]
}

// StatFile fills in os-level metadata for a FileInfo.
func StatFile(path string) (int64, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	return info.Size(), info.ModTime(), nil
}
