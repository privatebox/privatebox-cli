package cmd

import (
	"github.com/privatebox/privatebox-cli/internal/config"
)

func runStatus(args []string, jsonOut bool) {
	noFlags("status", args)
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}

	token := cfg.GetToken()
	if token == "" {
		if jsonOut {
			printJSON(map[string]any{"session_saved": false, "server_verified": false})
			return
		}
		textPrintln("Not logged in. Run 'privatebox auth login'.")
		return
	}

	if jsonOut {
		printJSON(map[string]any{
			"session_saved":         true,
			"server_verified":       false,
			"verification_required": cfg.VerificationRequired,
			"name":                  cfg.Name,
			"email":                 cfg.Email,
			"api_base_url":          cfg.APIBaseURL,
		})
		return
	}

	textPrintln("Local session only; not checked with the server.")
	if cfg.VerificationRequired {
		textPrintln("Verification required.")
	}
	if cfg.Name != "" {
		textPrintf("Saved session for %s <%s>\n", cfg.Name, cfg.Email)
	} else {
		textPrintf("Saved session for %s\n", cfg.Email)
	}
}
