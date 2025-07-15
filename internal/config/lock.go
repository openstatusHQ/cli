package config

import (
	"errors"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

var lock = "openstatus.lock"

func ReadLockFile() (*Config, error) {

	var out Config


	if _, err := os.Stat(lock); errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}

	file := file.Provider(lock)

	err := k.Load(file, yaml.Parser())

	if err != nil {
		return nil, err
	}


	err = k.Unmarshal("", &out)
	if err != nil {
		return nil, err
	}

	return &out, nil

}
