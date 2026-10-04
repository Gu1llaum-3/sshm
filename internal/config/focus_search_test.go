package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTempConfigDir points the sshm config directory to a temporary one
func useTempConfigDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	path, err := GetAppConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// AC-6: the search has the focus at startup unless config.json says otherwise,
// and the key is not written to the file of users who don't set it
func TestFocusSearchOnStart_AC6(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		path := useTempConfigDir(t)
		appConfig, err := LoadAppConfig()
		if err != nil {
			t.Fatalf("LoadAppConfig() error = %v", err)
		}
		if !appConfig.IsFocusSearchOnStart() {
			t.Error("IsFocusSearchOnStart() = false without config, want true")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("default config.json not written: %v", err)
		}
		if strings.Contains(string(content), "focus_search_on_start") {
			t.Errorf("default config.json contains the key:\n%s", content)
		}
	})

	for _, tt := range []struct {
		content string
		want    bool
	}{
		{`{"focus_search_on_start": false}`, false},
		{`{"focus_search_on_start": true}`, true},
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
			if got := appConfig.IsFocusSearchOnStart(); got != tt.want {
				t.Errorf("IsFocusSearchOnStart() = %v, want %v", got, tt.want)
			}
		})
	}
}
