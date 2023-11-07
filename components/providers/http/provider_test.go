package http

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	tmpFile, err := fs.Create("ammo")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("  {}\n\n")) // content is important only for jsonDecoder
	require.NoError(t, err)

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
			provdr, err := NewProvider(fs, tc.conf)
			if err != nil {
				t.Fatalf("failed to create provider: %s", err)
			}

			p, ok := provdr.(*provider.Provider)
			require.True(t, ok)
			require.NotNil(t, p)
			defer func() {
				err := p.Close()
				require.NoError(t, err)
			}()

			assert.NotNil(t, p.Sink)
			assert.Equal(t, fs, p.FS)
			assert.Equal(t, tc.expected, p.Config.Decoder)
			assert.Equal(t, tc.filePath, p.Config.File)

			if p.Decoder == nil && tc.expected != "" {
				t.Error("decoder is nil")
			}
		})
	}
}
