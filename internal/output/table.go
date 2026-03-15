package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/imgdup/image-dupl-detector/internal/comparator"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
)

func renderTable(w io.Writer, groups []comparator.DuplicateGroup, stats Stats, useColor bool) error {
	color.NoColor = !useColor

	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	faint := color.New(color.Faint)

	fprintf := func(format string, a ...interface{}) {
		fmt.Fprintf(w, format, a...)
	}

	if len(groups) == 0 {
		fprintf("\n")
		green.Fprintf(w, "  ✓ No duplicates found.\n")
		fprintf("\n")
		fprintf("  Files scanned:  %s\n", formatInt(stats.FilesScanned))
		fprintf("  Scan time:      %s\n", stats.ScanTime)
		fprintf("\n")
		return nil
	}

	// Calculate box width to fit the longest path + open hint without truncation
	minWidth := 50
	boxInner := minWidth
	for _, g := range groups {
		for _, f := range g.Files {
			pathLen := len([]rune(f.Info.Path)) + 4 // "    " indent
			hintLen := len([]rune(fmt.Sprintf(`open "%s"`, f.Info.Path))) + 4
			if pathLen > boxInner {
				boxInner = pathLen
			}
			if hintLen > boxInner {
				boxInner = hintLen
			}
		}
	}

	divider := strings.Repeat("─", boxInner+4)
	hbar := strings.Repeat("─", boxInner+2)

	fprintf("\n")
	cyan.Fprintf(w, "  Found %d duplicate group(s) across %s files\n", len(groups), formatInt(stats.FilesScanned))
	fprintf("  %s\n\n", divider)

	for _, g := range groups {
		cyan.Fprintf(w, "  GROUP %d OF %d", g.ID, len(groups))
		fprintf("  ·  %d files  ·  Similarity: %.1f%%\n", len(g.Files), g.Similarity)

		fprintf("  ┌%s┐\n", hbar)

		for i, f := range g.Files {
			if i > 0 {
				fprintf("  ├%s┤\n", hbar)
			}

			if f.IsKeep {
				green.Fprintf(w, "  │  ★ KEEP\n")
			} else {
				yellow.Fprintf(w, "  │  DUPLICATE\n")
			}

			fprintf("  │    %s\n", f.Info.Path)
			fprintf("  │    %s\n", formatMeta(f.Info))
			faint.Fprintf(w, "  │    open \"%s\"\n", f.Info.Path)
		}

		fprintf("  └%s┘\n\n", hbar)
	}

	fprintf("  %s\n", divider)
	cyan.Fprintf(w, "  SUMMARY\n\n")
	fprintf("    Files scanned:     %s\n", formatInt(stats.FilesScanned))
	fprintf("    Duplicate groups:  %d\n", len(groups))
	fprintf("    Duplicate files:   %d\n", comparator.DuplicateFileCount(groups))
	fprintf("    Space recoverable: %s\n", formatBytes(comparator.TotalRecoverableSpace(groups)))
	fprintf("    Scan time:         %s\n", stats.ScanTime)
	fprintf("\n")
	fprintf("  What next?\n\n")
	faint.Fprintf(w, "    • Review each group above before deleting anything\n")
	faint.Fprintf(w, "    • Run `open <path>` to preview a file in Finder/Preview\n")
	fprintf("\n")
	green.Fprintf(w, "  ✓ Done. No files were modified.\n")
	fprintf("  %s\n", divider)

	return nil
}

func formatMeta(fi scanner.FileInfo) string {
	parts := []string{formatBytes(fi.Size)}
	if fi.Width > 0 && fi.Height > 0 {
		parts = append(parts, fmt.Sprintf("%d×%d", fi.Width, fi.Height))
	}
	if fi.Duration > 0 {
		parts = append(parts, formatDuration(fi.Duration))
	}
	if !fi.ModTime.IsZero() {
		parts = append(parts, fi.ModTime.Format("2006-01-02"))
	}
	return strings.Join(parts, "  ·  ")
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
