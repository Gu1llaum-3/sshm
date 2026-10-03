package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const twoKeysConfig = `Host keys
    HostName keys.example.com
    IdentityFile ~/.ssh/a
    IdentityFile "~/.ssh/b c"
`

func writeTwoKeysConfig(t *testing.T) string {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configFile, []byte(twoKeysConfig), 0600); err != nil {
		t.Fatal(err)
	}
	return configFile
}

func identityLinesOf(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, line := range strings.Split(string(content), "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "IdentityFile") {
			got = append(got, line)
		}
	}
	return got
}

// AC-6: changing the main key in the edit form leaves the extra keys untouched
func TestEditFormKeepsExtraIdentities_AC6(t *testing.T) {
	// the form only accepts a main key that exists on disk
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "new"), []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}

	configFile := writeTwoKeysConfig(t)
	form, err := NewEditForm("keys", NewStyles(80), 80, 60, configFile)
	if err != nil {
		t.Fatalf("NewEditForm() error = %v", err)
	}
	form.inputs[3].SetValue("~/.ssh/new")

	msg, ok := form.submitEditForm()().(editFormSubmitMsg)
	if !ok || msg.err != nil {
		t.Fatalf("submitEditForm() error = %v", msg.err)
	}
	want := []string{`IdentityFile ~/.ssh/new`, `IdentityFile "~/.ssh/b c"`}
	if got := identityLinesOf(t, configFile); !reflect.DeepEqual(got, want) {
		t.Errorf("IdentityFile lines = %q, want %q", got, want)
	}
}

// AC-6: the edit form and the info view show the extra keys
func TestFormsShowExtraIdentities_AC6(t *testing.T) {
	configFile := writeTwoKeysConfig(t)

	form, err := NewEditForm("keys", NewStyles(80), 80, 60, configFile)
	if err != nil {
		t.Fatalf("NewEditForm() error = %v", err)
	}
	if view := form.View(); !strings.Contains(view, "~/.ssh/b c") {
		t.Errorf("edit form does not show the extra key:\n%s", view)
	}

	info, err := NewInfoForm("keys", NewStyles(80), 120, 60, configFile)
	if err != nil {
		t.Fatalf("NewInfoForm() error = %v", err)
	}
	if view := info.View(); !strings.Contains(view, "~/.ssh/b c") {
		t.Errorf("info view does not show the extra key:\n%s", view)
	}
}
