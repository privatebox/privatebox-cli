package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yourorg/privatebox-cli/internal/api"
	"github.com/yourorg/privatebox-cli/internal/config"
	"github.com/yourorg/privatebox-cli/internal/deviceid"
)

const version = "0.2.0"

// Execute parses global flags, then dispatches to the right subcommand.
// This is a small hand-rolled router (stdlib flag package only);
// swap in spf13/cobra later if the command tree grows much larger.
func Execute() {
	fs := flag.NewFlagSet("privatebox", flag.ContinueOnError)
	fs.Usage = printHelp

	jsonOut := fs.Bool("json", false, "output machine-readable JSON")
	help := fs.Bool("help", false, "show this help message")
	showVersion := fs.Bool("version", false, "show version")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	if *help {
		printHelp()
		return
	}
	if *showVersion {
		fmt.Println("privatebox version " + version)
		return
	}

	args := fs.Args()
	if len(args) == 0 {
		printHelp()
		return
	}

	switch args[0] {
	case "auth":
		runAuth(args[1:], *jsonOut)
	case "status":
		runStatus(args[1:], *jsonOut)
	case "items":
		runItems(args[1:], *jsonOut)
	case "order":
		runOrder(args[1:], *jsonOut)
	case "meta":
		runMeta(args[1:], *jsonOut)
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printHelp()
		os.Exit(1)
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

Examples:
  privatebox auth login --email you@example.com
  privatebox order scan --items 1001 --items 1002 --destroy
  privatebox --json items sent
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
