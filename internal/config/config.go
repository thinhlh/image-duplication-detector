package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OutputFormat defines the output rendering format.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// ScanConfig holds all user-provided settings for a scan run.
type ScanConfig struct {
	Folder         string
	Recursive      bool
	SimilarityPct  int
	OutputFormat   OutputFormat
	OutputFile     string
	NoColor        bool
	Quiet          bool
	VideoSupported bool // set by startup after checking for ffmpeg
	CacheEnabled   bool // set by startup after checking bbolt
}

// DefaultConfig returns a ScanConfig populated with sensible defaults.
func DefaultConfig() *ScanConfig {
	return &ScanConfig{
		SimilarityPct: 90,
		OutputFormat:  FormatTable,
		CacheEnabled:  true,
	}
}

// Validate checks that all fields are within acceptable ranges.
func (c *ScanConfig) Validate() error {
	if c.Folder == "" {
		return errors.New("folder path is required")
	}

	abs, err := expandPath(c.Folder)
	if err != nil {
		return fmt.Errorf("invalid folder path: %w", err)
	}
	c.Folder = abs

	info, err := os.Stat(c.Folder)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("folder not found: %s", c.Folder)
		}
		return fmt.Errorf("cannot access folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", c.Folder)
	}

	if c.SimilarityPct < 1 || c.SimilarityPct > 100 {
		return fmt.Errorf("similarity must be between 1 and 100, got %d", c.SimilarityPct)
	}

	switch c.OutputFormat {
	case FormatTable, FormatJSON, FormatCSV:
		// valid
	default:
		return fmt.Errorf("output format must be table, json, or csv, got %q", c.OutputFormat)
	}

	return nil
}

// expandPath expands ~ and resolves the path to absolute.
func expandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}
