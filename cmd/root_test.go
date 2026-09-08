package cmd

import (
	"reflect"
	"testing"
)

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantOptions globalOptions
		wantArgs    []string
	}{
		{
			name:        "global-looking command flag value",
			args:        []string{"order", "send", "--address", "--json", "--json"},
			wantOptions: globalOptions{json: true},
			wantArgs:    []string{"order", "send", "--address", "--json"},
		},
		{
			name:        "JSON before command",
			args:        []string{"--json", "status"},
			wantOptions: globalOptions{json: true},
			wantArgs:    []string{"status"},
		},
		{
			name:        "JSON after command",
			args:        []string{"items", "sent", "--page", "2", "--json"},
			wantOptions: globalOptions{json: true},
			wantArgs:    []string{"items", "sent", "--page", "2"},
		},
		{
			name:        "short help",
			args:        []string{"status", "-h"},
			wantOptions: globalOptions{help: true},
			wantArgs:    []string{"status"},
		},
		{
			name:        "long help",
			args:        []string{"status", "--help"},
			wantOptions: globalOptions{help: true},
			wantArgs:    []string{"status"},
		},
		{
			name:        "short version",
			args:        []string{"status", "-v"},
			wantOptions: globalOptions{showVersion: true},
			wantArgs:    []string{"status"},
		},
		{
			name:        "long version",
			args:        []string{"status", "--version"},
			wantOptions: globalOptions{showVersion: true},
			wantArgs:    []string{"status"},
		},
		{
			name:        "explicit false",
			args:        []string{"status", "--json=false", "-v=false"},
			wantOptions: globalOptions{},
			wantArgs:    []string{"status"},
		},
		{
			name:        "separator stops global parsing",
			args:        []string{"status", "--", "--json"},
			wantOptions: globalOptions{},
			wantArgs:    []string{"status", "--", "--json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, args, err := parseGlobalFlags(tt.args)
			if err != nil {
				t.Fatalf("parseGlobalFlags() error = %v", err)
			}
			if options != tt.wantOptions {
				t.Errorf("options = %#v, want %#v", options, tt.wantOptions)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %#v, want %#v", args, tt.wantArgs)
			}
		})
	}
}
