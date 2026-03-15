package output

import (
	"io"
	"os"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
	"github.com/imgdup/image-dupl-detector/internal/config"
	"github.com/mattn/go-isatty"
)

// Stats holds summary statistics for a scan run.
type Stats struct {
	FilesScanned int
	ScanTime     string
}

// Render writes duplicate groups to stdout (and optionally to a file).
func Render(groups []comparator.DuplicateGroup, stats Stats, cfg *config.ScanConfig) error {
	// Detect TTY for color support
	useColor := !cfg.NoColor && isatty.IsTerminal(os.Stdout.Fd())

	// Write to stdout
	if err := renderTo(os.Stdout, groups, stats, cfg, useColor); err != nil {
		return err
	}

	// Optionally write to file (no ANSI codes)
	if cfg.OutputFile != "" {
		f, err := os.Create(cfg.OutputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := renderTo(f, groups, stats, cfg, false); err != nil {
			return err
		}
	}

	return nil
}

func renderTo(w io.Writer, groups []comparator.DuplicateGroup, stats Stats, cfg *config.ScanConfig, useColor bool) error {
	switch cfg.OutputFormat {
	case config.FormatJSON:
		return renderJSON(w, groups, stats)
	case config.FormatCSV:
		return renderCSV(w, groups, stats)
	default:
		return renderTable(w, groups, stats, useColor)
	}
}
