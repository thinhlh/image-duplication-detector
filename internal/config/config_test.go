package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SimilarityPct != 90 {
		t.Errorf("expected default similarity 90, got %d", cfg.SimilarityPct)
	}
	if cfg.OutputFormat != FormatTable {
		t.Errorf("expected default format table, got %s", cfg.OutputFormat)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Folder = t.TempDir()
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidate_MissingFolder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Folder = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty folder, got nil")
	}
}

func TestValidate_NonExistentFolder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Folder = "/this/path/does/not/exist/xyz"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for non-existent folder")
	}
}

func TestValidate_FileNotDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Folder = "/etc/hosts"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when folder is a file")
	}
}

func TestValidate_SimilarityRange(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		pct     int
		wantErr bool
	}{
		{1, false},
		{90, false},
		{100, false},
		{0, true},
		{101, true},
		{-1, true},
	}
	for _, tc := range cases {
		cfg := DefaultConfig()
		cfg.Folder = dir
		cfg.SimilarityPct = tc.pct
		err := cfg.Validate()
		if tc.wantErr && err == nil {
			t.Errorf("similarity=%d: expected error, got nil", tc.pct)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("similarity=%d: unexpected error: %v", tc.pct, err)
		}
	}
}

func TestValidate_OutputFormat(t *testing.T) {
	dir := t.TempDir()
	for _, fmt := range []OutputFormat{FormatTable, FormatJSON, FormatCSV} {
		cfg := DefaultConfig()
		cfg.Folder = dir
		cfg.OutputFormat = fmt
		if err := cfg.Validate(); err != nil {
			t.Errorf("format=%s: unexpected error: %v", fmt, err)
		}
	}

	cfg := DefaultConfig()
	cfg.Folder = dir
	cfg.OutputFormat = "xml"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid format 'xml'")
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	path, err := expandPath("~/something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "~/something" {
		t.Error("~ was not expanded")
	}
	if len(path) == 0 || path[0] != '/' {
		t.Errorf("expected absolute path, got: %s", path)
	}
}
