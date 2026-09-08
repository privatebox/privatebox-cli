package cmd

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/privatebox/privatebox-cli/internal/api"
)

func runItems(args []string, jsonOut bool) {
	switch {
	case len(args) == 0 || args[0] == "inbox":
		rest := args
		if len(args) > 0 && args[0] == "inbox" {
			rest = args[1:]
		}
		listItems(jsonOut, "inbox", rest)
	case args[0] == "sent":
		listItems(jsonOut, "sent", args[1:])
	case args[0] == "scanned":
		listScannedItems(jsonOut, args[1:])
	case strings.HasPrefix(args[0], "-"):
		listItems(jsonOut, "inbox", args)
	default:
		fail(2, fmt.Sprintf("unknown items subcommand: %s", args[0]))
	}
}

func listItems(jsonOut bool, kind string, args []string) {
	fs := flag.NewFlagSet("items "+kind, flag.ContinueOnError)
	page := fs.Int("page", 1, "page number")
	parseFlags(fs, args)
	if *page < 1 {
		fail(2, "--page must be positive")
	}

	_, client := requireClient()

	if kind == "sent" {
		listSentItems(client, *page, jsonOut)
		return
	}

	res, err := client.GetInboxItems(*page)
	if err != nil {
		fatal("error fetching items:", err)
	}

	if jsonOut {
		printJSON(res)
		return
	}
	items := res.Items
	if len(items) == 0 {
		textPrintln("No items.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tRECEIVED\tWEIGHT\tTYPE\tSTATUS\tSCAN\tFROM\tTO")
	for _, it := range items {
		from := it.From.Text
		if it.From.Name != "" {
			from = it.From.Name
		}
		textFprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", it.ID, it.Received, it.Weight, it.Type, it.Status, it.ScanStatus, from, it.To)
	}
	w.Flush()
	printFooter(res.Pagination, res.Meta)
}

func listSentItems(client *api.Client, page int, jsonOut bool) {
	res, err := client.GetSentItems(page)
	if err != nil {
		fatal("error fetching items:", err)
	}

	if jsonOut {
		printJSON(res)
		return
	}
	if len(res.Items) == 0 {
		textPrintln("No items.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSENT\tTYPE\tWEIGHT\tSTATUS\tFROM\tTO\tDESTINATION")
	for _, it := range res.Items {
		textFprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", it.ID, it.SentAt, it.Type, it.Weight, it.Status, it.From, it.To, it.Destination)
	}
	w.Flush()
	printFooter(res.Pagination, res.Meta)
}

func listScannedItems(jsonOut bool, args []string) {
	fs := flag.NewFlagSet("items scanned", flag.ContinueOnError)
	page := fs.Int("page", 1, "page number")
	parseFlags(fs, args)
	if *page < 1 {
		fail(2, "--page must be positive")
	}

	_, client := requireClient()

	res, err := client.GetScannedItems(*page)
	if err != nil {
		fatal("error fetching scanned items:", err)
	}

	if jsonOut {
		printJSON(res)
		return
	}
	items := res.Items
	if len(items) == 0 {
		textPrintln("No scanned items.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSCANNED DATE\tTYPE\tPAGES\tFROM\tTO\tURL")
	for _, it := range items {
		from := it.Item.From.Name
		if from == "" {
			from = it.Item.From.Text
		}
		textFprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			it.ScanID,
			formatDateTime(it.ScannedDate),
			it.Item.Type,
			it.Pages,
			from,
			it.Item.To,
			clickable(it.URL),
		)
	}
	w.Flush()
	printFooter(res.Pagination, res.Meta)
}

func printFooter(p api.Pagination, m api.Meta) {
	if p.Total <= 0 {
		return
	}
	parts := []string{fmt.Sprintf("%d items", p.Total)}
	if p.CurrentPage > 0 {
		parts = append(parts, fmt.Sprintf("page %d of %d", p.CurrentPage, p.LastPage))
	}
	if m.StatusMessage != "" {
		parts = append(parts, m.StatusMessage)
	}
	textPrintln("\n" + strings.Join(parts, " | "))
	if p.LastPage > 1 {
		textPrintln("Tip: view other pages with --page N (e.g. --page 2)")
	}
}

func formatDateTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04")
}

func urlBaseName(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

// Plain URLs work when piped and do not introduce terminal escape sequences.
func clickable(u string) string { return safeText(u) }
