package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func LoadConfig() (*PlatformConfig, error) {
	return LoadConfigFromFile("./config.yaml")
}

func LoadConfigFromFile(filename string) (*PlatformConfig, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := PlatformConfig{}
	err = yaml.Unmarshal(content, &config)

	return &config, err
}
