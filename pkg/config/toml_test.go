package config

import (
	"testing"
)

func TestWriteConfigFile(t *testing.T) {
	defaultConfig := DefaultConfig()
	err := WriteConfigFile("./audiusd_test.toml", defaultConfig)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
}
