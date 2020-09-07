package config

type PlatformConfig struct {
	StorageConfig   StorageConfig   `yaml:"storage"`
	MessagingConfig MessagingConfig `yaml:"messaging"`
	APIServerConfig APIServerConfig `yaml:"api"`
	GatewayConfigs  []GatewayConfig `yaml:"gateways"`
}

type StorageConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type MessagingConfig struct {
	Type string `yaml:"type"`
}

type APIServerConfig struct {
	Port int `yaml:"port"`
}

type GatewayConfig struct {
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
	SslPort  int    `yaml:"sslPort,omitempty"`
}
