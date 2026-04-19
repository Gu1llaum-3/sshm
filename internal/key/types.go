package key

import "context"

// Runner abstracts local subprocess execution for OpenSSH tooling.
type Runner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	Run(ctx context.Context, name string, args ...string) error
}

// Reference describes an explicit IdentityFile declaration in ssh_config.
type Reference struct {
	Host                 string `json:"host"`
	SourceFile           string `json:"source_file,omitempty"`
	Line                 int    `json:"line,omitempty"`
	DeclaredIdentityFile string `json:"declared_identity_file,omitempty"`
}

// InventoryItem describes a local key file plus explicit config references.
type InventoryItem struct {
	Path          string      `json:"path"`
	PublicKeyPath string      `json:"public_key_path,omitempty"`
	Permissions   string      `json:"permissions"`
	Fingerprint   string      `json:"fingerprint"`
	Algorithm     string      `json:"algorithm,omitempty"`
	References    []Reference `json:"references,omitempty"`
	CanDelete     bool        `json:"can_delete"`
}

// GenerateOptions controls ssh-keygen invocation.
type GenerateOptions struct {
	Name      string
	Algorithm string
	Comment   string
	Directory string
	KDFRounds int
	DryRun    bool
}

// GenerateResult reports generated file paths.
type GenerateResult struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

// AddOptions controls ssh-add invocation.
type AddOptions struct {
	Path          string
	AppleKeychain bool
	DryRun        bool
}

// DeployOptions controls public-key deployment to a remote host.
type DeployOptions struct {
	Target       string
	User         string
	Port         string
	ProxyJump    string
	ProxyCommand string
	Identity     string
	ConfigPath   string
	DryRun       bool
}

// DeployPlan describes the concrete local command SSHM will execute.
type DeployPlan struct {
	Command       string
	Args          []string
	PublicKeyPath string
	Target        string
}

// AttachOptions controls IdentityFile attachment to an existing host block.
type AttachOptions struct {
	Host       string
	Identity   string
	ConfigPath string
	DryRun     bool
}

// AttachResult reports the concrete host occurrence that was updated.
type AttachResult struct {
	HostName         string
	Identity         string
	SourceFile       string
	Line             int
	AlreadyAttached  bool
	DeclaredIdentity string
}
