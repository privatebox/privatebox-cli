package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/privatebox/privatebox-cli/internal/api"
	"github.com/privatebox/privatebox-cli/internal/config"
	"github.com/privatebox/privatebox-cli/internal/deviceid"
)

var version = "dev"
var currentOptions globalOptions

type globalOptions struct {
	json        bool
	help        bool
	showVersion bool
	noInput     bool
	readOnly    bool
}

// Execute parses global flags, then dispatches to the right subcommand.
// This is a small hand-rolled router (stdlib flag package only);
// swap in spf13/cobra later if the command tree grows much larger.
func Execute() {
	options, args, err := parseGlobalFlags(os.Args[1:])

	currentOptions = options
	api.UserAgent = "privatebox/" + version
	if err != nil {
		fail(2, err.Error())
	}
	if options.showVersion {
		if options.json {
			printJSON(map[string]string{"version": version})
		} else {
			fmt.Println("privatebox version " + version)
		}
		return
	}
	if len(args) == 0 {
		printHelp()
		return
	}
	if args[0] == "help" {
		currentOptions.help = true
		args = args[1:]
		if len(args) == 0 {
			printHelp()
			return
		}
	}
	if options.readOnly && len(args) > 1 && args[0] == "order" && args[1] != "send-cost" {
		fatal("--read-only forbids order creation")
	}

	switch args[0] {
	case "auth":
		runAuth(args[1:], options.json)
	case "status":
		runStatus(args[1:], options.json)
	case "items":
		runItems(args[1:], options.json)
	case "order":
		runOrder(args[1:], options.json)
	case "meta":
		runMeta(args[1:], options.json)
	case "help":
		printHelp()
	default:
		fail(2, "unknown command: "+args[0])
	}
}

func parseGlobalFlags(args []string) (globalOptions, []string, error) {
	fs := flag.NewFlagSet("privatebox", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	var options globalOptions
	fs.BoolVar(&options.noInput, "no-input", false, "never prompt for input")
	fs.BoolVar(&options.readOnly, "read-only", false, "refuse order creation (not an API permission boundary)")
	fs.BoolVar(&options.json, "json", false, "output machine-readable JSON")
	fs.BoolVar(&options.help, "h", false, "show this help message")
	fs.BoolVar(&options.help, "help", false, "show this help message")
	fs.BoolVar(&options.showVersion, "v", false, "show version")
	fs.BoolVar(&options.showVersion, "version", false, "show version")

	ordered := globalFlagsFirst(args)
	if err := fs.Parse(ordered); err != nil {
		// Preserve JSON diagnostics even when an earlier global flag is invalid.
		for _, arg := range ordered {
			if !strings.HasPrefix(arg, "-") || arg == "--" {
				break
			}
			if arg == "--json" {
				options.json = true
			}
			if strings.HasPrefix(arg, "--json=") {
				if v, e := strconv.ParseBool(strings.TrimPrefix(arg, "--json=")); e == nil {
					options.json = v
				}
			}
		}
		return options, nil, err
	}
	return options, fs.Args(), nil
}

// globalFlagsFirst moves recognised global flags before the command so the
// standard library flag parser accepts them on either side of a subcommand.
// Arguments following -- are left untouched.
func globalFlagsFirst(args []string) []string {
	global := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if isGlobalFlag(arg) {
			global = append(global, arg)
			continue
		}
		rest = append(rest, arg)
		// A command flag value may itself look like a global flag.
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if !hasValue && strings.HasPrefix(arg, "-") && valueFlags[name] && i+1 < len(args) {
			i++
			rest = append(rest, args[i])
		}
	}

	return append(global, rest...)
}

func isGlobalFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") || arg == "-" {
		return false
	}

	name := strings.TrimPrefix(arg, "-")
	name = strings.TrimPrefix(name, "-")
	name, _, _ = strings.Cut(name, "=")

	switch name {
	case "json", "h", "help", "v", "version", "no-input", "read-only":
		return true
	default:
		return false
	}
}

const rootHelp = `privatebox - PrivateBox command line client

Usage:
  privatebox [--json] <command> <subcommand> [flags]

Available Commands:
  auth login        Log in with email + password (--email you@example.com)
  auth verify-code   Submit a one-time code interactively or with --code-stdin
  auth logout        Clear this endpoint's local session (not server revocation)
  status             Show saved session metadata (not a server validation)
  items              List items in your inbox
  items sent         List items you've sent
  items scanned      List scanned items with plain scan URLs
  order scan         Request a scan of items (--items ID1,ID2 [--destroy])
  order send         Create a send order (--items ID1,ID2 --receivers-name NAME --service-id ID [--search-id ID | --address ADDR] --country-iso ISO [--add-new])
  order send-cost    Estimate shipping (--items ID1,ID2 --country NZ --address ADDR [--city --suburb ...])
  order destroy      Queue items for destruction (--items ID1,ID2)
  meta countries     List reference country codes
  meta frequency     List reference scan-frequency options

Global Flags:
  --json           Output JSON; never prompt interactively
  --no-input       Never prompt; fail when input is required
  --read-only      Refuse order creation (a local guard, not API authorisation)
  -h, --help       Show this help message
  -v, --version    Show version

Global flags may be placed before or after the command.

Examples:
  privatebox auth login --email you@example.com
  privatebox order scan --items 1001 --items 1002 --destroy --yes
  privatebox items sent --json
`

func printHelp() {
	if currentOptions.json {
		printJSON(map[string]string{"help": rootHelp})
		return
	}
	fmt.Print(rootHelp)
}

// --- shared helpers used across subcommand files ---

func fatal(args ...interface{}) {
	fail(1, fmt.Sprint(args...))
}

func printJSON(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("error encoding JSON:", err)
	}
	fmt.Println(string(b))
}

// requireClient loads config and fails if the user isn't logged in.
func requireClient() (*config.Config, *api.Client) {
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}
	token := cfg.GetToken()
	if cfg.VerificationRequired {
		fatal("Session requires verification. Run privatebox auth verify-code.")
	}
	if token == "" {
		fatal("Not logged in. Run 'privatebox auth login' first.")
	}
	return cfg, api.NewClient(cfg.APIBaseURL, token, deviceid.ID())
}

// optionalClient loads config and returns a client whether or not the
// user is logged in — used for public/reference endpoints like meta.
func optionalClient() (*config.Config, *api.Client) {
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}
	return cfg, api.NewClient(cfg.APIBaseURL, cfg.GetToken(), deviceid.ID())
}

var valueFlags = map[string]bool{
	"email": true, "page": true, "items": true, "country": true, "address": true,
	"address-detail": true, "suburb": true, "city": true, "state": true, "post-code": true,
	"receivers-name": true, "service-id": true, "search-id": true, "country-iso": true,
}

func fail(code int, message string) {
	if currentOptions.json {
		kind := "command_failed"
		if code == 2 {
			kind = "invalid_arguments"
		}
		b, _ := json.Marshal(map[string]any{"error": map[string]string{"code": kind, "message": message}})
		fmt.Println(string(b))
	} else {
		fmt.Fprintln(os.Stderr, safeText(message))
	}
	os.Exit(code)
}

// parseFlags rejects every leftover argument before callers can contact the API.
// Duplicate singleton options are errors; only --items explicitly accumulates.
func parseFlags(fs *flag.FlagSet, args []string) {
	fs.SetOutput(io.Discard)
	if currentOptions.help {
		var flags []map[string]string
		fs.VisitAll(func(f *flag.Flag) {
			flags = append(flags, map[string]string{"name": f.Name, "description": f.Usage, "default": f.DefValue})
		})
		if currentOptions.json {
			printJSON(map[string]any{"usage": "privatebox " + fs.Name() + " [flags]", "flags": flags})
		} else {
			fmt.Println("Usage: privatebox " + fs.Name() + " [flags]")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			fmt.Println("Global flags: --json --no-input --read-only --help --version")
		}
		os.Exit(0)
	}
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name, _, inline := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if f == nil {
			fail(2, "unknown option: "+arg)
		}
		if seen[name] && name != "items" {
			fail(2, "duplicate option: --"+name)
		}
		seen[name] = true
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); (!ok || !b.IsBoolFlag()) && !inline {
			i++
		}
	}
	if err := fs.Parse(args); err != nil {
		fail(2, err.Error())
	}
	if fs.NArg() != 0 {
		fail(2, "unexpected positional arguments: "+strings.Join(fs.Args(), " "))
	}
}
func noFlags(name string, args []string) {
	parseFlags(flag.NewFlagSet(name, flag.ContinueOnError), args)
}

func groupHelp(name, commands string) {
	message := "Usage: privatebox " + name + " <" + commands + ">; use <subcommand> --help for options."
	if currentOptions.help {
		if currentOptions.json {
			printJSON(map[string]string{"help": message})
		} else {
			fmt.Println(message)
		}
		return
	}
	fail(2, message)
}
