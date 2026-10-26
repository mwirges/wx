package tlsdial

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// defaultClientSites are production files that dial through http.DefaultClient.
var defaultClientSites = []struct {
	file string
	host string
}{
	{"internal/location/geocode.go", "nominatim.openstreetmap.org"},
	{"internal/location/ipgeo.go", "ipinfo.io"},
	{"internal/provider/nws/radar.go", "opengeo.ncep.noaa.gov"},
	{"internal/provider/nws/radar.go", "mesonet.agron.iastate.edu"},
	{"internal/provider/nws/radar.go", "api.weather.gov"},
}

func TestProductionClientsUseSystemRoots(t *testing.T) {
	if http.DefaultClient.Transport != nil {
		t.Fatalf("DefaultClient transport is %T", http.DefaultClient.Transport)
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("DefaultTransport is %T", http.DefaultTransport)
	}
	if cfg := base.TLSClientConfig; cfg != nil && (cfg.InsecureSkipVerify || cfg.RootCAs != nil) {
		t.Fatal("DefaultTransport does not use system roots")
	}
	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			c := up.client()
			if c == nil {
				t.Fatal("nil client")
			}
			if c.Transport != nil {
				t.Fatalf("transport %T; wx clients keep the default system roots", c.Transport)
			}
		})
	}
}

func TestDefaultClientCallSites(t *testing.T) {
	root := moduleRoot(t)
	for _, site := range defaultClientSites {
		body, err := os.ReadFile(filepath.Join(root, site.file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "http.DefaultClient") {
			t.Errorf("%s no longer uses http.DefaultClient", site.file)
		}
		if !strings.Contains(string(body), site.host) {
			t.Errorf("%s no longer names %s", site.file, site.host)
		}
	}
}

func TestProductionSourceUsesSystemRoots(t *testing.T) {
	root := moduleRoot(t)
	covered := map[string]bool{}
	for _, up := range upstreams {
		covered[up.host] = true
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "build", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ImportSpec:
				imp, err := strconv.Unquote(n.Path.Value)
				if err == nil && strings.Contains(imp, "x509roots/fallback") {
					t.Errorf("%s: imports %s; wx uses system roots", fset.Position(n.Pos()), imp)
				}
			case *ast.KeyValueExpr:
				if name := exprName(n.Key); name == "InsecureSkipVerify" || name == "RootCAs" {
					t.Errorf("%s: sets %s", fset.Position(n.Pos()), name)
				}
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					if name := exprName(lhs); name == "InsecureSkipVerify" || name == "RootCAs" {
						t.Errorf("%s: assigns %s", fset.Position(lhs.Pos()), name)
					}
				}
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					return true
				}
				s, err := strconv.Unquote(n.Value)
				if err != nil {
					return true
				}
				host := upstreamHost(s)
				if host != "" && !covered[host] {
					t.Errorf("%s: upstream host %s is not covered by the TLS client tests", rel, host)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func exprName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return ""
	}
}

func upstreamHost(s string) string {
	if !strings.Contains(s, "://") {
		return ""
	}
	rest := s[strings.Index(s, "://")+3:]
	rest = strings.SplitN(rest, "/", 2)[0]
	host := strings.SplitN(rest, ":", 2)[0]
	switch {
	case host == "ipinfo.io":
		return host
	case strings.HasSuffix(host, "weather.gov"),
		strings.HasSuffix(host, "noaa.gov"),
		strings.HasSuffix(host, "open-meteo.com"),
		strings.HasSuffix(host, "openstreetmap.org"),
		strings.HasSuffix(host, "iastate.edu"):
		return host
	default:
		return ""
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
