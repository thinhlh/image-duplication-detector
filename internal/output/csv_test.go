package output

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
)

func TestRenderCSV_ValidCSV(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/tmp/a.jpg", 40000, true),
			makeFile("/tmp/b.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 4, ScanTime: "0.1s"}

	if err := renderCSV(&buf, groups, stats); err != nil {
		t.Fatalf("renderCSV error: %v", err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}

	// Header + 2 data rows
	if len(records) != 3 {
		t.Errorf("expected 3 rows (header + 2 files), got %d", len(records))
	}
}

func TestRenderCSV_Header(t *testing.T) {
	var buf bytes.Buffer
	_ = renderCSV(&buf, nil, Stats{})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 1 {
		t.Fatal("no output")
	}
	header := lines[0]
	for _, col := range []string{"group_id", "similarity_pct", "keep", "path", "size_bytes"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV header missing column: %s", col)
		}
	}
}

func TestRenderCSV_CorrectColumnCount(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0, makeFile("/a.jpg", 40000, true), makeFile("/b.jpg", 38000, false)),
	}
	_ = renderCSV(&buf, groups, Stats{})

	r := csv.NewReader(&buf)
	records, _ := r.ReadAll()

	expectedCols := 8 // group_id, similarity_pct, keep, path, size_bytes, width, height, modified
	for i, row := range records {
		if len(row) != expectedCols {
			t.Errorf("row %d: expected %d columns, got %d", i, expectedCols, len(row))
		}
	}
}

func TestRenderCSV_KeepField(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 100.0,
			makeFile("/keep.jpg", 50000, true),
			makeFile("/dup.jpg", 48000, false),
		),
	}
	_ = renderCSV(&buf, groups, Stats{})

	r := csv.NewReader(&buf)
	records, _ := r.ReadAll()

	// records[0] = header, records[1] = keep, records[2] = dup
	if len(records) < 3 {
		t.Fatal("not enough rows")
	}
	keepCol := 2 // 0-indexed: group_id, similarity_pct, keep
	if records[1][keepCol] != "true" {
		t.Errorf("expected keep=true for first file, got %q", records[1][keepCol])
	}
	if records[2][keepCol] != "false" {
		t.Errorf("expected keep=false for second file, got %q", records[2][keepCol])
	}
}
