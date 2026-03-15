package hasher

import (
	"crypto/md5"
	"fmt"
	"image"
	"io"
	"os"

	"github.com/corona10/goimagehash"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // WebP decoder
)

// hashImage computes the MD5 and pHash for an image file.
func hashImage(path string) (md5hex string, phash uint64, err error) {
	// MD5 pass
	md5hex, err = fileMD5(path)
	if err != nil {
		return "", 0, fmt.Errorf("md5 %s: %w", path, err)
	}

	// Decode image with EXIF auto-orientation
	img, err := imaging.Open(path, imaging.AutoOrientation(true))
	if err != nil {
		return "", 0, fmt.Errorf("decode %s: %w", path, err)
	}

	// Compute pHash
	h, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return "", 0, fmt.Errorf("phash %s: %w", path, err)
	}

	return md5hex, h.GetHash(), nil
}

// imageSize returns the pixel dimensions of an image file.
func imageSize(path string) (width, height int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// fileMD5 computes the hex MD5 checksum of a file.
func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
