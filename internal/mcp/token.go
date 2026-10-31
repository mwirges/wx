package mcp

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mwirges/wx/internal/config"
)

const tokenFileName = "mcp.token"

// DefaultTokenPath is mcp.token beside the config file.
func DefaultTokenPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), tokenFileName)
}

// GenerateToken returns 32 random bytes as 64 hex characters.
// The value is suitable for an Authorization header and contains no spaces.
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("mcp: token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// WriteTokenFile stores token at path with mode 0600. The file is replaced
// atomically. Symlinks are not followed: a temp file in the same directory
// is renamed over the destination entry. The token is not logged.
func WriteTokenFile(path, token string) error {
	token = strings.TrimSpace(token)
	if err := validateToken(token); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mcp: token dir: %w", err)
	}
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return errors.New("mcp: token file must not be a symlink")
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("mcp: token temp: %w", err)
	}
	_, werr := io.WriteString(f, token+"\n")
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp)
		if werr != nil {
			return werr
		}
		return cerr
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Chmod(path, 0o600)
}

// LoadTokenFile reads a single-line token. The file must be mode 0600
// (no group or world permission bits) and must not be a symlink.
// The returned string is the secret; callers must not log it.
func LoadTokenFile(path string) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("mcp: token file: %w", err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("mcp: token file must not be a symlink")
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("mcp: token file must be a regular file")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("mcp: token file %s is not mode 0600 (mode %#o); refusing to use it", path, st.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("mcp: token file: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if err := validateToken(token); err != nil {
		return "", err
	}
	return token, nil
}

func validateToken(token string) error {
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return errors.New("mcp: require_token is set but the token is missing or empty")
	}
	if len(token) < 16 {
		return errors.New("mcp: token must be at least 16 characters")
	}
	if len(token) > 256 {
		return errors.New("mcp: token is too long")
	}
	return nil
}

// LoadServeAuth resolves the bearer for one serve. require_token false
// returns no token and does not read a file. The token value is not placed
// in the error text.
func LoadServeAuth(configPath string, mcpCfg config.MCPSettings, tokenFileFlag string) (require bool, token string, err error) {
	if !mcpCfg.RequireToken {
		return false, "", nil
	}
	path := strings.TrimSpace(tokenFileFlag)
	if path == "" {
		path = strings.TrimSpace(mcpCfg.TokenFile)
	}
	if path == "" {
		path = DefaultTokenPath(configPath)
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(configPath), path)
	}
	tok, err := LoadTokenFile(path)
	if err != nil {
		return true, "", fmt.Errorf("mcp: require_token is set: %w", err)
	}
	return true, tok, nil
}

// tokenGuard rejects /mcp and /healthz (and any other route on this server)
// when require is set and the bearer does not match. The response is 401
// with S6 code "unauthorized". The body never echoes the header or the
// expected token. Host/Origin must already have been checked by guard.
func tokenGuard(require bool, want string, next http.Handler) http.Handler {
	if !require {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := bearerToken(r.Header.Get("Authorization"))
		if !tokenMatches(got, want) {
			writeUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	const prefix = "bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func tokenMatches(got, want string) bool {
	if got == "" || want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("WWW-Authenticate", `Bearer realm="wx-mcp"`)
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "unauthorized",
		"message": "bearer token required",
	})
}
