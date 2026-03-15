package output

import (
	"encoding/json"
	"io"

	"github.com/imgdup/image-dupl-detector/internal/comparator"
)

type jsonOutput struct {
	FilesScanned    int            `json:"files_scanned"`
	ScanTime        string         `json:"scan_time"`
	DuplicateGroups int            `json:"duplicate_groups"`
	DuplicateFiles  int            `json:"duplicate_files"`
	SpaceRecoverable int64         `json:"space_recoverable_bytes"`
	Groups          []jsonGroup    `json:"groups"`
}

type jsonGroup struct {
	ID         int       `json:"id"`
	Similarity float64   `json:"similarity_pct"`
	Files      []jsonFile `json:"files"`
}

type jsonFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size_bytes"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	ModTime  string `json:"modified"`
	IsKeep   bool   `json:"keep"`
}

func renderJSON(w io.Writer, groups []comparator.DuplicateGroup, stats Stats) error {
	out := jsonOutput{
		FilesScanned:     stats.FilesScanned,
		ScanTime:         stats.ScanTime,
		DuplicateGroups:  len(groups),
		DuplicateFiles:   comparator.DuplicateFileCount(groups),
		SpaceRecoverable: comparator.TotalRecoverableSpace(groups),
		Groups:           make([]jsonGroup, 0, len(groups)),
	}

	for _, g := range groups {
		jg := jsonGroup{
			ID:         g.ID,
			Similarity: g.Similarity,
			Files:      make([]jsonFile, 0, len(g.Files)),
		}
		for _, f := range g.Files {
			jg.Files = append(jg.Files, jsonFile{
				Path:    f.Info.Path,
				Size:    f.Info.Size,
				Width:   f.Info.Width,
				Height:  f.Info.Height,
				ModTime: f.Info.ModTime.Format("2006-01-02T15:04:05Z07:00"),
				IsKeep:  f.IsKeep,
			})
		}
		out.Groups = append(out.Groups, jg)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
