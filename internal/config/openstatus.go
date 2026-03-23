package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Monitors map[string]Monitor

func ReadOpenStatus(path string) (Monitors, error) {
	k := koanf.New(".")

	f := file.Provider(path)
	if err := k.Load(f, yaml.Parser()); err != nil {
		return nil, err
	}

	var out Monitors
	if err := k.Unmarshal("", &out); err != nil {
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
