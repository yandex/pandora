package answlog

type Config struct {
	Enabled  bool     `config:"enabled"`
	Path     string   `config:"path"`
	Filter   string   `config:"filter" valid:"oneof=all warning error"`
	Sampling Sampling `config:"sampling"`
}

type Sampling struct {
	Enabled bool `config:"enabled"`
	Pattern any  `config:"pattern"`
}
