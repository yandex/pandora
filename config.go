package main

type GlobalConfig struct{}

type AmmoProviderConfig struct{}

type GunConfig struct{}

type ScheduleStepConfig struct {
	StepType   string
	Parameters map[string]int
}

type LimiterConfig struct {
	Schedule []ScheduleStep
}

type UserConfig struct {
	GunConfig GunConfig
}

type UserPoolConfig struct {
	StartupSchedule LimiterConfig
	UserConfig      UserConfig
}
