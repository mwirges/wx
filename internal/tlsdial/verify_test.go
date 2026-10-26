// The module's go 1.26.10 line keeps the pre-1.27 default, where SSL_CERT_FILE
// does not override the platform verifier. This test-only setting opts into the
// Go 1.27 behavior so a production client with a nil TLS config trusts the
// test CA written by TestMain. It is not compiled into the wx binary.
//go:debug x509sslcertoverrideplatform=1

package tlsdial

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/mwirges/wx/internal/airquality"
	"github.com/mwirges/wx/internal/chase"
	"github.com/mwirges/wx/internal/climate"
	"github.com/mwirges/wx/internal/cpc"
	"github.com/mwirges/wx/internal/history"
	"github.com/mwirges/wx/internal/nowcast"
	"github.com/mwirges/wx/internal/provider/nws"
	"github.com/mwirges/wx/internal/provider/openmeteo"
	"github.com/mwirges/wx/internal/sounding"
	"github.com/mwirges/wx/internal/spc"
	"github.com/mwirges/wx/internal/tropics"
)

// testCA is the only trust anchor for this process. TestMain writes it to
// SSL_CERT_FILE before any handshake. On Go 1.27 that variable replaces the
// platform verifier (macOS Security.framework, Windows CryptoAPI), so a
// production client with a nil transport still verifies against this CA and
// nothing else. The dial redirect below only moves TCP to the httptest
// listener; it does not set RootCAs or InsecureSkipVerify.
var testCA certAuth

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "wx-ca-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ca, err := newCA("wx test ca")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testCA = ca
	path := dir + "/ca.pem"
	if err := os.WriteFile(path, ca.pem, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// An empty directory keeps a developer SSL_CERT_DIR from mixing extra
	// roots into the pool. Go loads SSL_CERT_FILE once, on first use.
	if err := os.Setenv("SSL_CERT_FILE", path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("SSL_CERT_DIR", dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// upstream is one production HTTP client and a host it dials.
type upstream struct {
	name   string
	host   string
	client func() *http.Client
}

// upstreams is every wx client that dials an upstream weather or geo host.
// DefaultClient entries are the call sites in geocode, ipgeo, and nws radar.
var upstreams = []upstream{
	{"nws", "api.weather.gov", func() *http.Client { return httpClientOf(nws.New()) }},
	{"chase", "api.weather.gov", func() *http.Client { return httpClientOf(chase.NewClient("")) }},
	{"spc-watches", "api.weather.gov", func() *http.Client { return httpClientOf(spc.NewClient()) }},
	{"radar-nws", "api.weather.gov", func() *http.Client { return http.DefaultClient }},
	{"open-meteo", "api.open-meteo.com", func() *http.Client { return httpClientOf(openmeteo.New()) }},
	{"nowcast", "api.open-meteo.com", func() *http.Client { return httpClientOf(nowcast.NewClient("")) }},
	{"history", "api.open-meteo.com", func() *http.Client { return httpClientOf(history.NewClient("")) }},
	{"sounding-open-meteo", "api.open-meteo.com", func() *http.Client { return httpClientOf(sounding.NewClient("", "")) }},
	{"air-quality", "air-quality-api.open-meteo.com", func() *http.Client { return httpClientOf(airquality.NewClient("")) }},
	{"rcc-acis", "data.rcc-acis.org", func() *http.Client { return httpClientOf(climate.NewClient("")) }},
	{"spc-mapservices", "mapservices.weather.noaa.gov", func() *http.Client { return httpClientOf(spc.NewClient()) }},
	{"cpc", "mapservices.weather.noaa.gov", func() *http.Client { return httpClientOf(cpc.NewClient()) }},
	{"nominatim", "nominatim.openstreetmap.org", func() *http.Client { return http.DefaultClient }},
	{"ipinfo", "ipinfo.io", func() *http.Client { return http.DefaultClient }},
	{"sounding-spc", "www.spc.noaa.gov", func() *http.Client { return httpClientOf(sounding.NewClient("", "")) }},
	{"spc-md", "www.spc.noaa.gov", func() *http.Client { return httpClientOf(spc.NewClient()) }},
	{"nhc", "www.nhc.noaa.gov", func() *http.Client { return httpClientOf(tropics.NewClient(nil)) }},
	{"opengeo", "opengeo.ncep.noaa.gov", func() *http.Client { return http.DefaultClient }},
	{"iem", "mesonet.agron.iastate.edu", func() *http.Client { return http.DefaultClient }},
}

func TestSSLCertFileOverridesPlatformVerifier(t *testing.T) {
	leaf := mustLeaf(t, "example.com", testCA)
	parsed, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsed.Verify(x509.VerifyOptions{DNSName: "example.com"}); err != nil {
		t.Fatalf("SSL_CERT_FILE=%s did not become the process trust store: %v", os.Getenv("SSL_CERT_FILE"), err)
	}
}

func TestUpstreamClientTrustsTestCA(t *testing.T) {
	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			addr := serveTLS(t, mustLeaf(t, up.host, testCA))
			assertOK(t, up.client(), up.host, addr)
		})
	}
}

func TestUpstreamClientRejectsUntrustedCert(t *testing.T) {
	other, err := newCA("wx untrusted ca")
	if err != nil {
		t.Fatal(err)
	}
	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			addr := serveTLS(t, mustLeaf(t, up.host, other))
			assertUnknownAuthority(t, up.client(), up.host, addr)
		})
	}
}

func TestUpstreamClientRejectsSelfSignedCert(t *testing.T) {
	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			addr := serveTLS(t, mustSelfSigned(t, up.host))
			assertUnknownAuthority(t, up.client(), up.host, addr)
		})
	}
}

func TestUpstreamClientRejectsWrongHost(t *testing.T) {
	for _, up := range upstreams {
		t.Run(up.name, func(t *testing.T) {
			addr := serveTLS(t, mustLeaf(t, "wrong.example", testCA))
			_, err := doRequest(t, up.client(), up.host, addr)
			var hostErr x509.HostnameError
			if !errors.As(err, &hostErr) {
				t.Fatalf("host %s: want hostname mismatch, got %v", up.host, err)
			}
		})
	}
}

func assertOK(t *testing.T, client *http.Client, host, addr string) {
	t.Helper()
	resp, err := doRequest(t, client, host, addr)
	if err != nil {
		t.Fatalf("host %s: trusted test CA was rejected: %v", host, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("host %s: status %d body %q", host, resp.StatusCode, body)
	}
}

func assertUnknownAuthority(t *testing.T, client *http.Client, host, addr string) {
	t.Helper()
	_, err := doRequest(t, client, host, addr)
	var unknown x509.UnknownAuthorityError
	if !errors.As(err, &unknown) {
		t.Fatalf("host %s: want unknown authority, got %v", host, err)
	}
}

// doRequest performs one GET with the production client. The only test change
// on the transport is DialContext, so the handshake is local, plus Proxy
// disabled so HTTP_PROXY cannot leave the machine. TLSClientConfig stays nil
// and verification uses the SSL_CERT_FILE pool.
func doRequest(t *testing.T, client *http.Client, host, addr string) (*http.Response, error) {
	t.Helper()
	if client == nil {
		t.Fatal("nil http.Client")
	}
	if client.Transport != nil {
		t.Fatalf("host %s: production client has transport %T", host, client.Transport)
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("DefaultTransport is %T", http.DefaultTransport)
	}
	tr := base.Clone()
	if cfg := tr.TLSClientConfig; cfg != nil && (cfg.InsecureSkipVerify || cfg.RootCAs != nil) {
		t.Fatal("DefaultTransport does not use system roots")
	}
	tr.TLSClientConfig = nil
	tr.Proxy = nil
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
	prev := client.Transport
	client.Transport = tr
	defer func() { client.Transport = prev }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/ok", nil)
	if err != nil {
		t.Fatal(err)
	}
	return client.Do(req)
}

func serveTLS(t *testing.T, cert tls.Certificate) string {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

type certAuth struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

func newCA(commonName string) (certAuth, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return certAuth{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return certAuth{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return certAuth{}, err
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return certAuth{}, err
	}
	return certAuth{
		cert: parsed,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}, nil
}

func mustLeaf(t *testing.T, host string, ca certAuth) tls.Certificate {
	t.Helper()
	cert, err := leafCert(host, ca.cert, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func mustSelfSigned(t *testing.T, host string) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := leafTemplate(host)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func leafCert(host string, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl, err := leafTemplate(host)
	if err != nil {
		return tls.Certificate{}, err
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, parentKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func leafTemplate(host string) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	return &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, nil
}

var httpClientPtr = reflect.TypeOf((*http.Client)(nil))

func httpClientOf(root any) *http.Client {
	found := collectClients(reflect.ValueOf(root), map[uintptr]bool{})
	if len(found) != 1 {
		panic(fmt.Sprintf("%T contains %d *http.Client, want 1", root, len(found)))
	}
	return found[0]
}

func collectClients(v reflect.Value, seen map[uintptr]bool) []*http.Client {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return collectClients(v.Elem(), seen)
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		if v.Type() == httpClientPtr {
			return []*http.Client{exportClient(v)}
		}
		if seen[v.Pointer()] {
			return nil
		}
		seen[v.Pointer()] = true
		return collectClients(v.Elem(), seen)
	case reflect.Struct:
		var out []*http.Client
		for i := 0; i < v.NumField(); i++ {
			out = append(out, collectClients(v.Field(i), seen)...)
		}
		return out
	default:
		return nil
	}
}

func exportClient(v reflect.Value) *http.Client {
	if v.CanInterface() {
		return v.Interface().(*http.Client)
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(*http.Client)
}
