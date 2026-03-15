package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/imgdup/image-dupl-detector/internal/config"
)

// Run performs all pre-scan dependency and permission checks.
// It modifies cfg in-place to reflect available capabilities.
// Returns an error only for fatal conditions that prevent scanning.
func Run(cfg *config.ScanConfig) error {
	// 1. Validate folder
	info, err := os.Stat(cfg.Folder)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("folder not found: %s", cfg.Folder)
		}
		return fmt.Errorf("cannot access folder %s: %w", cfg.Folder, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", cfg.Folder)
	}

	// 2. Check folder is readable
	entries, err := os.ReadDir(cfg.Folder)
	if err != nil {
		return fmt.Errorf("cannot read folder %s: %w", cfg.Folder, err)
	}
	_ = entries

	// 3. Check similarity range (defensive, already validated in config)
	if cfg.SimilarityPct < 1 || cfg.SimilarityPct > 100 {
		return fmt.Errorf("similarity must be between 1 and 100")
	}

	// 4. Check ffmpeg availability
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Fprintln(os.Stderr, "[WARN] ffmpeg not found — video files will be skipped.")
		fmt.Fprintln(os.Stderr, "       Install with: brew install ffmpeg")
		cfg.VideoSupported = false
	} else {
		cfg.VideoSupported = true
	}

	// 5. Check bbolt cache availability — just ensure the directory is creatable
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[WARN] Cannot determine home directory — scanning without cache.")
		cfg.CacheEnabled = false
	} else {
		cacheDir := filepath.Join(home, ".imgdup")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "[WARN] Cannot create cache directory — scanning without cache.")
			cfg.CacheEnabled = false
		} else {
			cfg.CacheEnabled = true
		}
	}

	return nil
}
