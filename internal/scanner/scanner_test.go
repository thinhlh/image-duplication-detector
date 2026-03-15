package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/imgdup/image-dupl-detector/internal/config"
)

func makeTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Write files large enough to pass the 1KB threshold
	padding := make([]byte, 2048)

	writeFile := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("photo.jpg", append([]byte("JFIF"), padding...))
	writeFile("image.png", append([]byte("PNG"), padding...))
	writeFile("clip.mp4", append([]byte("MP4"), padding...))
	writeFile("doc.pdf", padding)         // unsupported
	writeFile("small.jpg", []byte("tiny")) // < 1KB, should be skipped

	// Hidden file
	writeFile(".hidden.jpg", append([]byte("JFIF"), padding...))

	// Subdirectory
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.jpg"), append([]byte("JFIF"), padding...), 0644); err != nil {
		t.Fatal(err)
	}

	// Hidden subdirectory
	hiddenSub := filepath.Join(dir, ".hidden_dir")
	if err := os.Mkdir(hiddenSub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenSub, "inside.jpg"), append([]byte("JFIF"), padding...), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func collectFiles(t *testing.T, cfg *config.ScanConfig) []FileInfo {
	t.Helper()
	ctx := context.Background()
	ch, _ := Scan(ctx, cfg)
	var files []FileInfo
	for f := range ch {
		files = append(files, f)
	}
	return files
}

func TestScan_NonRecursive(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: false,
		SimilarityPct:  90,
	}

	files := collectFiles(t, cfg)

	// Should find: photo.jpg, image.png — NOT small.jpg (too small), NOT doc.pdf, NOT .hidden.jpg,
	// NOT nested.jpg (subdirectory), NOT inside.jpg (hidden subdirectory)
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), filePaths(files))
	}
}

func TestScan_Recursive(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      true,
		VideoSupported: false,
		SimilarityPct:  90,
	}

	files := collectFiles(t, cfg)

	// Should find: photo.jpg, image.png, nested.jpg (sub/) — not inside hidden dir
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), filePaths(files))
	}
}

func TestScan_VideoSupported(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: true,
		SimilarityPct:  90,
	}

	files := collectFiles(t, cfg)

	// Should find: photo.jpg, image.png, clip.mp4
	if len(files) != 3 {
		t.Errorf("expected 3 files (including video), got %d: %v", len(files), filePaths(files))
	}
}

func TestScan_SkipsSmallFiles(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: false,
		SimilarityPct:  90,
	}
	files := collectFiles(t, cfg)
	for _, f := range files {
		if filepath.Base(f.Path) == "small.jpg" {
			t.Error("small.jpg should have been skipped (< 1KB)")
		}
	}
}

func TestScan_SkipsHiddenFiles(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: false,
		SimilarityPct:  90,
	}
	files := collectFiles(t, cfg)
	for _, f := range files {
		base := filepath.Base(f.Path)
		if len(base) > 0 && base[0] == '.' {
			t.Errorf("hidden file %s should not be scanned", base)
		}
	}
}

func TestScan_SkipsUnsupportedTypes(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: false,
		SimilarityPct:  90,
	}
	files := collectFiles(t, cfg)
	for _, f := range files {
		if filepath.Ext(f.Path) == ".pdf" {
			t.Error("PDF files should not be scanned")
		}
	}
}

func TestCountFiles_MatchesScan(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      true,
		VideoSupported: false,
		SimilarityPct:  90,
	}

	ctx := context.Background()
	count := CountFiles(ctx, cfg)
	files := collectFiles(t, cfg)

	if count != len(files) {
		t.Errorf("CountFiles=%d, but Scan produced %d files — counts must match", count, len(files))
	}
}

func TestScan_MediaTypeClassification(t *testing.T) {
	dir := makeTestDir(t)
	cfg := &config.ScanConfig{
		Folder:         dir,
		Recursive:      false,
		VideoSupported: true,
		SimilarityPct:  90,
	}
	files := collectFiles(t, cfg)
	for _, f := range files {
		switch filepath.Ext(f.Path) {
		case ".jpg", ".png":
			if f.MediaType != MediaImage {
				t.Errorf("%s: expected image type, got %s", f.Path, f.MediaType)
			}
		case ".mp4":
			if f.MediaType != MediaVideo {
				t.Errorf("%s: expected video type, got %s", f.Path, f.MediaType)
			}
		}
	}
}

func filePaths(files []FileInfo) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = filepath.Base(f.Path)
	}
	return paths
}
