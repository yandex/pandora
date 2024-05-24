package acceptance

type PandoraConfigLog struct {
	Level string `yaml:"level"`
}
type PandoraConfigMonitoringExpVar struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}
type PandoraConfigMonitoring struct {
	ExpVar PandoraConfigMonitoringExpVar `yaml:"expvar"`
}
type PandoraConfigGRPCGun struct {
	Type            string            `yaml:"type"`
	Target          string            `yaml:"target"`
	TLS             bool              `yaml:"tls"`
	ReflectPort     *int64            `yaml:"reflect_port,omitempty"`
	ReflectMetadata map[string]string `yaml:"reflect_metadata,omitempty"`
	SharedClient    struct {
		ClientNumber int  `yaml:"client-number,omitempty"`
		Enabled      bool `yaml:"enabled"`
	} `yaml:"shared-client,omitempty"`
}
type PandoraConfigAmmo struct {
	Type string `yaml:"type"`
	File string `yaml:"file"`
}
type PandoraConfigResult struct {
	Type string `yaml:"type"`
}
type PandoraConfigRps struct {
	Type     string `yaml:"type"`
	Duration string `yaml:"duration"`
	Ops      int    `yaml:"ops"`
}
type PandoraConfigStartup struct {
	Type  string `yaml:"type"`
	Times int    `yaml:"times"`
}
type PandoraConfigGRPCPool struct {
	ID      string               `yaml:"id"`
	Gun     PandoraConfigGRPCGun `yaml:"gun"`
	Ammo    PandoraConfigAmmo    `yaml:"ammo"`
	Result  PandoraConfigResult  `yaml:"result"`
	Rps     []PandoraConfigRps   `yaml:"rps"`
	Startup PandoraConfigStartup `yaml:"startup"`
}
type PandoraConfigGRPC struct {
	Pools      []PandoraConfigGRPCPool  `yaml:"pools"`
	Log        *PandoraConfigLog        `yaml:"log"`
	Monitoring *PandoraConfigMonitoring `yaml:"monitoring"`
}
