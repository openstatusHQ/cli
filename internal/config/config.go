package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"

	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

type TestsConfig struct {
	Ids []int `koanf:"ids"`
}

type Config struct {
	Tests TestsConfig `koanf:"tests"`
}

func ReadConfig(path string) (*Config, error) {
	k.Load(file.Provider(path), yaml.Parser())

	var out Config

	err := k.Unmarshal("", &out)
	if err != nil {
		return nil, err
	}

	return &out, nil

}
