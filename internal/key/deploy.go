package key

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/validation"
)

var hasSSHCopyID = func() bool {
	_, err := exec.LookPath("ssh-copy-id")
	return err == nil
}

type resolvedDeployTarget struct {
	destination string
	port        string
	sshOptions  []string
}

// BuildDeployPlan validates deployment options and returns the concrete command plan.
func BuildDeployPlan(opts DeployOptions) (DeployPlan, error) {
	configPath := strings.TrimSpace(opts.ConfigPath)

	target, err := resolveDeployTarget(opts)
	if err != nil {
		return DeployPlan{}, err
	}

	publicKeyPath, publicKey, err := resolvePublicKey(opts.Identity)
	if err != nil {
		return DeployPlan{}, err
	}

	if hasSSHCopyID() {
		args := make([]string, 0, 10)
		args = append(args, "-i", publicKeyPath)
		if configPath != "" {
			args = append(args, "-F", configPath)
		}
		if target.port != "" {
			args = append(args, "-p", target.port)
		}
		args = appendSSHOptions(args, target.sshOptions)
		args = append(args, target.destination)

		return DeployPlan{
			Command:       "ssh-copy-id",
			Args:          args,
			PublicKeyPath: publicKeyPath,
			Target:        target.destination,
		}, nil
	}

	quotedKey := shellSingleQuote(publicKey)
	remoteScript := fmt.Sprintf(
		"umask 077; mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && { grep -qxF %s ~/.ssh/authorized_keys || printf '%%s\\n' %s >> ~/.ssh/authorized_keys; }",
		quotedKey,
		quotedKey,
	)

	args := make([]string, 0, 12)
	if configPath != "" {
		args = append(args, "-F", configPath)
	}
	if target.port != "" {
		args = append(args, "-p", target.port)
	}
	args = appendSSHOptions(args, target.sshOptions)
	args = append(args, target.destination, "sh", "-c", remoteScript)

	return DeployPlan{
		Command:       "ssh",
		Args:          args,
		PublicKeyPath: publicKeyPath,
		Target:        target.destination,
	}, nil
}

// Deploy installs the selected public key onto the remote host using OpenSSH tooling.
func Deploy(ctx context.Context, runner Runner, opts DeployOptions) (DeployPlan, error) {
	plan, err := BuildDeployPlan(opts)
	if err != nil {
		return DeployPlan{}, err
	}
	if !opts.DryRun {
		if err := runner.Run(ctx, plan.Command, plan.Args...); err != nil {
			return DeployPlan{}, err
		}
	}
	return plan, nil
}

func resolveDeployTarget(opts DeployOptions) (resolvedDeployTarget, error) {
	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return resolvedDeployTarget{}, fmt.Errorf("deploy target is required")
	}

	user := strings.TrimSpace(opts.User)
	port := strings.TrimSpace(opts.Port)
	proxyJump := strings.TrimSpace(opts.ProxyJump)
	proxyCommand := strings.TrimSpace(opts.ProxyCommand)

	if port != "" && !validation.ValidatePort(port) {
		return resolvedDeployTarget{}, fmt.Errorf("port must be between 1 and 65535")
	}

	sshOptions := buildDeploySSHOptions(proxyJump, proxyCommand)

	var (
		matches []config.SSHHost
		err     error
	)
	if strings.TrimSpace(opts.ConfigPath) != "" {
		matches, err = config.LookupSSHHostsByNameFromFile(target, opts.ConfigPath)
	} else {
		matches, err = config.LookupSSHHostsByName(target)
	}
	if err != nil {
		return resolvedDeployTarget{}, err
	}

	if len(matches) > 1 {
		return resolvedDeployTarget{}, fmt.Errorf("host %q is ambiguous in ssh config: %d matches found", target, len(matches))
	}

	return resolvedDeployTarget{
		destination: composeUserHost(user, target),
		port:        port,
		sshOptions:  sshOptions,
	}, nil
}

func buildDeploySSHOptions(proxyJump string, proxyCommand string) []string {
	options := make([]string, 0, 2)
	if proxyJump != "" {
		options = append(options, "ProxyJump="+proxyJump)
	}
	if proxyCommand != "" {
		options = append(options, "ProxyCommand="+proxyCommand)
	}
	return options
}

func appendSSHOptions(args []string, options []string) []string {
	for _, option := range options {
		if strings.TrimSpace(option) == "" {
			continue
		}
		args = append(args, "-o", option)
	}
	return args
}

func resolvePublicKey(identity string) (string, string, error) {
	trimmed := strings.TrimSpace(identity)
	if trimmed == "" {
		return "", "", fmt.Errorf("identity path is required")
	}

	expanded, err := expandPath(trimmed)
	if err != nil {
		return "", "", err
	}

	publicKeyPath := expanded
	if !strings.HasSuffix(strings.ToLower(expanded), ".pub") {
		privateKeyPath, err := validateExistingPrivateKey(expanded)
		if err != nil {
			return "", "", err
		}
		publicKeyPath = privateKeyPath + ".pub"
	}

	info, err := os.Stat(publicKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("stat public key %q: %w", publicKeyPath, err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("public key %q is not a regular file", publicKeyPath)
	}

	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("read public key %q: %w", publicKeyPath, err)
	}

	publicKey := strings.TrimSpace(string(content))
	if publicKey == "" {
		return "", "", fmt.Errorf("public key %q is empty", publicKeyPath)
	}

	return filepath.Clean(publicKeyPath), publicKey, nil
}

func composeUserHost(user string, host string) string {
	if strings.TrimSpace(user) == "" {
		return host
	}
	return user + "@" + host
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
