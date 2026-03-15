package output

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
)

func renderCSV(w io.Writer, groups []comparator.DuplicateGroup, stats Stats) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header
	if err := cw.Write([]string{
		"group_id", "similarity_pct", "keep", "path", "size_bytes",
		"width", "height", "modified",
	}); err != nil {
		return err
	}

	for _, g := range groups {
		for _, f := range g.Files {
			keepStr := "false"
			if f.IsKeep {
				keepStr = "true"
			}
			row := []string{
				fmt.Sprintf("%d", g.ID),
				fmt.Sprintf("%.1f", g.Similarity),
				keepStr,
				f.Info.Path,
				fmt.Sprintf("%d", f.Info.Size),
				fmt.Sprintf("%d", f.Info.Width),
				fmt.Sprintf("%d", f.Info.Height),
				f.Info.ModTime.Format("2006-01-02T15:04:05Z07:00"),
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
