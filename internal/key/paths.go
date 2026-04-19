package key

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

var sshTokenPattern = regexp.MustCompile(`%[hprunCdiklLT]`)

var ignoredSSHFileSuffixes = [...]string{
	".pub",
	".backup",
	".bak",
	".tmp",
	".temp",
	".orig",
	".old",
	".log",
	".txt",
	".json",
	".md",
	".crt",
	".cer",
}

func defaultSSHDirectory() (string, error) {
	return config.GetSSHDirectory()
}

func expandPath(path string) (string, error) {
	expanded := strings.TrimSpace(path)
	expanded = strings.Trim(expanded, `"`)
	if expanded == "" {
		return "", fmt.Errorf("path is required")
	}

	expanded = os.ExpandEnv(expanded)
	expanded = expandWindowsHomeEnv(expanded)

	switch {
	case expanded == "~":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		expanded = homeDir
	case strings.HasPrefix(expanded, "~/"):
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		expanded = filepath.Join(homeDir, expanded[2:])
	case strings.HasPrefix(expanded, `~\`):
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		expanded = filepath.Join(homeDir, strings.ReplaceAll(expanded[2:], `\`, string(os.PathSeparator)))
	}

	return filepath.Clean(expanded), nil
}

func expandWindowsHomeEnv(path string) string {
	const prefix = `%USERPROFILE%`
	if len(path) < len(prefix) || !strings.EqualFold(path[:len(prefix)], prefix) {
		return path
	}

	home := os.Getenv("USERPROFILE")
	if home == "" {
		return path
	}

	rest := strings.TrimLeft(path[len(prefix):], `\/`)
	return filepath.Join(home, strings.ReplaceAll(rest, `\`, string(os.PathSeparator)))
}

func normalizeDeclaredIdentity(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || sshTokenPattern.MatchString(path) {
		return "", false
	}

	expanded, err := expandPath(path)
	if err != nil {
		return "", false
	}

	if !filepath.IsAbs(expanded) {
		return "", false
	}

	return expanded, true
}

func validateExistingPrivateKey(path string) (string, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(expanded)
	if err != nil {
		return "", fmt.Errorf("stat key %q: %w", expanded, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("key %q is not a regular file", expanded)
	}
	if strings.HasSuffix(strings.ToLower(expanded), ".pub") {
		return "", fmt.Errorf("expected a private key, got public key path %q", expanded)
	}

	return expanded, nil
}

func isIgnoredSSHFile(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "config",
		"known_hosts",
		"known_hosts.old",
		"authorized_keys",
		"authorized_keys.old",
		"environment",
		"allowed_signers",
		".ds_store":
		return true
	}

	for _, suffix := range ignoredSSHFileSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	return false
}

func permissionString(mode os.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}
