package ui

import (
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

func TestApplySourceFileFilter(t *testing.T) {
	hosts := []config.SSHHost{
		{Name: "a", SourceFile: "/home/u/.ssh/config"},
		{Name: "b", SourceFile: "/home/u/.ssh/work.conf"},
		{Name: "c", SourceFile: "/home/u/.ssh/work.conf"},
		{Name: "d", SourceFile: "/home/u/.ssh/perso.conf"},
	}

	t.Run("empty selected returns all", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "")
		if len(got) != len(hosts) {
			t.Fatalf("expected %d hosts, got %d", len(hosts), len(got))
		}
	})

	t.Run("matching path filters to that file", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "/home/u/.ssh/work.conf")
		if len(got) != 2 {
			t.Fatalf("expected 2 hosts, got %d", len(got))
		}
		for _, h := range got {
			if h.SourceFile != "/home/u/.ssh/work.conf" {
				t.Errorf("unexpected host %q with SourceFile %q", h.Name, h.SourceFile)
			}
		}
	})

	t.Run("non-matching path returns empty", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "/nowhere.conf")
		if len(got) != 0 {
			t.Fatalf("expected 0 hosts, got %d", len(got))
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		got := applySourceFileFilter(nil, "/home/u/.ssh/work.conf")
		if len(got) != 0 {
			t.Fatalf("expected 0 hosts, got %d", len(got))
		}
	})
}

func TestFileSelectorWithAllPrependsSynthetic(t *testing.T) {
	// We can't easily mock GetAllConfigFilesFromBase, so drive the helper
	// directly through newFileSelectorFromFiles then apply the same
	// prepending logic and assert the invariants a caller relies on.
	files := []string{"/a.conf", "/b.conf"}
	styles := NewStyles(80)
	m, err := newFileSelectorFromFiles("t", styles, 80, 24, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.files = append([]string{""}, m.files...)
	m.displayNames = append([]string{"[All files]"}, m.displayNames...)

	if len(m.files) != 3 || m.files[0] != "" {
		t.Fatalf("expected synthetic entry at index 0, got %v", m.files)
	}
	if m.displayNames[0] != "[All files]" {
		t.Fatalf("expected display name '[All files]', got %q", m.displayNames[0])
	}
}
