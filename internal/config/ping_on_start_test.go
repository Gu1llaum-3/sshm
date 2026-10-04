package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-7 (#73): hosts are pinged at startup unless config.json says otherwise,
// and the key is not written to the file of users who don't set it
func TestPingOnStart_AC7(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		path := useTempConfigDir(t)
		appConfig, err := LoadAppConfig()
		if err != nil {
			t.Fatalf("LoadAppConfig() error = %v", err)
		}
		if !appConfig.IsPingOnStart() {
			t.Error("IsPingOnStart() = false without config, want true")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("default config.json not written: %v", err)
		}
		if strings.Contains(string(content), "ping_on_start") {
			t.Errorf("default config.json contains the key:\n%s", content)
		}
	})

	for _, tt := range []struct {
		content string
		want    bool
	}{
		{`{"ping_on_start": false}`, false},
		{`{"ping_on_start": true}`, true},
		{`{"key_bindings": {"quit_keys": ["q"]}}`, true},
	} {
		t.Run(tt.content, func(t *testing.T) {
			path := useTempConfigDir(t)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			appConfig, err := LoadAppConfig()
			if err != nil {
				t.Fatalf("LoadAppConfig() error = %v", err)
			}
			if got := appConfig.IsPingOnStart(); got != tt.want {
				t.Errorf("IsPingOnStart() = %v, want %v", got, tt.want)
			}
		})
	}

	var nilConfig *AppConfig
	if !nilConfig.IsPingOnStart() {
		t.Error("IsPingOnStart() on a nil config = false, want true")
	}
}
