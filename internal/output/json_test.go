package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
)

func TestRenderJSON_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/tmp/a.jpg", 40000, true),
			makeFile("/tmp/b.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 4, ScanTime: "0.1s"}

	if err := renderJSON(&buf, groups, stats); err != nil {
		t.Fatalf("renderJSON error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, buf.String())
	}
}

func TestRenderJSON_ContainsExpectedFields(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0,
			makeFile("/a.jpg", 40000, true),
			makeFile("/b.jpg", 38000, false),
		),
	}
	stats := Stats{FilesScanned: 5, ScanTime: "0.5s"}

	_ = renderJSON(&buf, groups, stats)

	out := buf.String()
	for _, field := range []string{"files_scanned", "duplicate_groups", "groups", "similarity_pct", "path", "keep"} {
		if !strings.Contains(out, field) {
			t.Errorf("JSON output missing field: %s", field)
		}
	}
}

func TestRenderJSON_NoGroups(t *testing.T) {
	var buf bytes.Buffer
	stats := Stats{FilesScanned: 10, ScanTime: "0.2s"}

	if err := renderJSON(&buf, nil, stats); err != nil {
		t.Fatalf("renderJSON error: %v", err)
	}

	var out jsonOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.DuplicateGroups != 0 {
		t.Errorf("expected 0 groups, got %d", out.DuplicateGroups)
	}
	if out.FilesScanned != 10 {
		t.Errorf("expected 10 files scanned, got %d", out.FilesScanned)
	}
}

func TestRenderJSON_NoAnsiCodes(t *testing.T) {
	var buf bytes.Buffer
	groups := []comparator.DuplicateGroup{
		makeGroup(1, 97.0, makeFile("/a.jpg", 40000, true), makeFile("/b.jpg", 38000, false)),
	}
	_ = renderJSON(&buf, groups, Stats{FilesScanned: 2, ScanTime: "0.1s"})

	if strings.Contains(buf.String(), "\033[") {
		t.Error("JSON output contains ANSI escape codes")
	}
}
