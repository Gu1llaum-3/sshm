package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

func TestInfoInheritedTagsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("# FileTags: prod, eu\n# Tags: prod, db\nHost database\n HostName db.example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := runInfo(&out, "database", path, false); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	var result struct {
		Result struct {
			Tags          []string `json:"tags"`
			InheritedTags []string `json:"inheritedTags"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Result.Tags, []string{"prod", "eu", "db"}) || !reflect.DeepEqual(result.Result.InheritedTags, []string{"prod", "eu"}) {
		t.Fatalf("JSON result = %+v", result)
	}
}

func TestSearchOutputsInheritedTags(t *testing.T) {
	hosts := []config.SSHHost{{Name: "database", Hostname: "db.example.com", Tags: []string{"prod", "db"}, InheritedTags: []string{"prod", "eu"}}}
	for _, format := range []string{"json", "table"} {
		t.Run(format, func(t *testing.T) {
			out, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			original := os.Stdout
			os.Stdout = out
			defer func() { os.Stdout = original }()
			if format == "json" {
				outputJSON(hosts)
			} else {
				outputTable(hosts)
			}
			os.Stdout = original
			if _, err := out.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(out)
			if err != nil {
				t.Fatal(err)
			}
			if format == "json" {
				var results []struct {
					Tags []string `json:"tags"`
				}
				if err := json.Unmarshal(data, &results); err != nil {
					t.Fatal(err)
				}
				if len(results) != 1 || !reflect.DeepEqual(results[0].Tags, []string{"prod", "eu", "db"}) {
					t.Fatalf("JSON = %s", data)
				}
			} else if !strings.Contains(string(data), "prod, eu, db") {
				t.Fatalf("table = %s", data)
			}
		})
	}
}

func TestCompletionHonorsInheritedHiddenTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("# FileTags: hidden\nHost secret\n HostName secret.example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := configFile
	configFile = path
	defer func() { configFile = original }()
	completions, _ := RootCmd.ValidArgsFunction(RootCmd, nil, "")
	if len(completions) != 0 {
		t.Fatalf("hidden completions: %v", completions)
	}
	// Hidden affects listing and completion, but the host remains connectable.
	exists, err := config.QuickHostExistsInFile("secret", path)
	if err != nil || !exists {
		t.Fatalf("hidden host lookup: exists=%v, err=%v", exists, err)
	}
}

func TestCLIWarningsOnceAndSeparateFromJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	child := filepath.Join(dir, "child")
	for name, content := range map[string]string{
		path:  "Include child\nInclude child\n",
		child: "Host database\n HostName db.example.com\n#FileTags: hidden\n",
	} {
		if err := os.WriteFile(name, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, command := range []string{"search", "info"} {
		t.Run(command, func(t *testing.T) {
			stdout, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer stdout.Close()
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			defer stderr.Close()
			oldOut, oldErr := os.Stdout, os.Stderr
			oldConfig, oldFormat, oldTagsOnly, oldNamesOnly := configFile, outputFormat, tagsOnly, namesOnly
			os.Stdout, os.Stderr = stdout, stderr
			configFile, outputFormat, tagsOnly, namesOnly = path, "json", false, false
			defer func() {
				os.Stdout, os.Stderr = oldOut, oldErr
				configFile, outputFormat, tagsOnly, namesOnly = oldConfig, oldFormat, oldTagsOnly, oldNamesOnly
			}()
			if command == "search" {
				runSearch(searchCmd, nil)
			} else if code := runInfo(stdout, "database", path, false); code != 0 {
				t.Fatalf("info exited %d", code)
			}
			data, err := os.ReadFile(stdout.Name())
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(data) || !strings.Contains(string(data), "database") {
				t.Fatalf("invalid JSON: %s", data)
			}
			diagnostic, err := os.ReadFile(stderr.Name())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(diagnostic), "warning:") != 1 || !strings.Contains(string(diagnostic), child+":3") {
				t.Fatalf("warnings: %s", diagnostic)
			}
		})
	}
}
