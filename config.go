package main

type GlobalConfig struct{}

type AmmoProviderConfig struct{}

type GunConfig struct{}

type ResultListenerConfig struct{}

type LimiterConfig struct {
	LimiterType string
	Parameters  map[string]interface{}
}

type CompositeLimiterConfig struct {
	Steps []LimiterConfig
}

type UserConfig struct {
	Name           string
	Gun            *GunConfig
	AmmoProvider   *AmmoProviderConfig
	ResultListener *ResultListenerConfig
	Limiter        *LimiterConfig
}

type UserPoolConfig struct {
	Name           string
	Gun            *GunConfig
	AmmoProvider   *AmmoProviderConfig
	ResultListener *ResultListenerConfig
	UserLimiter    *LimiterConfig
	StartupLimiter *LimiterConfig
	UserConfig     *UserConfig
}
