package config

import (
	"errors"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Lock struct {
	Monitor Monitor `yaml:"monitor"`
	ID      int     `yaml:"id"`
}

type MonitorsLock map[string]Lock

func ReadLockFile(filename string) (MonitorsLock, error) {

	var out MonitorsLock
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return MonitorsLock{}, nil
	}

	file := file.Provider(filename)
	var k = koanf.New(".")

	err := k.Load(file, yaml.Parser())

	if err != nil {
		return nil, err
	}
	err = k.Unmarshal("", &out)
	if err != nil {
		return nil, err
	}

	for _, value := range out {
		ConvertAssertionTargets(value.Monitor.Assertions)
	}

	return out, nil

}
