package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEndpointPolicy(t *testing.T) {
	for _, raw := range []string{"http://api.privatebox.co.nz/v1", "https://user:pass@example.invalid", "https://example.invalid/?q=x", "https://example.invalid/#fragment", "file:///tmp/api"} {
		if _, err := CanonicalBaseURL(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"http://127.0.0.1:3456/v1", "http://[::1]:3456/v1", "https://api.privatebox.co.nz/v1"} {
		if _, err := CanonicalBaseURL(raw); err != nil {
			t.Error(err)
		}
	}
}
func TestAuthenticatedRedirectIsNeverFollowed(t *testing.T) {
	hits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	_, err := NewClient(source.URL, "synthetic", "test-device").GetInboxItems(1)
	if err == nil || hits != 0 {
		t.Fatal("redirect policy failed", err, hits)
	}
}
func TestScanAcknowledgementAndPostcodeContract(t *testing.T) {
	var postcode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "privatebox/") {
			t.Error("missing CLI User-Agent")
		}
		if r.URL.Path == "/order/scan" {
			w.Write([]byte(`{"status_code":200,"status_message":"Scan ordered"}`))
			return
		}
		var body struct {
			Address struct {
				PostCode string `json:"post_code"`
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		postcode = body.Address.PostCode
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "synthetic", "")
	ack, err := c.RequestScan([]int{1}, false)
	if err != nil || ack.StatusMessage != "Scan ordered" {
		t.Fatal(ack, err)
	}
	for _, value := range []string{"0600", "SW1A 1AA"} {
		_, err := c.GetSendCost(SendCostRequest{Address: SendAddress{PostCode: value}})
		if err != nil || postcode != value {
			t.Fatal("postcode changed", postcode, err)
		}
	}
}
func TestWriteTimeoutIsUncertainAndNotRetried(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "synthetic", "")
	c.httpClient.Timeout = 10 * time.Millisecond
	_, err := c.RequestDestroy([]int{1})
	if err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatal(err)
	}
	srv.Close()
	if requests != 1 {
		t.Fatal("order was retried", requests)
	}
}
