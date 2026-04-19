package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// SetSSHHostIdentity updates the exact concrete host occurrence to use the given IdentityFile.
// It fails closed for multi-host declarations and for blocks that contain multiple explicit
// IdentityFile directives, because those shapes are not safe to rewrite generically.
func SetSSHHostIdentity(host SSHHost, identity string) error {
	if strings.TrimSpace(host.Name) == "" {
		return fmt.Errorf("host name is required")
	}
	if strings.TrimSpace(host.SourceFile) == "" {
		return fmt.Errorf("host '%s' has no source file", host.Name)
	}
	if host.LineNumber <= 0 {
		return fmt.Errorf("host '%s' has no line number", host.Name)
	}

	normalizedIdentity, err := normalizeAttachedIdentity(identity)
	if err != nil {
		return err
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	if err := backupConfig(host.SourceFile); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	content, err := os.ReadFile(host.SourceFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	targetIdx := host.LineNumber - 1
	if targetIdx < 0 || targetIdx >= len(lines) {
		return fmt.Errorf("host '%s' line %d is out of range", host.Name, host.LineNumber)
	}

	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[targetIdx])), "host ") {
		return fmt.Errorf("line %d in %s is not a Host declaration", host.LineNumber, host.SourceFile)
	}
	hostNames := strings.Fields(strings.TrimSpace(lines[targetIdx])[5:])
	if len(hostNames) > 1 {
		return fmt.Errorf("host '%s' is part of a multi-host declaration; split it in Edit Host before attaching a key", host.Name)
	}

	blockEnd := len(lines)
	for i := targetIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if isSSHSectionBoundary(trimmed) {
			blockEnd = i
			break
		}
	}

	identityIndexes := make([]int, 0, 1)
	for i := targetIdx + 1; i < blockEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if hasSSHDirective(trimmed, "identityfile") {
			identityIndexes = append(identityIndexes, i)
		}
	}
	if len(identityIndexes) > 1 {
		return fmt.Errorf("host '%s' has multiple explicit IdentityFile directives; attach is blocked under current safety rules", host.Name)
	}

	identityLine := "    IdentityFile " + formatSSHConfigValue(normalizedIdentity)
	switch len(identityIndexes) {
	case 0:
		insertAt := blockEnd
		for insertAt > targetIdx+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
			insertAt--
		}
		lines = slices.Insert(lines, insertAt, identityLine)
	case 1:
		idx := identityIndexes[0]
		indent := leadingWhitespace(lines[idx])
		if indent == "" {
			indent = "    "
		}
		lines[idx] = indent + "IdentityFile " + formatSSHConfigValue(normalizedIdentity)
	}

	return os.WriteFile(host.SourceFile, []byte(strings.Join(lines, "\n")), 0600)
}

func normalizeAttachedIdentity(identity string) (string, error) {
	path := strings.TrimSpace(identity)
	if path == "" {
		return "", fmt.Errorf("identity path is required")
	}
	if strings.HasSuffix(path, ".pub") {
		path = strings.TrimSuffix(path, ".pub")
	}

	expanded, err := expandSSHPath(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(expanded); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("private key not found: %s", expanded)
		}
		return "", err
	}
	return expanded, nil
}

// NormalizeAttachedIdentityForKey resolves an attach identity to an existing private-key path.
func NormalizeAttachedIdentityForKey(identity string) (string, error) {
	return normalizeAttachedIdentity(identity)
}

func expandSSHPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	path = os.ExpandEnv(path)
	path = expandWindowsHomeEnv(path)

	if path == "~" {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = homeDir
	} else if strings.HasPrefix(path, "~/") {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	} else if strings.HasPrefix(path, `~\`) {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(homeDir, strings.ReplaceAll(strings.TrimPrefix(path, `~\`), `\`, string(os.PathSeparator)))
	}
	return filepath.Clean(path), nil
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

func isSSHSectionBoundary(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "host ") || strings.HasPrefix(lower, "match ")
}

func hasSSHDirective(line string, directive string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	return strings.EqualFold(fields[0], directive)
}

func leadingWhitespace(line string) string {
	idx := 0
	for idx < len(line) {
		if line[idx] != ' ' && line[idx] != '\t' {
			break
		}
		idx++
	}
	return line[:idx]
}
