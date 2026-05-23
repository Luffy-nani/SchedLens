package appconfig

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Poller  PollerConfig  `yaml:"poller"`
	Metrics MetricsConfig `yaml:"metrics"`
	Healer  HealerConfig  `yaml:"healer"`
}

type PollerConfig struct {
	Interval string `yaml:"interval"`
}

type MetricsConfig struct {
	StarvationThresholdMs uint64 `yaml:"starvation_threshold_ms"`
	CpuDeltaThreshold     uint64 `yaml:"cpu_delta_threshold"`
	StarvationTicks       int    `yaml:"starvation_ticks"`
}

type HealerConfig struct {
	Enabled         bool `yaml:"enabled"`
	ReniceValue     int  `yaml:"renice_value"`
	CooldownSeconds int  `yaml:"cooldown_seconds"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) // We are reading the config.yml file
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
