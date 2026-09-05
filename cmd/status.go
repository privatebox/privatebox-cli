package cmd

import (
	"fmt"

	"github.com/privatebox/privatebox-cli/internal/config"
)

func runStatus(_ []string, jsonOut bool) {
	cfg, err := config.Load()
	if err != nil {
		fatal("error loading config:", err)
	}

	token := cfg.GetToken()
	if token == "" {
		if jsonOut {
			printJSON(map[string]any{"logged_in": false})
			return
		}
		fmt.Println("Not logged in. Run 'privatebox auth login'.")
		return
	}

	if jsonOut {
		printJSON(map[string]any{
			"logged_in":    true,
			"name":         cfg.Name,
			"email":        cfg.Email,
			"api_base_url": cfg.APIBaseURL,
		})
		return
	}

	if cfg.Name != "" {
		fmt.Printf("Logged in as %s <%s>\n", cfg.Name, cfg.Email)
	} else {
		fmt.Printf("Logged in as %s\n", cfg.Email)
	}
}
