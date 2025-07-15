package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

type Monitors map[string]Monitor

func ReadOpenStatus(path string) ([]Monitor, error) {
	f := file.Provider(path)

	err := k.Load(f, yaml.Parser())

	if err != nil {
		return nil, err
	}

	var out Monitors

	err = k.Unmarshal("", &out)

	for _, value := range out {
		for _, assertion := range value.Assertions {
			if assertion.Kind == Header || assertion.Kind == TextBody {
				assertion.Target = assertion.Target.(string)
			}
			if assertion.Kind == StatusCode {
				assertion.Target = assertion.Target.(int)
			}
		}
	}

	var monitor []Monitor
	for _, value := range out {
		for _, assertion := range value.Assertions {
			if assertion.Kind == Header || assertion.Kind == TextBody {
				assertion.Target = assertion.Target.(string)
			}
			if assertion.Kind == StatusCode {
				assertion.Target = assertion.Target.(int)
			}
		}
		monitor = append(monitor, value)
	}

	if err != nil {
		return nil, err
	}

	return monitor, nil
}
