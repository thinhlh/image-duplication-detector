package hasher

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"strconv"

	"github.com/corona10/goimagehash"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// hashVideo extracts 5 keyframes and returns per-frame pHashes.
// Returns an error if ffmpeg is not available.
func hashVideo(path string) (md5hex string, frameHashes []uint64, err error) {
	// MD5 of the raw file
	md5hex, err = fileMD5(path)
	if err != nil {
		return "", nil, fmt.Errorf("md5 %s: %w", path, err)
	}

	// Get video duration in seconds
	duration, err := videoDuration(path)
	if err != nil {
		// If we can't get duration, sample at fixed intervals
		duration = 60.0
	}

	// Sample at 5%, 25%, 50%, 75%, 95%
	offsets := []float64{0.05, 0.25, 0.50, 0.75, 0.95}
	frameHashes = make([]uint64, 0, len(offsets))

	for _, pct := range offsets {
		ts := duration * pct
		h, err := extractFrameHash(path, ts)
		if err != nil {
			// Skip this frame if extraction fails
			continue
		}
		frameHashes = append(frameHashes, h)
	}

	if len(frameHashes) == 0 {
		return "", nil, fmt.Errorf("could not extract any frames from %s", path)
	}

	return md5hex, frameHashes, nil
}

// extractFrameHash extracts a single frame at the given timestamp (seconds)
// and returns its pHash.
func extractFrameHash(path string, ts float64) (uint64, error) {
	buf := bytes.NewBuffer(nil)

	err := ffmpeg.Input(path, ffmpeg.KwArgs{"ss": strconv.FormatFloat(ts, 'f', 2, 64)}).
		Output("pipe:", ffmpeg.KwArgs{
			"vframes": 1,
			"format":  "image2",
			"vcodec":  "mjpeg",
			"q:v":     "2",
		}).
		WithOutput(buf).
		Silent(true).
		Run()
	if err != nil {
		return 0, fmt.Errorf("ffmpeg extract frame at %.2fs: %w", ts, err)
	}

	img, err := jpeg.Decode(buf)
	if err != nil {
		return 0, fmt.Errorf("decode frame jpeg: %w", err)
	}

	h, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return 0, fmt.Errorf("phash frame: %w", err)
	}

	return h.GetHash(), nil
}

// videoDuration returns the duration of a video in seconds using ffprobe.
func videoDuration(path string) (float64, error) {
	buf := bytes.NewBuffer(nil)

	err := ffmpeg.Input(path).
		Output("pipe:", ffmpeg.KwArgs{
			"format": "null",
		}).
		WithOutput(buf).
		Silent(true).
		Run()
	_ = err // ffmpeg to null always "fails" (no output file), ignore

	// Use ffprobe instead
	out, err := ffmpeg.Probe(path)
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	// Parse duration from ffprobe JSON output
	// ffprobe returns JSON like: {"format":{"duration":"123.456"}}
	const durationKey = `"duration":"`
	idx := indexOf(out, durationKey)
	if idx == -1 {
		return 0, fmt.Errorf("duration not found in ffprobe output")
	}
	start := idx + len(durationKey)
	end := indexOf(out[start:], `"`)
	if end == -1 {
		return 0, fmt.Errorf("malformed ffprobe duration")
	}
	durStr := out[start : start+end]
	dur, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", durStr, err)
	}
	return dur, nil
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
