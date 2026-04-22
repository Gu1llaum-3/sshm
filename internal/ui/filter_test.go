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
