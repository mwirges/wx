//go:build tlssmoke

// Package tlssmoke dials wx upstreams with the process trust store.
// It is excluded from `go test ./...` unless -tags tlssmoke is set.
// `make tls-smoke` is the supported entry point and is not a default gate.
package tlssmoke

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"
)

// hosts matches the upstream clients covered by internal/tlsdial.
var hosts = []string{
	"api.weather.gov",
	"api.open-meteo.com",
	"air-quality-api.open-meteo.com",
	"data.rcc-acis.org",
	"mapservices.weather.noaa.gov",
	"nominatim.openstreetmap.org",
	"ipinfo.io",
	"www.spc.noaa.gov",
	"www.nhc.noaa.gov",
	"opengeo.ncep.noaa.gov",
	"mesonet.agron.iastate.edu",
}

func TestLiveSmoke(t *testing.T) {
	for _, host := range hosts {
		t.Run(host, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			d := tls.Dialer{
				Config: &tls.Config{
					ServerName: host,
					MinVersion: tls.VersionTLS12,
				},
			}
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, "443"))
			if err != nil {
				t.Fatalf("verify %s: %v", host, err)
			}
			defer conn.Close()
			state := conn.(*tls.Conn).ConnectionState()
			if len(state.PeerCertificates) == 0 {
				t.Fatalf("%s: handshake returned no peer certificate", host)
			}
			t.Logf("%s issuer %s", host, state.PeerCertificates[len(state.PeerCertificates)-1].Issuer)
		})
	}
}
