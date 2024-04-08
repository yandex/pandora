package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_replacePort(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int64
		want string
	}{
		{
			name: "zero port",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
			port: 0,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
		},
		{
			name: "replace ipv6",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:8888",
			port: 9999,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:9999",
		},
		{
			name: "add port to ipv6",
			host: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]",
			port: 9999,
			want: "[2a02:6b8:c02:901:0:fc5f:9a6c:4]:9999",
		},
		{
			name: "replace ipv4",
			host: "127.0.0.1:8888",
			port: 9999,
			want: "127.0.0.1:9999",
		},
		{
			name: "replace host",
			host: "localhost:8888",
			port: 9999,
			want: "localhost:9999",
		},
		{
			name: "add port",
			host: "localhost",
			port: 9999,
			want: "localhost:9999",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replacePort(tt.host, tt.port)
			require.Equal(t, tt.want, got)
		})
	}
}
