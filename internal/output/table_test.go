package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
)

func makeGroup(id int, similarity float64, files ...comparator.RankedFile) comparator.DuplicateGroup {
	return comparator.DuplicateGroup{
		ID:         id,
		Files:      files,
		Similarity: similarity,
	}
}

func makeFile(path string, size int64, keep bool) comparator.RankedFile {
	return comparator.RankedFile{
		Info: scanner.FileInfo{
			Path:    path,
			Size:    size,
			Width:   100,
			Height:  100,
			ModTime: time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
		},
		IsKeep: keep,
	}
}

func TestRenderTable_NoColor_NoAnsiCodes(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/tmp/img_a.jpg", 40000, true),
			makeFile("/tmp/img_a_copy.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 4, ScanTime: "0.1s"}

	if err := renderTable(&buf, groups, stats, false); err != nil {
		t.Fatalf("renderTable error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Error("output contains ANSI escape codes when color is disabled")
	}
}

func TestRenderTable_ContainsKeepAndDuplicate(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/tmp/img_a.jpg", 40000, true),
			makeFile("/tmp/img_a_copy.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 4, ScanTime: "0.1s"}

	_ = renderTable(&buf, groups, stats, false)
	out := buf.String()

	if !strings.Contains(out, "★ KEEP") {
		t.Error("output missing KEEP marker")
	}
	if !strings.Contains(out, "DUPLICATE") {
		t.Error("output missing DUPLICATE label")
	}
}

func TestRenderTable_ContainsOpenHint(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/tmp/photo.jpg", 40000, true),
			makeFile("/tmp/photo_copy.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 2, ScanTime: "0.1s"}

	_ = renderTable(&buf, groups, stats, false)
	out := buf.String()

	if !strings.Contains(out, `open "/tmp/photo.jpg"`) {
		t.Error("output missing open hint for KEEP file")
	}
	if !strings.Contains(out, `open "/tmp/photo_copy.jpg"`) {
		t.Error("output missing open hint for DUPLICATE file")
	}
}

func TestRenderTable_ContainsFullAbsolutePaths(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/Users/alice/Photos/img.jpg", 40000, true),
			makeFile("/Users/alice/Photos/img_copy.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 2, ScanTime: "0.1s"}

	_ = renderTable(&buf, groups, stats, false)
	out := buf.String()

	if !strings.Contains(out, "/Users/alice/Photos/img.jpg") {
		t.Error("full absolute path not found in output")
	}
}

func TestRenderTable_NoDuplicates(t *testing.T) {
	var buf bytes.Buffer
	stats := Stats{FilesScanned: 10, ScanTime: "0.5s"}

	_ = renderTable(&buf, nil, stats, false)
	out := buf.String()

	if !strings.Contains(out, "No duplicates found") {
		t.Errorf("expected 'No duplicates found' message, got:\n%s", out)
	}
}

func TestRenderTable_SummaryStats(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 100.0,
			makeFile("/a.jpg", 50000, true),
			makeFile("/b.jpg", 48000, false),
		),
	}
	stats := Stats{FilesScanned: 5, ScanTime: "1.2s"}

	_ = renderTable(&buf, groups, stats, false)
	out := buf.String()

	if !strings.Contains(out, "SUMMARY") {
		t.Error("output missing SUMMARY section")
	}
	if !strings.Contains(out, "No files were modified") {
		t.Error("output missing safety notice")
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"日本語テスト", 5, "日本..."}, // multi-byte runes: max=5 → 2 chars + "..."
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range cases {
		got := formatBytes(tc.bytes)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestFormatInt(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1204, "1,204"},
		{1000000, "1,000,000"},
	}
	for _, tc := range cases {
		got := formatInt(tc.n)
		if got != tc.want {
			t.Errorf("formatInt(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
