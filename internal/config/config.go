package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type TestsConfig struct {
	Ids []int `koanf:"ids"`
}

type Config struct {
	Tests TestsConfig `koanf:"tests"`
}

func ReadConfig(path string) (*Config, error) {
	k := koanf.New(".")

	f := file.Provider(path)
	if err := k.Load(f, yaml.Parser()); err != nil {
		return nil, err
	}

	var out Config
	if err := k.Unmarshal("", &out); err != nil {
		return nil, err
	}

	return &out, nil
}
