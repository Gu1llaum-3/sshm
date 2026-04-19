package key

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// BuildAddArgs validates the key path and returns ssh-add arguments.
func BuildAddArgs(path string) (string, []string, error) {
	normalized, err := validateExistingPrivateKey(path)
	if err != nil {
		return "", nil, err
	}
	return normalized, []string{normalized}, nil
}

// SupportsAppleUseKeychain detects local support for Apple's ssh-add flag.
func SupportsAppleUseKeychain(ctx context.Context, runner Runner) (bool, error) {
	if runtime.GOOS != "darwin" {
		return false, nil
	}

	out, err := runner.Output(ctx, "ssh-add", "--apple-use-keychain", "-l")
	return appleUseKeychainProbeSupported(out, err)
}

func appleUseKeychainProbeSupported(out []byte, err error) (bool, error) {
	lower := strings.ToLower(string(out))
	if strings.Contains(lower, "illegal option") ||
		strings.Contains(lower, "invalid option") ||
		strings.Contains(lower, "unknown option") ||
		strings.Contains(lower, "unrecognized option") ||
		strings.Contains(lower, "usage:") {
		return false, nil
	}
	if err != nil && strings.TrimSpace(lower) == "" {
		return false, err
	}

	return true, nil
}

// Add loads a private key into the local ssh-agent.
func Add(ctx context.Context, runner Runner, opts AddOptions) (string, []string, error) {
	normalized, args, err := BuildAddArgs(opts.Path)
	if err != nil {
		return "", nil, err
	}

	if opts.AppleKeychain {
		supported, err := SupportsAppleUseKeychain(ctx, runner)
		if err != nil {
			return "", nil, err
		}
		if !supported {
			return "", nil, fmt.Errorf("--apple-keychain is not supported by the local ssh-add")
		}
		args = append([]string{"--apple-use-keychain"}, args...)
	}

	if !opts.DryRun {
		if err := runner.Run(ctx, "ssh-add", args...); err != nil {
			return "", nil, err
		}
	}

	return normalized, args, nil
}
