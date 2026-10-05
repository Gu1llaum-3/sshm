package config

import "testing"

// AC-3 (#48): showing the options in the form then reading them back gives
// exactly the same options
func TestOptionsRoundTripThroughTheForm_AC3(t *testing.T) {
	for _, options := range []string{
		"SetEnv TERM=xterm-256color",
		"SendEnv LANG LC_*",
		"CertificateFile ~/.ssh/id-other-cert.pub",
		`LocalCommand echo "a b"`,
		"ServerAliveInterval 60",
		"SetEnv TERM=xterm-256color\nSendEnv LANG LC_*\nServerAliveInterval 60",
	} {
		t.Run(options, func(t *testing.T) {
			shown := FormatSSHOptionsForCommand(options)
			if got := ParseSSHOptionsFromCommand(shown); got != options {
				t.Errorf("form shows %q, read back as %q, want %q", shown, got, options)
			}
		})
	}
}

// AC-5 (#48): what users type in the form today still gives the same options
func TestTypedOptionsAreStillAccepted_AC5(t *testing.T) {
	for typed, want := range map[string]string{
		"-o Compression=yes -o ServerAliveInterval=60": "Compression yes\nServerAliveInterval 60",
		"Compression yes":                    "Compression yes",
		"ForwardX11 true":                    "ForwardX11 true",
		"-o SetEnv=TERM=xterm-256color":      "SetEnv TERM=xterm-256color",
		"-oServerAliveInterval=30":           "ServerAliveInterval 30",
		"-o CertificateFile=~/.ssh/id-other": "CertificateFile ~/.ssh/id-other",
		"LocalCommand tool -opt":             "LocalCommand tool -opt",
		"Compression yes -o ForwardX11=yes":  "Compression yes\nForwardX11 yes",
		"Compression=yes -o ForwardX11=yes":  "Compression yes\nForwardX11 yes",
		"-o LocalCommand=tool -opt":          "LocalCommand tool -opt",
	} {
		t.Run(typed, func(t *testing.T) {
			if got := ParseSSHOptionsFromCommand(typed); got != want {
				t.Errorf("ParseSSHOptionsFromCommand(%q) = %q, want %q", typed, got, want)
			}
		})
	}
}
