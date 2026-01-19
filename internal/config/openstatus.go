package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

type Monitors map[string]Monitor

func ReadOpenStatus(path string) (Monitors, error) {
	f := file.Provider(path)

	err := k.Load(f, yaml.Parser())

	if err != nil {
		return nil, err
	}

	var out Monitors

	err = k.Unmarshal("", &out)

	if err != nil {
		return nil, err
	}

	for _, value := range out {
		ConvertAssertionTargets(value.Assertions)
	}

	return out, nil
}

func ParseConfigMonitorsToMonitor(monitors Monitors) []Monitor {
	var monitor []Monitor
	for _, value := range monitors {
		ConvertAssertionTargets(value.Assertions)
		monitor = append(monitor, value)
	}

	return monitor
}
