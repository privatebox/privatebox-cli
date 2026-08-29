package cmd

import "fmt"

func runMeta(args []string, jsonOut bool) {
	if len(args) < 1 {
		fatal("usage: privatebox meta <countries|frequency>")
	}
	switch args[0] {
	case "countries":
		showCountries(jsonOut)
	case "frequency":
		showFrequencies(jsonOut)
	default:
		fatal(fmt.Sprintf("unknown meta subcommand: %s", args[0]))
	}
}

func showCountries(jsonOut bool) {
	_, client := optionalClient()
	countries, err := client.GetCountries()
	if err != nil {
		fatal("error fetching countries:", err)
	}
	if jsonOut {
		printJSON(countries)
		return
	}
	fmt.Printf("%-6s %s\n", "CODE", "NAME")
	for _, c := range countries {
		name := c.PrintableName
		if name == "" {
			name = c.Name
		}
		fmt.Printf("%-6s %s\n", c.ISO, name)
	}
}

func showFrequencies(jsonOut bool) {
	_, client := optionalClient()
	freqs, err := client.GetFrequencies()
	if err != nil {
		fatal("error fetching frequency options:", err)
	}
	if jsonOut {
		printJSON(freqs)
		return
	}
	fmt.Printf("%-6s %s\n", "ID", "NAME")
	for _, f := range freqs {
		fmt.Printf("%-6s %s\n", f.ID, f.Name)
	}
}
