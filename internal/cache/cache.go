package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Cache is a file-based JSON cache.
// On macOS and Linux, New stores it in ~/.cache/wx/.
// On iOS the app container root rejects ~/.cache, so New uses WX_CACHE_DIR
// when the mobile bridge sets it via WxSetCacheDir (os.Setenv on the Go
// side). Without that override, iOS falls back to UserCacheDir/wx.
// Use New() for a real cache, NewNoOp() to disable caching.
type Cache struct {
	dir  string
	noop bool
}

type envelope struct {
	Expires time.Time       `json:"expires"`
	Data    json.RawMessage `json:"data"`
}

// New returns a Cache, creating the directory if needed.
// WX_CACHE_DIR, when set to an absolute path, wins. The iOS app sets it
// through WxSetCacheDir at startup (not libc setenv alone).
// Without it, iOS uses UserCacheDir/wx. Other systems keep ~/.cache/wx.
func New() (*Cache, error) {
	dir, err := resolveDir(runtime.GOOS)
	if err != nil {
		return nil, err
	}
	return NewWithDir(dir)
}

// resolveDir picks the cache root. goos is runtime.GOOS except in tests.
func resolveDir(goos string) (string, error) {
	if dir := strings.TrimSpace(os.Getenv("WX_CACHE_DIR")); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("cache: WX_CACHE_DIR must be absolute")
		}
		return dir, nil
	}
	if goos == "ios" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("cache: cache dir: %w", err)
		}
		return filepath.Join(base, "wx"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cache: home dir: %w", err)
	}
	return filepath.Join(home, ".cache", "wx"), nil
}

// NewWithDir returns a Cache rooted at the given directory (useful in tests).
func NewWithDir(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cache: mkdir %s: %w", dir, err)
	}
	return &Cache{dir: dir}, nil
}

// NewNoOp returns a Cache that never stores or retrieves anything.
func NewNoOp() *Cache {
	return &Cache{noop: true}
}

// Get deserializes cached data into v. Returns false if the entry is absent or expired.
func (c *Cache) Get(key string, v any) bool {
	if c.noop {
		return false
	}
	data, err := os.ReadFile(c.keyToPath(key))
	if err != nil {
		return false
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return false
	}
	if time.Now().After(env.Expires) {
		return false
	}
	return json.Unmarshal(env.Data, v) == nil
}

// Set serializes v with the given TTL and writes it to the cache.
func (c *Cache) Set(key string, v any, ttl time.Duration) error {
	if c.noop {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	env := envelope{
		Expires: time.Now().Add(ttl),
		Data:    raw,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("cache: marshal envelope: %w", err)
	}
	return os.WriteFile(c.keyToPath(key), data, 0o644)
}

func (c *Cache) keyToPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, fmt.Sprintf("%x.json", h))
}
