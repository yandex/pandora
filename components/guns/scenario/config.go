package scenario

//import "time"
//
//type ClientGunConfig struct {
//	Target string `validate:"endpoint,required"`
//	SSL    bool
//	Base   BaseGunConfig `config:",squash"`
//}
//
//type HTTPGunConfig struct {
//	Gun    ClientGunConfig `config:",squash"`
//	Client ClientConfig    `config:",squash"`
//}
//
//type HTTP2GunConfig struct {
//	Gun    ClientGunConfig `config:",squash"`
//	Client ClientConfig    `config:",squash"`
//}
//
//type ClientConfig struct {
//	Redirect  bool            // When true, follow HTTP redirects.
//	Dialer    DialerConfig    `config:"dial"`
//	Transport TransportConfig `config:",squash"`
//}
//
//func DefaultClientConfig() ClientConfig {
//	return ClientConfig{
//		Transport: DefaultTransportConfig(),
//		Dialer:    DefaultDialerConfig(),
//		Redirect:  false,
//	}
//}
//
//// TransportConfig can be mapped on http.Transport.
//// See http.Transport for details.
//type TransportConfig struct {
//	TLSHandshakeTimeout   time.Duration `config:"tls-handshake-timeout"`
//	DisableKeepAlives     bool          `config:"disable-keep-alives"`
//	DisableCompression    bool          `config:"disable-compression"`
//	MaxIdleConns          int           `config:"max-idle-conns"`
//	MaxIdleConnsPerHost   int           `config:"max-idle-conns-per-host"`
//	IdleConnTimeout       time.Duration `config:"idle-conn-timeout"`
//	ResponseHeaderTimeout time.Duration `config:"response-header-timeout"`
//	ExpectContinueTimeout time.Duration `config:"expect-continue-timeout"`
//}
//
//func DefaultTransportConfig() TransportConfig {
//	return TransportConfig{
//		MaxIdleConns:          0, // No limit.
//		IdleConnTimeout:       90 * time.Second,
//		TLSHandshakeTimeout:   1 * time.Second,
//		ExpectContinueTimeout: 1 * time.Second,
//		DisableCompression:    true,
//	}
//}
//
//// DialerConfig can be mapped on net.Dialer.
//// Set net.Dialer for details.
//type DialerConfig struct {
//	DNSCache bool `config:"dns-cache" map:"-"`
//
//	Timeout   time.Duration `config:"timeout"`
//	DualStack bool          `config:"dual-stack"`
//
//	// IPv4/IPv6 settings should not matter really,
//	// because target should be dialed using pre-resolved addr.
//	FallbackDelay time.Duration `config:"fallback-delay"`
//	KeepAlive     time.Duration `config:"keep-alive"`
//}
//
//func DefaultDialerConfig() DialerConfig {
//	return DialerConfig{
//		DNSCache:  true,
//		DualStack: true,
//		Timeout:   3 * time.Second,
//		KeepAlive: 120 * time.Second,
//	}
//}
//
//type BaseGunConfig struct {
//	AutoTag   AutoTagConfig   `config:"auto-tag"`
//	AnswLog   AnswLogConfig   `config:"answlog"`
//	HTTPTrace HTTPTraceConfig `config:"httptrace"`
//}
//
//// AutoTagConfig configure automatic tags generation based on ammo URI. First AutoTag URI path elements becomes tag.
//// Example: /my/very/deep/page?id=23&param=33 -> /my/very when uri-elements: 2.
//type AutoTagConfig struct {
//	Enabled     bool `config:"enabled"`
//	URIElements int  `config:"uri-elements" validate:"min=1"` // URI elements used to autotagging
//	NoTagOnly   bool `config:"no-tag-only"`                   // When true, autotagged only ammo that has no tag before.
//}
//
//type AnswLogConfig struct {
//	Enabled bool   `config:"enabled"`
//	Path    string `config:"path"`
//	Filter  string `config:"filter" valid:"oneof=all warning error"`
//}
//
//type HTTPTraceConfig struct {
//	DumpEnabled  bool `config:"dump"`
//	TraceEnabled bool `config:"trace"`
//}
//
//func DefaultBaseGunConfig() BaseGunConfig {
//	return BaseGunConfig{
//		AutoTagConfig{
//			Enabled:     false,
//			URIElements: 2,
//			NoTagOnly:   true,
//		},
//		AnswLogConfig{
//			Enabled: false,
//			Path:    "answ.log",
//			Filter:  "error",
//		},
//		HTTPTraceConfig{
//			DumpEnabled:  false,
//			TraceEnabled: false,
//		},
//	}
//}
