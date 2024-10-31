package config

import "github.com/knadh/koanf/v2"

var k = koanf.New(".")

type TestsConfig struct {
	Ids []int `koanf:"ids"`
}

type Config struct {
	Tests TestsConfig `koanf:"tests"`
}
