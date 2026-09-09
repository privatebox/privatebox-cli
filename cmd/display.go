package cmd

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
)

// Treat all API strings as untrusted terminal input. Keep the JSON value intact.
func safeText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			return ' '
		}
		return r
	}, s)
}
func safeArgs(args []any) []any {
	out := make([]any, len(args))
	for i, v := range args {
		out[i] = v
		if v != nil && reflect.TypeOf(v).Kind() == reflect.String {
			out[i] = safeText(reflect.ValueOf(v).String())
		}
	}
	return out
}
func textPrintf(format string, args ...any)               { fmt.Printf(format, safeArgs(args)...) }
func textPrintln(args ...any)                             { fmt.Println(safeArgs(args)...) }
func textFprintf(w io.Writer, format string, args ...any) { fmt.Fprintf(w, format, safeArgs(args)...) }
