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
