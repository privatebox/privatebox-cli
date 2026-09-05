package cmd

import (
	"bufio"
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
		fatal("usage: privatebox auth <login|verify-code>")
	}
	switch args[0] {
	case "login":
		runAuthLogin(args[1:], jsonOut)
	case "verify-code":
		runAuthVerifyCode(args[1:], jsonOut)
	case "logout":
		runAuthLogout(jsonOut)
	default:
		fatal(fmt.Sprintf("unknown auth subcommand: %s", args[0]))
	}
}

func runAuthLogout(jsonOut bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}
	cfg.Name = ""
	cfg.Email = ""
	if err := cfg.ClearToken(); err != nil {
		fatal("error clearing session:", err)
	}

	if jsonOut {
		printJSON(map[string]any{"status": "logged_out"})
		return
	}
	fmt.Println("Logged out.")
}

func runAuthLogin(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)
	email := fs.String("email", "", "account email")
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}

	e := strings.TrimSpace(*email)
	if e == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Email: ")
		line, _ := reader.ReadString('\n')
		e = strings.TrimSpace(line)
	}

	fmt.Print("Password: ")
	password, err := terminal.ReadPassword()
	if err != nil {
		fatal("error reading password:", err)
	}

	client := api.NewClient(cfg.APIBaseURL, "", deviceid.ID())
	result, err := client.Login(e, password)
	if err != nil {
		fatal("login failed:", err)
	}

	if result.Email != "" {
		e = result.Email
	}
	cfg.Name = result.Name
	cfg.Email = e

	if err := cfg.SetToken(result.Token); err != nil {
		fatal("error saving session:", err)
	}

	if result.VerificationRequired {
		if jsonOut {
			printJSON(map[string]any{"status": "verification_required", "email": e})
			return
		}
		fmt.Println("A verification code was sent to your email.")
		code, err := promptCode()
		if err != nil {
			fatal("error reading code:", err)
		}
		verifyClient := api.NewClient(cfg.APIBaseURL, result.Token, deviceid.ID())
		if _, err := verifyClient.VerifyCode(code); err != nil {
			fatal("verification failed:", err)
		}
		if jsonOut {
			printJSON(map[string]any{"status": "logged_in", "email": e, "keyring": cfg.UsingKeyring})
			return
		}
		fmt.Printf("Verified. Logged in as %s\n", e)
		return
	}

	if jsonOut {
		printJSON(map[string]any{"status": "logged_in", "email": e, "keyring": cfg.UsingKeyring})
		return
	}
	fmt.Printf("Logged in as %s\n", e)
	if !cfg.UsingKeyring {
		fmt.Println("Note: no OS keyring found — session token stored in ~/.privatebox/config.json instead.")
	}
}

func promptCode() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Verification code: ")
	line, err := reader.ReadString('\n')
	return strings.TrimSpace(line), err
}

func runAuthVerifyCode(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("auth verify-code", flag.ExitOnError)
	code := fs.String("code", "", "one-time verification code")
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}
	token := cfg.GetToken()
	if token == "" {
		fatal("No session found — run 'privatebox auth login' first.")
	}

	verifyCode := strings.TrimSpace(*code)
	if verifyCode == "" {
		verifyCode, err = promptCode()
		if err != nil {
			fatal("error reading code:", err)
		}
	}
	if verifyCode == "" {
		fatal("--code is required")
	}

	client := api.NewClient(cfg.APIBaseURL, token, deviceid.ID())
	newToken, err := client.VerifyCode(verifyCode)
	if err != nil {
		fatal("verification failed:", err)
	}
	if newToken != "" {
		if err := cfg.SetToken(newToken); err != nil {
			fatal("error saving session:", err)
		}
	}

	if jsonOut {
		printJSON(map[string]any{"status": "logged_in", "email": cfg.Email, "keyring": cfg.UsingKeyring})
		return
	}
	fmt.Printf("Verified. Logged in as %s\n", cfg.Email)
}
