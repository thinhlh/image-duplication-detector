package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const bucketName = "hashes"

// Entry is the cached representation of a file's hash data.
type Entry struct {
	MD5         string   `json:"md5"`
	PHash       uint64   `json:"phash"`
	FrameHashes []uint64 `json:"frame_hashes,omitempty"`
}

// Cache wraps a bbolt database for hash persistence.
type Cache struct {
	db       *bolt.DB
	writeCh  chan writeReq
	doneCh   chan struct{}
}

type writeReq struct {
	key   string
	entry Entry
}

// Open opens (or creates) the bbolt cache database.
func Open() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determine home dir: %w", err)
	}
	cacheDir := filepath.Join(home, ".imgdup")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	dbPath := filepath.Join(cacheDir, "cache.db")
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}

	// Ensure bucket exists
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	c := &Cache{
		db:      db,
		writeCh: make(chan writeReq, 128),
		doneCh:  make(chan struct{}),
	}

	// Start single writer goroutine
	go c.writer()

	return c, nil
}

// Key returns the cache key for a file path and modification time.
func Key(path string, modTime time.Time) string {
	return fmt.Sprintf("%s|%d", path, modTime.Unix())
}

// Get retrieves a cached entry. Returns nil if not found.
func (c *Cache) Get(key string) *Entry {
	var entry *Entry
	_ = c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(key))
		if data == nil {
			return nil
		}
		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			return nil
		}
		entry = &e
		return nil
	})
	return entry
}

// Put queues a cache write (non-blocking, handled by writer goroutine).
func (c *Cache) Put(key string, entry Entry) {
	select {
	case c.writeCh <- writeReq{key: key, entry: entry}:
	default:
		// Drop if channel full — cache is best-effort
	}
}

// Close flushes pending writes and closes the database.
func (c *Cache) Close() {
	close(c.writeCh)
	<-c.doneCh
	c.db.Close()
}

// writer is the single goroutine that handles all bbolt writes.
func (c *Cache) writer() {
	defer close(c.doneCh)

	batch := make([]writeReq, 0, 32)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		_ = c.db.Batch(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucketName))
			if b == nil {
				return nil
			}
			for _, req := range batch {
				data, err := json.Marshal(req.entry)
				if err != nil {
					continue
				}
				_ = b.Put([]byte(req.key), data)
			}
			return nil
		})
		batch = batch[:0]
	}

	for req := range c.writeCh {
		batch = append(batch, req)
		if len(batch) >= 32 {
			flush()
		}
	}
	flush()
}
