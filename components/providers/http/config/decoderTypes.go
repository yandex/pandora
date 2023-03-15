package config

type DecoderType string

const (
	DecoderURI      DecoderType = "uri"
	DecoderURIPost  DecoderType = "uripost"
	DecoderRaw      DecoderType = "raw"
	DecoderJSONLine DecoderType = "jsonline"
)

func (d DecoderType) IsValid() bool {
	switch d {
	case DecoderURI, DecoderURIPost, DecoderRaw, DecoderJSONLine:
		return true
	}
	return false
}
