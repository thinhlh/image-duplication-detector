package hasher

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/imgdup/image-dupl-detector/internal/cache"
	"github.com/imgdup/image-dupl-detector/internal/config"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
	"github.com/schollz/progressbar/v3"
)

// HashResult contains the computed hashes and metadata for a file.
type HashResult struct {
	FileInfo    scanner.FileInfo
	MD5         string
	PHash       uint64
	FrameHashes []uint64 // non-nil for videos
	FromCache   bool
}

// Hash processes FileInfo from the in channel using a worker pool.
// Returns a channel of HashResults and a channel of non-fatal errors.
func Hash(ctx context.Context, in <-chan scanner.FileInfo, cfg *config.ScanConfig, total int, c *cache.Cache) (<-chan HashResult, <-chan error) {
	out := make(chan HashResult, 256)
	errCh := make(chan error, 64)

	numImageWorkers := runtime.NumCPU()
	numVideoWorkers := runtime.NumCPU() / 2
	if numVideoWorkers < 1 {
		numVideoWorkers = 1
	}
	if numVideoWorkers > 4 {
		numVideoWorkers = 4
	}

	imageCh := make(chan scanner.FileInfo, 64)
	videoCh := make(chan scanner.FileInfo, 16)

	// Progress bar on stderr
	var bar *progressbar.ProgressBar
	if !cfg.Quiet && total > 0 {
		bar = progressbar.NewOptions(total,
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSetDescription("Scanning files..."),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerPadding: "░",
				BarStart:      "[",
				BarEnd:        "]",
			}),
			progressbar.OptionClearOnFinish(),
		)
	}

	advanceBar := func() {
		if bar != nil {
			_ = bar.Add(1)
		}
	}

	// Dispatcher: routes incoming FileInfo to the right worker pool
	go func() {
		defer close(imageCh)
		defer close(videoCh)
		for fi := range in {
			select {
			case <-ctx.Done():
				return
			default:
			}
			switch fi.MediaType {
			case scanner.MediaImage:
				select {
				case imageCh <- fi:
				case <-ctx.Done():
					return
				}
			case scanner.MediaVideo:
				if cfg.VideoSupported {
					select {
					case videoCh <- fi:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	var wg sync.WaitGroup

	// Image workers
	for i := 0; i < numImageWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fi := range imageCh {
				select {
				case <-ctx.Done():
					return
				default:
				}
				result, err := processImage(fi, cfg, c)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("[WARN] %s: %w", fi.Path, err):
					default:
					}
					advanceBar()
					continue
				}
				select {
				case out <- result:
				case <-ctx.Done():
					return
				}
				advanceBar()
			}
		}()
	}

	// Video workers
	for i := 0; i < numVideoWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fi := range videoCh {
				select {
				case <-ctx.Done():
					return
				default:
				}
				result, err := processVideo(fi, cfg, c)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("[WARN] %s: %w", fi.Path, err):
					default:
					}
					advanceBar()
					continue
				}
				select {
				case out <- result:
				case <-ctx.Done():
					return
				}
				advanceBar()
			}
		}()
	}

	// Closer: waits for all workers then closes output channels
	go func() {
		wg.Wait()
		if bar != nil {
			_ = bar.Finish()
		}
		close(out)
		close(errCh)
	}()

	return out, errCh
}

func processImage(fi scanner.FileInfo, cfg *config.ScanConfig, c *cache.Cache) (HashResult, error) {
	cacheKey := cache.Key(fi.Path, fi.ModTime)

	// Check cache
	if cfg.CacheEnabled && c != nil {
		if entry := c.Get(cacheKey); entry != nil {
			// Fill in image dimensions
			fi.Width, fi.Height = imageSize(fi.Path)
			return HashResult{
				FileInfo:  fi,
				MD5:       entry.MD5,
				PHash:     entry.PHash,
				FromCache: true,
			}, nil
		}
	}

	md5hex, phash, err := hashImage(fi.Path)
	if err != nil {
		return HashResult{}, err
	}

	fi.Width, fi.Height = imageSize(fi.Path)

	if cfg.CacheEnabled && c != nil {
		c.Put(cacheKey, cache.Entry{
			MD5:   md5hex,
			PHash: phash,
		})
	}

	return HashResult{
		FileInfo: fi,
		MD5:      md5hex,
		PHash:    phash,
	}, nil
}

func processVideo(fi scanner.FileInfo, cfg *config.ScanConfig, c *cache.Cache) (HashResult, error) {
	if !cfg.VideoSupported {
		return HashResult{}, fmt.Errorf("video support not available (ffmpeg not found)")
	}

	cacheKey := cache.Key(fi.Path, fi.ModTime)

	if cfg.CacheEnabled && c != nil {
		if entry := c.Get(cacheKey); entry != nil {
			return HashResult{
				FileInfo:    fi,
				MD5:         entry.MD5,
				FrameHashes: entry.FrameHashes,
				FromCache:   true,
			}, nil
		}
	}

	md5hex, frameHashes, err := hashVideo(fi.Path)
	if err != nil {
		return HashResult{}, err
	}

	if cfg.CacheEnabled && c != nil {
		c.Put(cacheKey, cache.Entry{
			MD5:         md5hex,
			FrameHashes: frameHashes,
		})
	}

	return HashResult{
		FileInfo:    fi,
		MD5:         md5hex,
		FrameHashes: frameHashes,
	}, nil
}
