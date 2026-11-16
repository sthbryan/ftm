package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// legacyFlags are the flags of the stdlib-flag command line before the
// command tree. They still work, hidden from help.
type legacyFlags struct {
	port int
}

func (l *legacyFlags) register(root *cobra.Command) {
	root.Flags().IntVar(&l.port, "port", 0, "Web view port")
	_ = root.Flags().MarkHidden("port")
}

// legacyBool are the old boolean flags; each maps to a command.
var legacyBool = map[string]bool{
	"web": true, "server": true, "update": true, "check": true, "uninstall": true, "version": true,
}

// translateLegacy rewrites an old-style command line (`ftm -web -port 8080`)
// into the command it meant (`ftm serve --open --port 8080`). A command line
// that names a command is left alone apart from spelling -port as --port.
func translateLegacy(args []string) []string {
	set := map[string]bool{}
	port := ""
	var rest []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			return respellPort(args)
		}
		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch {
		case legacyBool[name]:
			on := true
			if hasValue {
				on, _ = strconv.ParseBool(value)
			}
			set[name] = on
		case name == "port":
			if hasValue {
				port = value
			} else if i+1 < len(args) {
				i++
				port = args[i]
			}
		default:
			rest = append(rest, arg)
		}
	}

	var out []string
	switch {
	case set["version"]:
		out = []string{"version"}
	case set["uninstall"]:
		out = []string{"uninstall", "--yes"}
	case set["check"]:
		out = []string{"self-update", "--check"}
	case set["update"]:
		out = []string{"self-update"}
	case set["web"]:
		out = []string{"serve", "--open"}
	case set["server"]:
		out = []string{"serve"}
	}
	if port != "" && (len(out) == 0 || out[0] == "serve") {
		out = append(out, "--port", port)
	}
	return append(out, rest...)
}

// respellPort turns the old single-dash -port into --port, which is how the
// command tree spells it.
func respellPort(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if a == "-port" || strings.HasPrefix(a, "-port=") {
			a = "-" + a
		}
		out[i] = a
	}
	return out
}
