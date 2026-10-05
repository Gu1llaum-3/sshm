package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// tokenizerCases are config lines with the arguments OpenSSH reads from them
// (checked with ssh -G, OpenSSH 10.3).
var tokenizerCases = []struct {
	name string
	line string
	args []string
}{
	{"plain value", `IdentityFile /a/id`, []string{`/a/id`}},
	{"double quotes keep inner spaces", `IdentityFile "/a  b/id"`, []string{`/a  b/id`}},
	{"single quotes", `IdentityFile '/a b/id'`, []string{`/a b/id`}},
	{"escaped space", `IdentityFile /a\ b/id`, []string{`/a b/id`}},
	{"windows path unquoted", `IdentityFile C:\Users\me\id`, []string{`C:\Users\me\id`}},
	{"windows path double-quoted", `IdentityFile "C:\Users\me\id"`, []string{`C:\Users\me\id`}},
	{"escaped backslashes", `IdentityFile C:\\x\\id`, []string{`C:\x\id`}},
	{"escaped backslashes inside double quotes", `IdentityFile "C:\\x\\id"`, []string{`C:\x\id`}},
	{"escaped quote inside double quotes", `IdentityFile "/a\"b"`, []string{`/a"b`}},
	{"backslash-space kept inside single quotes", `IdentityFile '/a\ b'`, []string{`/a\ b`}},
	{"escaped backslash inside single quotes", `IdentityFile '/a\\b'`, []string{`/a\b`}},
	{"escaped quote inside single quotes", `IdentityFile '/a\'b'`, []string{`/a'b`}},
	{"backslash-space kept inside double quotes", `IdentityFile "/a\ b"`, []string{`/a\ b`}},
	{"quoted part glued to text", `IdentityFile "/a"b`, []string{`/ab`}},
	{"tab inside double quotes", "IdentityFile \"/a\tb\"", []string{"/a\tb"}},
	{"trailing comment", `IdentityFile /a/id # old key`, []string{`/a/id`}},
	{"hash inside a word", `IdentityFile /a/id#x`, []string{`/a/id#x`}},
	{"hash inside quotes", `IdentityFile "/a #b"`, []string{`/a #b`}},
	{"key=value", `IdentityFile=/a/id`, []string{`/a/id`}},
	{"key = value", `IdentityFile = /a/id`, []string{`/a/id`}},
	{"tab separators", "\tIdentityFile\t/a/id", []string{`/a/id`}},
	{"several arguments", `Host a "b c" 'd'`, []string{`a`, `b c`, `d`}},
	{"quoted user", `User "john doe"`, []string{`john doe`}},
}

func TestSplitConfigLineFollowsOpenSSH(t *testing.T) {
	for _, tt := range tokenizerCases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := splitConfigLine(tt.line)
			if !ok {
				t.Fatalf("splitConfigLine(%q) returned false", tt.line)
			}
			if !reflect.DeepEqual(got.args, tt.args) {
				t.Errorf("args = %q, want %q", got.args, tt.args)
			}
		})
	}
}

func TestSplitConfigLineKeywordAndRawText(t *testing.T) {
	tests := []struct {
		line         string
		keyword      string
		raw          string
		rawNoComment string
	}{
		{`ProxyCommand ssh -W "%h:%p" bastion # c`, "ProxyCommand", `ssh -W "%h:%p" bastion # c`, `ssh -W "%h:%p" bastion`},
		{`ProxyCommand=ssh -W %h:%p bastion`, "ProxyCommand", `ssh -W %h:%p bastion`, `ssh -W %h:%p bastion`},
		{`  SetEnv FOO="a  b"   `, "SetEnv", `FOO="a  b"`, `FOO="a  b"`},
		{"HostName\t=\tex.com", "HostName", "ex.com", "ex.com"},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, ok := splitConfigLine(tt.line)
			if !ok {
				t.Fatalf("splitConfigLine(%q) returned false", tt.line)
			}
			if got.keyword != tt.keyword || got.raw != tt.raw || got.rawNoComment != tt.rawNoComment {
				t.Errorf("got keyword=%q raw=%q rawNoComment=%q, want %q %q %q",
					got.keyword, got.raw, got.rawNoComment, tt.keyword, tt.raw, tt.rawNoComment)
			}
		})
	}
}

func TestSplitConfigLineSkipsBlankLinesAndComments(t *testing.T) {
	for _, line := range []string{"", "   ", "\t", "# a comment", "   # Tags: prod"} {
		if _, ok := splitConfigLine(line); ok {
			t.Errorf("splitConfigLine(%q) = true, want false", line)
		}
	}
}

// sshm stays lenient where OpenSSH rejects the line (decision of the plan)
func TestSplitConfigLineIsLenient_AC4(t *testing.T) {
	tests := []struct {
		name string
		line string
		args []string
	}{
		{"unterminated quote keeps the rest", `IdentityFile "/a b`, []string{`/a b`}},
		{"unquoted path with spaces gives several words", `IdentityFile C:\My Drive\key`, []string{`C:\My`, `Drive\key`}},
		{"keyword alone", `Host`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := splitConfigLine(tt.line)
			if !ok {
				t.Fatalf("splitConfigLine(%q) returned false", tt.line)
			}
			if !reflect.DeepEqual(got.args, tt.args) {
				t.Errorf("args = %q, want %q", got.args, tt.args)
			}
		})
	}
}

// The tokenizer reads IdentityFile values exactly like the installed ssh client.
func TestSplitConfigLineMatchesSSHClient(t *testing.T) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("ssh client not installed")
	}
	for _, tt := range tokenizerCases {
		if !strings.HasPrefix(tt.line, "IdentityFile") {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			if err := os.WriteFile(configFile, []byte("Host h\n  "+tt.line+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(sshPath, "-F", configFile, "-G", "h").Output()
			if err != nil {
				t.Fatalf("ssh -G failed: %v", err)
			}
			var fromSSH []string
			for _, l := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(l, "identityfile ") {
					fromSSH = append(fromSSH, strings.TrimPrefix(l, "identityfile "))
				}
			}
			got, _ := splitConfigLine(tt.line)
			if !reflect.DeepEqual(got.args, fromSSH) {
				t.Errorf("tokenizer = %q, ssh -G = %q", got.args, fromSSH)
			}
		})
	}
}
