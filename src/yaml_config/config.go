package yaml_config

import (
	"fmt"
	"strconv"
)

type IConfig interface {
	GetStringByKey(string) (string, error)
}

type Config struct {
	yml map[string]string
}

func NewConfig(m map[string]string) (*Config, error) {
	return &Config{m}, nil
}

func (config *Config) GetStringByKey(key string) (string, error) {
	if val, ok := config.yml[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("key does not exist")
}

func (config *Config) GetIntByKey(key string) (int, error) {
	if val, ok := config.yml[key]; ok {
		res, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}
		return res, nil
	}
	return 0, fmt.Errorf("key does not exist")
}
