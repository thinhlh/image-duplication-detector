package comparator

import (
	"testing"
	"time"

	"github.com/imgdup/image-dupl-detector/internal/hasher"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
)

func makeN(n int) []hasher.HashResult {
	results := make([]hasher.HashResult, n)
	for i := range results {
		results[i] = hasher.HashResult{
			FileInfo: scanner.FileInfo{
				Path:      "/img/file.jpg",
				MediaType: scanner.MediaImage,
				Size:      int64(1000 + i),
				ModTime:   time.Now(),
			},
			MD5:   "unique",
			PHash: deterministicHash(uint64(i)),
		}
	}
	return results
}

func BenchmarkCompare_1k(b *testing.B) {
	data := makeN(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compare(data, 90)
	}
}

func BenchmarkCompare_5k(b *testing.B) {
	data := makeN(5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compare(data, 90)
	}
}

func BenchmarkResultAccum_NoPrealloc(b *testing.B) {
	total := 10000
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var results []hasher.HashResult
		for j := 0; j < total; j++ {
			results = append(results, hasher.HashResult{})
		}
		_ = results
	}
}

func BenchmarkResultAccum_Prealloc(b *testing.B) {
	total := 10000
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		results := make([]hasher.HashResult, 0, total)
		for j := 0; j < total; j++ {
			results = append(results, hasher.HashResult{})
		}
		_ = results
	}
}
