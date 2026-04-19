package key

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

// Inventory returns local key files plus explicit ssh_config references.
func Inventory(ctx context.Context, runner Runner, configPath string) ([]InventoryItem, error) {
	sshDir, err := defaultSSHDirectory()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ssh directory: %w", err)
	}

	references, err := loadDeclaredReferences(configPath)
	if err != nil {
		return nil, err
	}

	items := make([]InventoryItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || isIgnoredSSHFile(entry.Name()) {
			continue
		}

		path := filepath.Join(sshDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		out, err := runner.Output(ctx, "ssh-keygen", "-lf", path)
		if err != nil {
			continue
		}

		fingerprint, algorithm := parseFingerprintOutput(string(out))
		if fingerprint == "" {
			continue
		}

		normalized := filepath.Clean(path)
		publicKeyPath := ""
		if info, err := os.Stat(normalized + ".pub"); err == nil && info.Mode().IsRegular() {
			publicKeyPath = normalized + ".pub"
		}

		refs := slices.Clone(references[normalized])
		items = append(items, InventoryItem{
			Path:          normalized,
			PublicKeyPath: publicKeyPath,
			Permissions:   permissionString(info.Mode()),
			Fingerprint:   fingerprint,
			Algorithm:     algorithm,
			References:    refs,
			CanDelete:     len(refs) == 0,
		})
	}

	slices.SortFunc(items, func(a, b InventoryItem) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return items, nil
}

func loadDeclaredReferences(configPath string) (map[string][]Reference, error) {
	var (
		hosts []config.SSHHost
		err   error
	)

	if configPath == "" {
		hosts, err = config.ParseSSHConfig()
	} else {
		hosts, err = config.ParseSSHConfigFile(configPath)
	}
	if err != nil {
		return nil, fmt.Errorf("parse ssh config: %w", err)
	}

	references := make(map[string][]Reference)
	for _, host := range hosts {
		normalized, ok := normalizeDeclaredIdentity(host.Identity)
		if !ok {
			continue
		}

		references[normalized] = append(references[normalized], Reference{
			Host:                 host.Name,
			SourceFile:           host.SourceFile,
			Line:                 host.LineNumber,
			DeclaredIdentityFile: strings.TrimSpace(host.Identity),
		})
	}

	for path, refs := range references {
		slices.SortFunc(refs, func(a, b Reference) int {
			if diff := cmp.Compare(a.Host, b.Host); diff != 0 {
				return diff
			}
			if diff := cmp.Compare(a.SourceFile, b.SourceFile); diff != 0 {
				return diff
			}
			return cmp.Compare(a.Line, b.Line)
		})
		references[path] = refs
	}

	return references, nil
}

func parseFingerprintOutput(out string) (fingerprint string, algorithm string) {
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) < 2 {
		return "", ""
	}

	fingerprint = fields[1]
	last := fields[len(fields)-1]
	if strings.HasPrefix(last, "(") && strings.HasSuffix(last, ")") {
		algorithm = strings.TrimSuffix(strings.TrimPrefix(last, "("), ")")
	}

	return fingerprint, algorithm
}
