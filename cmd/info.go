package cmd

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"

	"github.com/spf13/cobra"
)

type infoResponse struct {
	Schema   string      `json:"schema"`
	OK       bool        `json:"ok"`
	Hostname string      `json:"hostname"`
	Result   *infoResult `json:"result"`
	Error    *infoError  `json:"error"`
}

type infoResult struct {
	CanonicalName string      `json:"canonical_name"`
	Target        infoTarget  `json:"target"`
	IdentityFile  *string     `json:"identity_file"`
	ProxyJump     *string     `json:"proxy_jump"`
	ProxyCommand  *string     `json:"proxy_command"`
	Options       *string     `json:"options"`
	Tags          []string    `json:"tags"`
	RemoteCommand *string     `json:"remote_command"`
	RequestTTY    *string     `json:"request_tty"`
	Source        *infoSource `json:"source"`
}

type infoTarget struct {
	Host     string  `json:"host"`
	Hostname *string `json:"hostname"`
	User     *string `json:"user"`
	Port     *int    `json:"port"`
}

type infoSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type infoError struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

func maybeString(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func maybePort(v string) (*int, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil, nil
	}
	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil, err
	}
	return &port, nil
}

func writeInfoJSON(out io.Writer, pretty bool, resp infoResponse) {
	var b []byte
	var err error
	if pretty {
		b, err = json.MarshalIndent(resp, "", "  ")
	} else {
		b, err = json.Marshal(resp)
	}
	if err != nil {
		_, _ = io.WriteString(out, `{"schema":"sshm.info.v1","ok":false,"hostname":"","result":null,"error":{"code":"INTERNAL","message":"failed to marshal JSON","details":null}}\n`)
		return
	}
	_, _ = out.Write(append(b, '\n'))
}

func runInfo(out io.Writer, hostnameArg string, cfgFile string, pretty bool) int {
	resp := infoResponse{
		Schema:   "sshm.info.v1",
		OK:       false,
		Hostname: hostnameArg,
		Result:   nil,
		Error:    nil,
	}

	var host *config.SSHHost
	var err error
	if cfgFile != "" {
		host, err = config.GetSSHHostFromFile(hostnameArg, cfgFile)
	} else {
		host, err = config.GetSSHHost(hostnameArg)
	}
	if err != nil {
		code := 1
		errCode := "CONFIG_ERROR"
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			code = 2
			errCode = "NOT_FOUND"
		}

		resp.Error = &infoError{Code: errCode, Message: msg, Details: nil}
		writeInfoJSON(out, pretty, resp)
		return code
	}

	port, portErr := maybePort(host.Port)
	if portErr != nil {
		resp.Error = &infoError{Code: "CONFIG_ERROR", Message: "invalid port in host configuration", Details: nil}
		writeInfoJSON(out, pretty, resp)
		return 1
	}

	res := infoResult{
		CanonicalName: host.Name,
		Target: infoTarget{
			Host:     hostnameArg,
			Hostname: maybeString(host.Hostname),
			User:     maybeString(host.User),
			Port:     port,
		},
		IdentityFile:  maybeString(host.Identity),
		ProxyJump:     maybeString(host.ProxyJump),
		ProxyCommand:  maybeString(host.ProxyCommand),
		Options:       maybeString(host.Options),
		Tags:          host.Tags,
		RemoteCommand: maybeString(host.RemoteCommand),
		RequestTTY:    maybeString(host.RequestTTY),
		Source: &infoSource{
			File: host.SourceFile,
			Line: host.LineNumber,
		},
	}

	resp.OK = true
	resp.Result = &res
	writeInfoJSON(out, pretty, resp)
	return 0
}

var infoPretty bool

var infoCmd = &cobra.Command{
	Use:           "info <hostname>",
	Short:         "Print machine-readable information about a host",
	Long:          "Print machine-readable information (JSON) about a configured SSH host.",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var hosts []config.SSHHost
		var err error
		if configFile != "" {
			hosts, err = config.ParseSSHConfigFile(configFile)
		} else {
			hosts, err = config.ParseSSHConfig()
		}
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		toCompleteLower := strings.ToLower(toComplete)
		for _, host := range hosts {
			if strings.HasPrefix(strings.ToLower(host.Name), toCompleteLower) {
				completions = append(completions, host.Name)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		exitCode := runInfo(cmd.OutOrStdout(), args[0], configFile, infoPretty)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

func init() {
	infoCmd.Flags().BoolVar(&infoPretty, "pretty", false, "Pretty-print JSON output")
	RootCmd.AddCommand(infoCmd)
}
