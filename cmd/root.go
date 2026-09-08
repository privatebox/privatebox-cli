package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/privatebox/privatebox-cli/internal/api"
	"github.com/privatebox/privatebox-cli/internal/config"
	"github.com/privatebox/privatebox-cli/internal/deviceid"
)

var version = "dev"

type globalOptions struct {
	json        bool
	help        bool
	showVersion bool
}

// Execute parses global flags, then dispatches to the right subcommand.
// This is a small hand-rolled router (stdlib flag package only);
// swap in spf13/cobra later if the command tree grows much larger.
func Execute() {
	options, args, err := parseGlobalFlags(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if options.help {
		printHelp()
		return
	}
	if options.showVersion {
		fmt.Println("privatebox version " + version)
		return
	}

	if len(args) == 0 {
		printHelp()
		return
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
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func parseGlobalFlags(args []string) (globalOptions, []string, error) {
	fs := flag.NewFlagSet("privatebox", flag.ContinueOnError)
	fs.Usage = printHelp

	var options globalOptions
	fs.BoolVar(&options.json, "json", false, "output machine-readable JSON")
	fs.BoolVar(&options.help, "h", false, "show this help message")
	fs.BoolVar(&options.help, "help", false, "show this help message")
	fs.BoolVar(&options.showVersion, "v", false, "show version")
	fs.BoolVar(&options.showVersion, "version", false, "show version")

	if err := fs.Parse(globalFlagsFirst(args)); err != nil {
		return globalOptions{}, nil, err
	}
	return options, fs.Args(), nil
}

// globalFlagsFirst moves recognised global flags before the command so the
// standard library flag parser accepts them on either side of a subcommand.
// Arguments following -- are left untouched.
func globalFlagsFirst(args []string) []string {
	global := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))

	for i, arg := range args {
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if isGlobalFlag(arg) {
			global = append(global, arg)
			continue
		}
		rest = append(rest, arg)
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
	case "json", "h", "help", "v", "version":
		return true
	default:
		return false
	}
}

func printHelp() {
	fmt.Print(`privatebox - PrivateBox command line client

Usage:
  privatebox [--json] <command> <subcommand> [flags]

Available Commands:
  auth login        Log in with email + password (--email you@example.com)
  auth verify-code   Submit a one-time code sent after login (--code 123456)
  auth logout        Clear the saved session and log out
  status             Show the current session
  items              List items in your inbox
  items sent         List items you've sent
  items scanned      List scanned items (clickable links to scans)
  order scan         Request a scan of items (--items ID1,ID2 [--destroy])
  order send         Create a send order (--items ID1,ID2 --receivers-name NAME --service-id ID [--search-id ID | --address ADDR] --country-iso ISO [--add-new])
  order send-cost    Estimate shipping (--items ID1,ID2 --country NZ --address ADDR [--city --suburb ...])
  order destroy      Queue items for destruction (--items ID1,ID2)
  meta countries     List reference country codes
  meta frequency     List reference scan-frequency options

Global Flags:
  --json           Output machine-readable JSON instead of tables/text
  -h, --help       Show this help message
  -v, --version    Show version

Global flags may be placed before or after the command.

Examples:
  privatebox auth login --email you@example.com
  privatebox order scan --items 1001 --items 1002 --destroy
  privatebox items sent --json
`)
}

// --- shared helpers used across subcommand files ---

func fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
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
