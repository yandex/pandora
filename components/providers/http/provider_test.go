package http

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/provider"
)

func TestNewProvider_invalidDecoder(t *testing.T) {
	fs := afero.NewMemMapFs()
	conf := config.Config{
		Decoder: "invalid",
	}

	p, err := NewProvider(fs, conf)
	if p != nil || err == nil {
		t.Error("expected error when creating provider with invalid decoder type")
	}
}

func TestNewProvider(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("InvalidDecoder", func(t *testing.T) {
	})

	tmpFile, err := fs.Create("ammo")
	if err != nil {
		t.Fatalf("failed to create temp file: %s", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("GET / HTTP/1.1\nHost: example.com\n\n")); err != nil {
		t.Fatalf("failed to write data to temp file: %s", err)
	}

	cases := []struct {
		name     string
		conf     config.Config
		expected config.DecoderType
		filePath string
	}{
		{
			name:     "URIDecoder",
			conf:     config.Config{Decoder: config.DecoderURI, Uris: []string{"http://example.com"}},
			expected: config.DecoderURI,
			filePath: "",
		},
		{
			name:     "FileDecoder",
			conf:     config.Config{Decoder: config.DecoderURI, File: tmpFile.Name()},
			expected: config.DecoderURI,
			filePath: tmpFile.Name(),
		},
		{
			name:     "DecoderURIPost",
			conf:     config.Config{Decoder: config.DecoderURIPost, File: tmpFile.Name()},
			expected: config.DecoderURIPost,
			filePath: tmpFile.Name(),
		},
		{
			name:     "DecoderRaw",
			conf:     config.Config{Decoder: config.DecoderRaw, File: tmpFile.Name()},
			expected: config.DecoderRaw,
			filePath: tmpFile.Name(),
		},
		{
			name:     "DecoderJSONLine",
			conf:     config.Config{Decoder: config.DecoderJSONLine, File: tmpFile.Name()},
			expected: config.DecoderJSONLine,
			filePath: tmpFile.Name(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			providr, err := NewProvider(fs, tc.conf)
			if err != nil {
				t.Fatalf("failed to create provider: %s", err)
			}

			provider, _ := providr.(*provider.Provider)

			defer func() {
				if err := provider.Close(); err != nil {
					t.Fatalf("failed to close provider: %s", err)
				}
			}()

			if provider == nil {
				t.Fatal("provider is nil")
			}

			if provider.Config.Decoder != tc.expected {
				t.Errorf("unexpected decoder type: got %s, want %s", provider.Config.Decoder, tc.expected)
			}
			if provider.Config.File != tc.filePath {
				t.Errorf("unexpected file path: got %s, want %s", provider.Config.File, tc.filePath)
			}

			if provider.Decoder == nil && tc.expected != "" {
				t.Error("decoder is nil")
			}

			if provider.FS != fs {
				t.Errorf("unexpected FS: got %v, want %v", provider.FS, fs)
			}

			if provider.Sink == nil {
				t.Error("sink is nil")
			}
		})
	}

}
