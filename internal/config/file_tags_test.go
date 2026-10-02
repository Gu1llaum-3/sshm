package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFileTagsCaseInsensitive(t *testing.T) {
	for _, directive := range []string{"# FileTags:", "# filetags:", "# FILETAGS:", "#FileTags:", "#  fIlEtAgS:", "#\tFileTags:"} {
		t.Run(directive, func(t *testing.T) {
			p := writeTempConfig(t, directive+" hidden\nHost h\n HostName h.example\n")
			hosts, err := ParseSSHConfigFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if len(FilterVisibleHosts(hosts)) != 0 {
				t.Fatalf("directive %q did not hide host: %+v", directive, hosts)
			}
		})
	}
}

func TestFileTagsLateDirectiveAfterDelete(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	p := writeTempConfig(t, "Host first\n HostName first.example\n\n#FileTags: hidden\nHost second\n HostName second.example\n")
	hosts, err := ParseSSHConfigFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(FilterVisibleHosts(hosts)) != 2 {
		t.Fatal("unexpected initial visibility")
	}
	if err := DeleteSSHHostFromFile("first", p); err != nil {
		t.Fatal(err)
	}
	hosts, err = ParseSSHConfigFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(FilterVisibleHosts(hosts)) != 1 {
		t.Fatalf("deleting first host activated ignored directive and hid remaining host: %+v", hosts)
	}
}

func TestFileTagsBoundaries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	paths := map[string]string{
		"main":       "# FileTags: parent\nInclude child\n# FileTags: ignored\nHost parent\n HostName p.example\n",
		"child":      "# FileTags: child, child, , shared\n# FileTags: shared, extra\nInclude grandchild\nHost a b\n HostName c.example\n",
		"grandchild": "# FileTags: grandchild\nHost grandchild\n HostName g.example\n",
	}
	for name, content := range paths {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	hosts, err := ParseSSHConfigFile(filepath.Join(dir, "main"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"parent": {"parent"}, "a": {"child", "shared", "extra"}, "b": {"child", "shared", "extra"}, "grandchild": {"grandchild"}}
	if len(hosts) != len(want) {
		t.Fatalf("hosts=%+v", hosts)
	}
	for _, h := range hosts {
		if !reflect.DeepEqual(h.InheritedTags, want[h.Name]) {
			t.Errorf("%s: %v want %v", h.Name, h.InheritedTags, want[h.Name])
		}
	}
	// Edit one alias, then delete every host: the file-level header must survive.
	child := filepath.Join(dir, "child")
	if err := UpdateSSHHostInFile("a", SSHHost{Name: "renamed", Hostname: "new.example", Tags: []string{"own"}}, child); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"renamed", "b"} {
		if err := DeleteSSHHostFromFile(name, child); err != nil {
			t.Fatal(err)
		}
	}
	if err := AddSSHHostToFile(SSHHost{Name: "replacement", Hostname: "r.example"}, child); err != nil {
		t.Fatal(err)
	}
	h, err := GetSSHHostFromFile("replacement", child)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.InheritedTags, []string{"child", "shared", "extra"}) {
		t.Fatalf("after edit/delete/add: %+v", h)
	}
}

func TestFormattedTags(t *testing.T) {
	h := SSHHost{InheritedTags: []string{"prod", "prod", "eu"}, Tags: []string{"prod", "db", "db"}}
	if got := h.FormattedTags(); !reflect.DeepEqual(got, []string{"%prod", "%eu", "#db"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFileTagsDeletePreservesOriginalHeader(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	p := writeTempConfig(t, "# FileTags: prod\n# FILETAGS: eu\n# Tags: own\nHost first\n HostName f.example\n\n# filetags: hidden\nHost second\n HostName s.example\n")
	for _, name := range []string{"first", "second"} {
		if err := DeleteSSHHostFromFile(name, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := AddSSHHostToFile(SSHHost{Name: "replacement", Hostname: "r.example"}, p); err != nil {
		t.Fatal(err)
	}
	host, err := GetSSHHostFromFile("replacement", p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(host.InheritedTags, []string{"prod", "eu"}) {
		t.Fatalf("header changed: %+v", host)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Ignored outside original file header: # filetags: hidden") {
		t.Fatalf("ignored directive not preserved: %s", data)
	}
}

// Exercise the add/delete operations used by MoveHostToFile with parsed hosts.
func TestFileTagsFollowDestinationOnTransfer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	source := writeTempConfig(t, "# FileTags: source, hidden\n# Tags: own\nHost moving\n HostName m.example\n")
	destination := writeTempConfig(t, "# FileTags: destination\n")
	host, err := GetSSHHostFromFile("moving", source)
	if err != nil {
		t.Fatal(err)
	}
	if err := AddSSHHostToFile(*host, destination); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSSHHostWithLine(*host); err != nil {
		t.Fatal(err)
	}
	moved, err := GetSSHHostFromFile("moving", destination)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(moved.InheritedTags, []string{"destination"}) || !reflect.DeepEqual(moved.Tags, []string{"own"}) {
		t.Fatalf("moved tags: %+v", moved)
	}
	if len(FilterVisibleHosts([]SSHHost{*moved})) != 1 {
		t.Fatal("source hidden tag leaked to destination")
	}
	remaining, err := ParseSSHConfigFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("source hosts: %+v", remaining)
	}
}

func TestFileTagsParserDoesNotWriteStderr(t *testing.T) {
	p := writeTempConfig(t, "Host first\n HostName f.example\n# FileTags: ignored\n")
	capture, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()
	original := os.Stderr
	os.Stderr = capture
	defer func() { os.Stderr = original }()
	for i := 0; i < 2; i++ {
		if _, err := ParseSSHConfigFile(p); err != nil {
			t.Fatal(err)
		}
	}
	info, err := capture.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Fatalf("parser wrote %d bytes to stderr", info.Size())
	}
}

func TestFileTagsWarningsIncludeLocationsAndDoNotAccumulate(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	parent := filepath.Join(dir, "config")
	for path, content := range map[string]string{
		child:  "Host child\n HostName child.example\n#FileTags: hidden\n",
		parent: "Include child\n#  FILETAGS: hidden\nInclude child\nHost parent\n HostName parent.example\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		hosts, warnings, err := ParseSSHConfigFileWithWarnings(parent)
		if err != nil {
			t.Fatal(err)
		}
		if len(hosts) != 2 || len(FilterVisibleHosts(hosts)) != 2 {
			t.Fatalf("hosts = %+v", hosts)
		}
		if len(warnings) != 2 {
			t.Fatalf("warnings = %v", warnings)
		}
		if !strings.Contains(warnings[0], child+":3") || !strings.Contains(warnings[1], parent+":2") || !strings.Contains(warnings[1], "Host or Include") {
			t.Fatalf("diagnostics lack location/boundary: %v", warnings)
		}
	}
}
