package cmd

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/privatebox/privatebox-cli/internal/api"
	"github.com/privatebox/privatebox-cli/internal/terminal"
)

func runOrder(args []string, jsonOut bool) {
	if len(args) < 1 {
		groupHelp("order", "scan|send|send-cost|destroy")
		return
	}
	switch args[0] {
	case "scan":
		runOrderScan(args[1:], jsonOut)
	case "send":
		runOrderSend(args[1:], jsonOut)
	case "send-cost":
		runOrderSendCost(args[1:], jsonOut)
	case "destroy":
		runOrderDestroy(args[1:], jsonOut)
	default:
		fatal(fmt.Sprintf("unknown order subcommand: %s", args[0]))
	}
}

func runOrderScan(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order scan", flag.ContinueOnError)
	items := itemFlag(fs)
	yes := fs.Bool("yes", false, "confirm destruction without prompting")
	destroy := fs.Bool("destroy", false, "destroy items automatically after scanning")
	parseFlags(fs, args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}

	if *destroy {
		confirmDestruction(*yes, ids)
	}
	_, client := requireClient()
	order, err := client.RequestScan(ids, *destroy)
	if err != nil {
		fatal("scan request failed:", err)
	}

	if jsonOut {
		printJSON(order)
		return
	}
	textPrintf("Scan requested for %d item(s). %s\n", len(ids), order.StatusMessage)
	if *destroy {
		textPrintln("Items will be destroyed automatically after scanning.")
	}
}

func runOrderSendCost(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order send-cost", flag.ContinueOnError)
	items := itemFlag(fs)
	country := fs.String("country", "", "destination country ISO code (e.g. NZ)")
	address := fs.String("address", "", "street address")
	addressDetail := fs.String("address-detail", "", "address detail (e.g. unit number)")
	suburb := fs.String("suburb", "", "suburb")
	city := fs.String("city", "", "city")
	state := fs.String("state", "", "state")
	postCode := fs.String("post-code", "", "postal code (text, including leading zeroes)")
	parseFlags(fs, args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}
	if strings.TrimSpace(*country) == "" {
		fatal("--country is required")
	}
	if strings.TrimSpace(*address) == "" {
		fatal("--address is required")
	}

	_, client := requireClient()
	res, err := client.GetSendCost(api.SendCostRequest{
		Country: *country,
		Items:   ids,
		Address: api.SendAddress{
			Address:       *address,
			AddressDetail: *addressDetail,
			Suburb:        *suburb,
			City:          *city,
			State:         *state,
			PostCode:      *postCode,
			CountryISO:    *country,
		},
	})
	if err != nil {
		fatal("send cost request failed:", err)
	}

	if jsonOut {
		printJSON(res)
		return
	}
	if len(res.Services) == 0 {
		textPrintln("No shipping services returned.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE ID\tCARRIER\tSERVICE\tESTIMATE\tBEFORE DISC\tFREE SENDING\tMESSAGE")
	for _, s := range res.Services {
		textFprintf(w, "%d\t%s\t%s\t%.2f\t%.2f\t%v\t%s\n",
			s.Service.ID, s.Service.Carrier, s.Service.Name, s.Estimate, s.EstimateBeforeDiscount, s.QualifiesForFreeSending, s.Message)
	}
	w.Flush()
	if res.Meta.StatusMessage != "" {
		textPrintln("\n" + res.Meta.StatusMessage)
	}
}

func parseItemIDs(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("empty item ID")
		}
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid item ID %q", p)
		}
		for _, existing := range ids {
			if n == existing {
				return nil, fmt.Errorf("duplicate item ID %d", n)
			}
		}
		ids = append(ids, n)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no item IDs provided")
	}
	return ids, nil
}

func runOrderSend(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order send", flag.ContinueOnError)
	items := itemFlag(fs)
	receiversName := fs.String("receivers-name", "", "recipient's name")
	serviceID := fs.Int("service-id", 0, "shipping service ID")
	addNew := fs.Bool("add-new", false, "create a new address entry")
	searchID := fs.String("search-id", "", "saved address search ID")
	countryISO := fs.String("country-iso", "", "destination country ISO code (e.g. NZ)")
	address := fs.String("address", "", "street address")
	addressDetail := fs.String("address-detail", "", "address detail (e.g. unit number)")
	suburb := fs.String("suburb", "", "suburb")
	city := fs.String("city", "", "city")
	state := fs.String("state", "", "destination state")
	postCode := fs.String("post-code", "", "postal code (text, including leading zeroes)")
	parseFlags(fs, args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}
	if strings.TrimSpace(*receiversName) == "" {
		fatal("--receivers-name is required")
	}
	if *serviceID <= 0 {
		fatal("--service-id is required")
	}
	if strings.TrimSpace(*countryISO) == "" {
		fatal("--country-iso is required")
	}
	if strings.TrimSpace(*searchID) == "" && strings.TrimSpace(*address) == "" {
		fatal("either --search-id or --address is required")
	}

	_, client := requireClient()
	result, err := client.CreateSendOrder(api.SendOrderRequest{
		Items:         ids,
		ReceiversName: *receiversName,
		ServiceID:     *serviceID,
		AddNew:        *addNew,
		Address: api.PlaceAddress{
			SearchID:      strPtr(*searchID),
			Address:       strPtr(*address),
			AddressDetail: strPtr(*addressDetail),
			Suburb:        strPtr(*suburb),
			City:          strPtr(*city),
			State:         strPtr(*state),
			CountryISO:    *countryISO,
			PostCode:      strPtr(*postCode),
		},
	})
	if err != nil {
		fatal("send order failed:", err)
	}

	if jsonOut {
		printJSON(result)
		return
	}
	textPrintf("Send order created for %d item(s).\n", len(ids))
	textPrintf("Service: %s\n", result.Service.Name)
	if result.AddressVerified {
		textPrintln("Address verified: yes")
	} else {
		textPrintln("Address verified: no")
	}
	if d := result.Destination; d.Address != "" {
		line := d.Address
		if d.Suburb != "" {
			line += ", " + d.Suburb
		}
		if d.City != "" {
			line += ", " + d.City
		}
		if d.PostCode != "" {
			line += " " + string(d.PostCode)
		}
		textPrintf("Destination: %s\n", line)
	}
	textPrintf("Estimated cost: $%.2f (range $%.2f - $%.2f)\n", result.Estimate, result.EstimateMin, result.EstimateMax)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func runOrderDestroy(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order destroy", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "confirm destruction without prompting")
	items := itemFlag(fs)
	parseFlags(fs, args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}

	confirmDestruction(*yes, ids)
	_, client := requireClient()
	result, err := client.RequestDestroy(ids)
	if err != nil {
		fatal("destroy request failed:", err)
	}

	if jsonOut {
		printJSON(result)
		return
	}
	textPrintf("Destruction queued for %d item(s).\n", len(ids))
	if result.StatusMessage != "" {
		textPrintln(result.StatusMessage)
	}
}

func itemFlag(fs *flag.FlagSet) *string {
	value := new(string)
	fs.Func("items", "positive item IDs, comma-separated or repeated", func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("empty item IDs")
		}
		if *value != "" {
			*value += ","
		}
		*value += s
		return nil
	})
	return value
}
func confirmDestruction(yes bool, ids []int) {
	if yes {
		return
	}
	if currentOptions.json || currentOptions.noInput || !terminal.IsTerminal(os.Stdin) {
		fatal("destruction requires explicit --yes when not interactive")
	}
	fmt.Fprintf(os.Stderr, "Destroy items %v? This is irreversible. Type yes to continue: ", ids)
	answer, err := terminal.ReadLine()
	if err != nil || answer != "yes" {
		fatal("destruction cancelled; no request sent")
	}
}
