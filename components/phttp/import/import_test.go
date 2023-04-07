package phttp

import (
	"net"
	"strconv"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	phttp "github.com/yandex/pandora/components/guns/http"
)

func TestImport_NotPanics(t *testing.T) {
	require.NotPanics(t, func() {
		Import(afero.NewOsFs())
	})
}

func Test_preResolveTargetAddr(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", nil)
	if listener != nil {
		defer listener.Close()
	}
	require.NoError(t, err)

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	tests := []struct {
		name           string
		target         string
		wantTargetAddr string
		wantDNSCache   bool
		wantErr        bool
	}{
		{
			name:           "ip target",
			target:         "localhost:" + port,
			wantTargetAddr: "127.0.0.1:" + port,
			wantDNSCache:   false,
			wantErr:        false,
		},
		{
			name:           "ip target",
			target:         "127.0.0.1:80",
			wantTargetAddr: "127.0.0.1:80",
			wantDNSCache:   false,
			wantErr:        false,
		},
		{
			name:           "failed",
			target:         "localhost:54321",
			wantTargetAddr: "localhost:54321",
			wantDNSCache:   true,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &phttp.ClientConfig{}
			conf.Dialer.DNSCache = true

			got, err := PreResolveTargetAddr(conf, tt.target)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantTargetAddr, got)
			require.Equal(t, tt.wantDNSCache, conf.Dialer.DNSCache)
		})
	}
}
