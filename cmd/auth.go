package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/privatebox/privatebox-cli/internal/api"
	"github.com/privatebox/privatebox-cli/internal/config"
	"github.com/privatebox/privatebox-cli/internal/deviceid"
	"github.com/privatebox/privatebox-cli/internal/terminal"
)

func runAuth(args []string, jsonOut bool) {
	if len(args) < 1 {
		groupHelp("auth", "login|verify-code|logout")
		return
	}
	switch args[0] {
	case "login":
		runAuthLogin(args[1:], jsonOut)
	case "verify-code":
		runAuthVerifyCode(args[1:], jsonOut)
	case "logout":
		noFlags("auth logout", args[1:])
		runAuthLogout(jsonOut)
	default:
		fail(2, "unknown auth subcommand: "+args[0])
	}
}
func runAuthLogout(jsonOut bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading session: ", err)
	}
	if err := cfg.ClearToken(); err != nil {
		fatal("error clearing session: ", err)
	}
	if jsonOut {
		printJSON(map[string]any{"status": "logged_out", "scope": "local_endpoint"})
		return
	}
	textPrintln("Local session cleared. This does not revoke the token on the server or clear other endpoints.")
}
func interactive() bool {
	return !currentOptions.json && !currentOptions.noInput && terminal.IsTerminal(os.Stdin)
}
func runAuthLogin(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	email := fs.String("email", "", "account email")
	passwordStdin := fs.Bool("password-stdin", false, "read password from one line of standard input (never put it in arguments)")
	parseFlags(fs, args)
	e := strings.TrimSpace(*email)
	if e == "" {
		if !interactive() || *passwordStdin {
			fail(2, "--email is required for non-interactive login")
		}
		fmt.Fprint(os.Stderr, "Email: ")
		line, err := terminal.ReadLine()
		if err != nil {
			fatal("error reading email: ", err)
		}
		e = strings.TrimSpace(line)
	}
	if e == "" {
		fail(2, "email is required")
	}
	var password string
	var err error
	if *passwordStdin {
		if terminal.IsTerminal(os.Stdin) {
			fail(2, "--password-stdin requires redirected input; use interactive login to hide terminal input")
		}
		password, err = terminal.ReadLine()
	} else {
		if !interactive() {
			fail(2, "non-interactive login requires --email and --password-stdin")
		}
		fmt.Fprint(os.Stderr, "Password: ")
		password, err = terminal.ReadPassword()
	}
	if err != nil {
		fatal("error reading password: ", err)
	}
	if password == "" {
		fail(2, "password must not be empty")
	}
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading session: ", err)
	}
	client := api.NewClient(cfg.APIBaseURL, "", deviceid.ID())
	result, err := client.Login(e, password)
	if err != nil {
		fatal("login failed: ", err)
	}
	if result.Email != "" {
		e = result.Email
	}
	cfg.Name, cfg.Email, cfg.VerificationRequired = result.Name, e, result.VerificationRequired
	if err := cfg.SetToken(result.Token); err != nil {
		fatal("error saving session: ", err)
	}
	warnStorage(cfg)
	if result.VerificationRequired {
		if !interactive() || *passwordStdin {
			loginOutput(cfg, "verification_required", jsonOut)
			return
		}
		fmt.Fprintln(os.Stderr, "A verification code was sent to your email.")
		code, err := promptCode()
		if err != nil {
			fatal("error reading code: ", err)
		}
		completeVerification(cfg, code)
	}
	loginOutput(cfg, "logged_in", jsonOut)
}
func warnStorage(cfg *config.Config) {
	if !cfg.UsingKeyring {
		fmt.Fprintln(os.Stderr, "Note: no OS keyring backend is available; the session is stored in an owner-only file under ~/.privatebox.")
	}
}
func loginOutput(cfg *config.Config, status string, jsonOut bool) {
	if jsonOut {
		printJSON(map[string]any{"status": status, "email": cfg.Email, "keyring": cfg.UsingKeyring})
		return
	}
	if status == "verification_required" {
		textPrintln("Verification required. Run privatebox auth verify-code.")
		return
	}
	textPrintf("Logged in as %s\n", cfg.Email)
}
func promptCode() (string, error) {
	if !interactive() {
		return "", fmt.Errorf("non-interactive verification requires --code-stdin")
	}
	fmt.Fprint(os.Stderr, "Verification code: ")
	// Verification codes are credentials too; do not echo them.
	return terminal.ReadPassword()
}
func runAuthVerifyCode(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("auth verify-code", flag.ContinueOnError)
	codeStdin := fs.Bool("code-stdin", false, "read verification code from one line of standard input")
	parseFlags(fs, args)
	var code string
	var err error
	if *codeStdin {
		if terminal.IsTerminal(os.Stdin) {
			fail(2, "--code-stdin requires redirected input")
		}
		code, err = terminal.ReadLine()
	} else {
		code, err = promptCode()
	}
	if err != nil {
		fatal("error reading code: ", err)
	}
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading session: ", err)
	}
	if cfg.GetToken() == "" {
		fatal("No saved session. Run privatebox auth login first.")
	}
	completeVerification(cfg, code)
	loginOutput(cfg, "logged_in", jsonOut)
}
func completeVerification(cfg *config.Config, code string) {
	code = strings.TrimSpace(code)
	if code == "" {
		fail(2, "verification code is required")
	}
	client := api.NewClient(cfg.APIBaseURL, cfg.GetToken(), deviceid.ID())
	token, err := client.VerifyCode(code)
	if err != nil {
		fatal("verification failed: ", err)
	}
	cfg.VerificationRequired = false
	if token != "" {
		err = cfg.SetToken(token)
	} else {
		err = cfg.SaveMeta()
	}
	if err != nil {
		fatal("error saving verified session: ", err)
	}
}
