package key

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var allowedGenerateAlgorithms = []string{"ed25519", "ecdsa", "rsa"}

func AllowedGenerateAlgorithms() []string {
	return slices.Clone(allowedGenerateAlgorithms)
}

func NormalizeGenerateAlgorithm(value string) (string, error) {
	algorithm := strings.ToLower(strings.TrimSpace(value))
	if algorithm == "" {
		return "ed25519", nil
	}
	if slices.Contains(allowedGenerateAlgorithms, algorithm) {
		return algorithm, nil
	}
	return "", fmt.Errorf("algorithm must be one of: %s", strings.Join(allowedGenerateAlgorithms, ", "))
}

// BuildGenerateArgs validates options and returns ssh-keygen arguments.
func BuildGenerateArgs(opts GenerateOptions) (GenerateResult, []string, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return GenerateResult{}, nil, fmt.Errorf("key name is required")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return GenerateResult{}, nil, fmt.Errorf("key name must be a file name, not a path")
	}

	directory := opts.Directory
	if strings.TrimSpace(directory) == "" {
		var err error
		directory, err = defaultSSHDirectory()
		if err != nil {
			return GenerateResult{}, nil, err
		}
	}

	directory, err := expandPath(directory)
	if err != nil {
		return GenerateResult{}, nil, err
	}

	privateKeyPath := filepath.Join(directory, name)
	if _, err := os.Stat(privateKeyPath); err == nil {
		return GenerateResult{}, nil, fmt.Errorf("key %q already exists", privateKeyPath)
	} else if !os.IsNotExist(err) {
		return GenerateResult{}, nil, fmt.Errorf("stat key %q: %w", privateKeyPath, err)
	}

	algorithm, err := NormalizeGenerateAlgorithm(opts.Algorithm)
	if err != nil {
		return GenerateResult{}, nil, err
	}

	args := []string{"-t", algorithm, "-f", privateKeyPath}
	if opts.Comment != "" {
		args = append(args, "-C", opts.Comment)
	}
	if opts.KDFRounds > 0 {
		args = append(args, "-a", fmt.Sprintf("%d", opts.KDFRounds))
	}

	return GenerateResult{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  privateKeyPath + ".pub",
	}, args, nil
}

// Generate executes ssh-keygen with validated options.
func Generate(ctx context.Context, runner Runner, opts GenerateOptions) (GenerateResult, error) {
	result, args, err := BuildGenerateArgs(opts)
	if err != nil {
		return GenerateResult{}, err
	}

	if !opts.DryRun {
		if err := os.MkdirAll(filepath.Dir(result.PrivateKeyPath), 0700); err != nil {
			return GenerateResult{}, fmt.Errorf("create key directory: %w", err)
		}
		if err := runner.Run(ctx, "ssh-keygen", args...); err != nil {
			return GenerateResult{}, err
		}
	}

	return result, nil
}
