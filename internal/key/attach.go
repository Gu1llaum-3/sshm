package key

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

// Attach sets IdentityFile for a uniquely resolved host occurrence.
func Attach(ctx context.Context, opts AttachOptions) (AttachResult, error) {
	_ = ctx

	hostName := strings.TrimSpace(opts.Host)
	if hostName == "" {
		return AttachResult{}, fmt.Errorf("host name is required")
	}

	host, err := resolveAttachTarget(hostName, opts.ConfigPath)
	if err != nil {
		return AttachResult{}, err
	}
	return AttachToConcreteHost(*host, opts.Identity, opts.DryRun)
}

// AttachToConcreteHost sets IdentityFile for one concrete host occurrence.
func AttachToConcreteHost(host config.SSHHost, identity string, dryRun bool) (AttachResult, error) {
	normalizedIdentity, err := configIdentityPath(identity)
	if err != nil {
		return AttachResult{}, err
	}

	result := AttachResult{
		HostName:         host.Name,
		Identity:         normalizedIdentity,
		SourceFile:       host.SourceFile,
		Line:             host.LineNumber,
		DeclaredIdentity: strings.TrimSpace(host.Identity),
	}
	if sameIdentityDeclaration(host.Identity, normalizedIdentity) {
		result.AlreadyAttached = true
		return result, nil
	}

	if dryRun {
		return result, nil
	}

	if err := config.SetSSHHostIdentity(host, normalizedIdentity); err != nil {
		return AttachResult{}, err
	}
	return result, nil
}

func resolveAttachTarget(hostName string, configPath string) (*config.SSHHost, error) {
	if configPath != "" {
		return config.GetUniqueSSHHostFromFile(hostName, configPath)
	}
	return config.GetUniqueSSHHost(hostName)
}

func configIdentityPath(identity string) (string, error) {
	normalized, err := config.NormalizeAttachedIdentityForKey(identity)
	if err != nil {
		return "", err
	}
	return filepath.Clean(normalized), nil
}

func sameIdentityDeclaration(declared string, normalizedIdentity string) bool {
	trimmed := strings.TrimSpace(declared)
	if trimmed == "" {
		return false
	}
	normalizedDeclared, err := config.NormalizeAttachedIdentityForKey(trimmed)
	if err != nil {
		return false
	}
	return filepath.Clean(normalizedDeclared) == normalizedIdentity
}
