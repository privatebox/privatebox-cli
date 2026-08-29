package cmd

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/yourorg/privatebox-cli/internal/api"
)

func runOrder(args []string, jsonOut bool) {
	if len(args) < 1 {
		fatal("usage: privatebox order <scan|send|send-cost|destroy>")
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
	fs := flag.NewFlagSet("order scan", flag.ExitOnError)
	items := fs.String("items", "", "item IDs to scan (comma-separated)")
	destroy := fs.Bool("destroy", false, "destroy items automatically after scanning")
	fs.Parse(args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
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
	fmt.Printf("Scan requested for %d item(s). Order ID: %s\n", len(ids), order.OrderID)
	if *destroy {
		fmt.Println("Items will be destroyed automatically after scanning.")
	}
}

func runOrderSendCost(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order send-cost", flag.ExitOnError)
	items := fs.String("items", "", "item IDs to send (comma-separated)")
	country := fs.String("country", "", "destination country ISO code (e.g. NZ)")
	address := fs.String("address", "", "street address")
	addressDetail := fs.String("address-detail", "", "address detail (e.g. unit number)")
	suburb := fs.String("suburb", "", "suburb")
	city := fs.String("city", "", "city")
	state := fs.String("state", "", "state")
	postCode := fs.Int("post-code", 0, "postal code")
	fs.Parse(args)

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
		fmt.Println("No shipping services returned.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CARRIER\tSERVICE\tESTIMATE\tBEFORE DISC\tFREE SENDING\tMESSAGE")
	for _, s := range res.Services {
		fmt.Fprintf(w, "%s\t%s\t%.2f\t%.2f\t%v\t%s\n",
			s.Service.Carrier, s.Service.Name, s.Estimate, s.EstimateBeforeDiscount, s.QualifiesForFreeSending, s.Message)
	}
	w.Flush()
	if res.Meta.StatusMessage != "" {
		fmt.Println("\n" + res.Meta.StatusMessage)
	}
}

func parseItemIDs(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid item ID %q", p)
		}
		ids = append(ids, n)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no item IDs provided")
	}
	return ids, nil
}

func runOrderSend(args []string, jsonOut bool) {
	fs := flag.NewFlagSet("order send", flag.ExitOnError)
	items := fs.String("items", "", "item IDs to send (comma-separated)")
	receiversName := fs.String("receivers-name", "", "recipient's name")
	serviceID := fs.Int("service-id", 0, "shipping service ID")
	addNew := fs.Bool("add-new", false, "create a new address entry")
	searchID := fs.String("search-id", "", "saved address search ID")
	countryISO := fs.String("country-iso", "", "destination country ISO code (e.g. NZ)")
	address := fs.String("address", "", "street address")
	addressDetail := fs.String("address-detail", "", "address detail (e.g. unit number)")
	suburb := fs.String("suburb", "", "suburb")
	city := fs.String("city", "", "city")
	postCode := fs.Int("post-code", 0, "postal code")
	fs.Parse(args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}
	if strings.TrimSpace(*receiversName) == "" {
		fatal("--receivers-name is required")
	}
	if *serviceID == 0 {
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
			CountryISO:    *countryISO,
			PostCode:      intPtr(*postCode),
		},
	})
	if err != nil {
		fatal("send order failed:", err)
	}

	if jsonOut {
		printJSON(result)
		return
	}
	fmt.Printf("Send order created for %d item(s).\n", len(ids))
	fmt.Printf("Service: %s\n", result.Service.Name)
	if result.AddressVerified {
		fmt.Println("Address verified: yes")
	} else {
		fmt.Println("Address verified: no")
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
		fmt.Printf("Destination: %s\n", line)
	}
	fmt.Printf("Estimated cost: $%.2f (range $%.2f - $%.2f)\n", result.Estimate, result.EstimateMin, result.EstimateMax)
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
	fs := flag.NewFlagSet("order destroy", flag.ExitOnError)
	items := fs.String("items", "", "item IDs to destroy (comma-separated)")
	fs.Parse(args)

	ids, err := parseItemIDs(*items)
	if err != nil {
		fatal("invalid --items:", err)
	}

	_, client := requireClient()
	result, err := client.RequestDestroy(ids)
	if err != nil {
		fatal("destroy request failed:", err)
	}

	if jsonOut {
		printJSON(result)
		return
	}
	fmt.Printf("Destruction queued for %d item(s).\n", len(ids))
	if result.StatusMessage != "" {
		fmt.Println(result.StatusMessage)
	}
}
