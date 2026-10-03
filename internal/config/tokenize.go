package config

import "strings"

// configLine is one line of an ssh config file, split the way OpenSSH splits it.
type configLine struct {
	keyword string   // keyword as written (case preserved)
	args    []string // arguments with quotes and escapes resolved, trailing comment dropped
	raw     string   // text after the keyword and its separator, verbatim
	rawArgs string   // raw without its trailing comment
}

// isVerbatimKeyword reports whether OpenSSH takes the rest of the line verbatim
// for this lowercased keyword, comments and quotes included.
func isVerbatimKeyword(keyword string) bool {
	switch keyword {
	case "proxycommand", "remotecommand", "localcommand", "knownhostscommand":
		return true
	}
	return false
}

// splitConfigLine splits a config line into its keyword and arguments.
// It returns false for blank lines and full-line comments.
//
// The rules follow OpenSSH (readconf.c and argv_split in misc.c):
//   - spaces and tabs separate words, and the keyword may be followed by "=";
//   - double and single quotes group words, and quoted parts glued to text are concatenated;
//   - a backslash only escapes a quote, a backslash, or (outside quotes) a space,
//     so Windows paths such as C:\Users\me\id are kept as written;
//   - a "#" at the start of a word starts a comment that runs to the end of the line.
//
// Where OpenSSH rejects the line, sshm stays lenient: an unterminated quote
// keeps the rest of the line.
func splitConfigLine(line string) (configLine, bool) {
	s := strings.Trim(line, " \t\r")
	if s == "" || s[0] == '#' {
		return configLine{}, false
	}

	end := strings.IndexAny(s, " \t=")
	if end < 0 {
		return configLine{keyword: s}, true
	}
	result := configLine{keyword: s[:end]}

	rest := strings.TrimLeft(s[end:], " \t")
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	result.raw = rest
	result.rawArgs = rest

	i := 0
	for i < len(rest) {
		for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
			i++
		}
		if i >= len(rest) {
			break
		}
		if rest[i] == '#' {
			result.rawArgs = strings.TrimRight(rest[:i], " \t")
			break
		}
		var arg strings.Builder
		var quote byte
		for ; i < len(rest); i++ {
			c := rest[i]
			if c == '\\' && i+1 < len(rest) {
				next := rest[i+1]
				if next == '\'' || next == '"' || next == '\\' || (quote == 0 && next == ' ') {
					arg.WriteByte(next)
					i++
					continue
				}
			}
			if quote == 0 && (c == ' ' || c == '\t') {
				break
			}
			if quote == 0 && (c == '"' || c == '\'') {
				quote = c
				continue
			}
			if quote != 0 && c == quote {
				quote = 0
				continue
			}
			arg.WriteByte(c)
		}
		result.args = append(result.args, arg.String())
	}
	return result, true
}
