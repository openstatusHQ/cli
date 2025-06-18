package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

type Monitors map[string]Monitor

func ReadOpenStatus(path string) (Monitors, error) {
	f := file.Provider(path)

	// r, _:= f.ReadBytes()

	// fmt.Printf("%v", string(r))
	// for _, line := range string(r) {
	// 	fmt.Println(line)
	// }
	err := k.Load(f, yaml.Parser())

	if err != nil {
		return nil, err
	}

	var out Monitors

	err = k.Unmarshal("", &out)

	return out, nil
}

func TranslateMonitors(monitors Monitors) []Monitor {
	var monitor []Monitor
	for _, value := range monitors {
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

	return monitor
}
