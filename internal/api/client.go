// Package api wraps HTTP calls to the PrivateBox API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var UserAgent = "privatebox/dev"

type Client struct {
	baseURL    string
	token      string
	deviceID   string
	httpClient *http.Client
}

func NewClient(baseURL, token, deviceID string) *Client {
	return &Client{
		baseURL:  baseURL,
		token:    token,
		deviceID: deviceID,
		httpClient: &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// doJSON sends a JSON request (if reqBody != nil) and decodes a JSON
// response into out (if out != nil). Adds the bearer token when set.
func (c *Client) doJSON(method, path string, reqBody, out interface{}) error {
	if _, err := CanonicalBaseURL(c.baseURL); err != nil {
		return err
	}
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.deviceID != "" {
		req.Header.Set("X-DeviceID", c.deviceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if strings.HasPrefix(path, "/order/") && path != "/order/send/cost" {
			return fmt.Errorf("order outcome unknown: request may have reached the server; check your inbox/orders or contact support before retrying")
		}
		return fmt.Errorf("request failed; check connectivity and try again")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 500 && strings.HasPrefix(path, "/order/") && path != "/order/send/cost" {
			return fmt.Errorf("order outcome unknown: server error; check orders before retrying")
		}
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, apiErrorMessage(msg))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(out); err != nil {
		if strings.HasPrefix(path, "/order/") && path != "/order/send/cost" {
			return fmt.Errorf("order outcome unknown: invalid acknowledgement; check orders before retrying")
		}
		return fmt.Errorf("invalid API response: %w", err)
	}
	return nil
}

// apiErrorMessage extracts a readable message from an error response
// body shaped like {"status_code": ..., "status_message": ...}, where
// status_message may be a string or an object of validation errors.
func apiErrorMessage(body []byte) string {
	var out struct {
		StatusCode    int             `json:"status_code"`
		StatusMessage json.RawMessage `json:"status_message"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return strings.TrimSpace(string(body))
	}

	var s string
	if err := json.Unmarshal(out.StatusMessage, &s); err == nil && s != "" {
		return s
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.StatusMessage, &fields); err == nil {
		var b strings.Builder
		for field, raw := range fields {
			var errs []string
			if err := json.Unmarshal(raw, &errs); err == nil {
				fmt.Fprintf(&b, "%s: %s; ", field, strings.Join(errs, ", "))
			}
		}
		if b.Len() > 0 {
			return strings.TrimSuffix(b.String(), "; ")
		}
	}

	return strings.TrimSpace(string(body))
}

// --- auth ---

type LoginResult struct {
	AccountBalance       json.RawMessage `json:"account_balance"`
	AccountID            json.Number     `json:"account_id"`
	Email                string          `json:"email"`
	Name                 string          `json:"name"`
	Token                string          `json:"token"`
	VerificationRequired bool            `json:"verification_required"`
}

// Login calls POST /login. The returned token is stored and sent as a
// Bearer token on all subsequent calls. If VerificationRequired is
// true, the email must be verified via VerifyCode before the session
// is fully usable.
func (c *Client) Login(email, password string) (LoginResult, error) {
	var out LoginResult
	err := c.doJSON(http.MethodPost, "/login", map[string]string{
		"email":    email,
		"password": password,
	}, &out)
	return out, err
}

// VerifyCode calls POST /user/validate_code to complete email
// verification. The request is authenticated with the Bearer token
// returned by Login. Returns a new token if the API issues one,
// otherwise an empty string.
func (c *Client) VerifyCode(code string) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	err := c.doJSON(http.MethodPost, "/user/validate_code", map[string]string{
		"code": code,
	}, &out)
	return out.Token, err
}

// --- items ---

// StringOrNumber accepts either a JSON string or number, preserving the
// original representation (e.g. an id of 54661 or a weight of "32g").
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = StringOrNumber(str)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(b, &num); err == nil {
		*s = StringOrNumber(num.String())
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into StringOrNumber", string(b))
}

type ItemFrom struct {
	Text string `json:"text"`
	Name string `json:"name"`
}

type Item struct {
	ID         StringOrNumber `json:"id"`
	BoxID      StringOrNumber `json:"box_id"`
	FolderID   StringOrNumber `json:"folder_id"`
	Weight     StringOrNumber `json:"weight"`
	Type       string         `json:"type"`
	Status     string         `json:"status"`
	ScanStatus string         `json:"scan_status"`
	Received   string         `json:"received"`
	From       ItemFrom       `json:"from"`
	To         string         `json:"to"`
}

// FlexList unmarshals either a JSON array or an object keyed by array
// indices, since some endpoints return a map on later pages.
type FlexList[T any] []T

func (f *FlexList[T]) UnmarshalJSON(b []byte) error {
	var arr []T
	if err := json.Unmarshal(b, &arr); err == nil {
		*f = arr
		return nil
	}

	var m map[string]T
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	numKeys := make([]int, 0, len(m))
	strKeys := make([]string, 0, len(m))
	for k := range m {
		if n, err := strconv.Atoi(k); err == nil {
			numKeys = append(numKeys, n)
		} else {
			strKeys = append(strKeys, k)
		}
	}
	sort.Ints(numKeys)
	sort.Strings(strKeys)

	out := make([]T, 0, len(m))
	for _, k := range numKeys {
		out = append(out, m[strconv.Itoa(k)])
	}
	for _, k := range strKeys {
		out = append(out, m[k])
	}
	*f = out
	return nil
}

// Pagination mirrors the pagination block in list responses.
type Pagination struct {
	CurrentPage  int    `json:"current_page"`
	FirstPageURL string `json:"first_page_url"`
	From         int    `json:"from"`
	LastPage     int    `json:"last_page"`
	LastPageURL  string `json:"last_page_url"`
	NextPageURL  string `json:"next_page_url"`
	Path         string `json:"path"`
	PerPage      int    `json:"per_page"`
	PrevPageURL  string `json:"prev_page_url"`
	To           int    `json:"to"`
	Total        int    `json:"total"`
}

// Meta mirrors the status block in list responses.
type Meta struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// ListResult wraps a page of items with its pagination and status meta.
type ListResult[T any] struct {
	Items      []T        `json:"items"`
	Pagination Pagination `json:"pagination"`
	Meta       Meta       `json:"meta"`
}

// itemsResponse is the paginated wrapper returned by the list endpoints.
type itemsResponse struct {
	Items FlexList[Item] `json:"items"`
	Pagination
	Meta
}

type scannedItemsResponse struct {
	Items FlexList[ScannedItem] `json:"items_scanned"`
	Pagination
	Meta
}

func pagedPath(p string, page int) string {
	if page > 1 {
		return fmt.Sprintf("%s?page=%d", p, page)
	}
	return p
}

func (c *Client) GetInboxItems(page int) (ListResult[Item], error) {
	var out itemsResponse
	err := c.doJSON(http.MethodGet, pagedPath("/items", page), nil, &out)
	return ListResult[Item]{Items: out.Items, Pagination: out.Pagination, Meta: out.Meta}, err
}

type SentItem struct {
	ID          StringOrNumber `json:"id"`
	Weight      StringOrNumber `json:"weight"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	From        string         `json:"from"`
	To          string         `json:"to"`
	Destination string         `json:"destination"`
	SentAt      string         `json:"sent_at"`
}

// sentItemsResponse is the paginated wrapper returned by /items/sent.
type sentItemsResponse struct {
	Items FlexList[SentItem] `json:"items_sent"`
	Pagination
	Meta
}

func (c *Client) GetSentItems(page int) (ListResult[SentItem], error) {
	var out sentItemsResponse
	err := c.doJSON(http.MethodGet, pagedPath("/items/sent", page), nil, &out)
	return ListResult[SentItem]{Items: out.Items, Pagination: out.Pagination, Meta: out.Meta}, err
}

// --- scanned items ---

type ScannedItem struct {
	ScanID      StringOrNumber `json:"scan_id"`
	ScannedDate string         `json:"scanned_date"`
	ArrivedDate string         `json:"arrived_date"`
	URL         string         `json:"url"`
	Item        Item           `json:"item"`
	Pages       StringOrNumber `json:"pages"`
}

// GetScannedItems calls GET /items/scanned.
func (c *Client) GetScannedItems(page int) (ListResult[ScannedItem], error) {
	var out scannedItemsResponse
	err := c.doJSON(http.MethodGet, pagedPath("/items/scanned", page), nil, &out)
	return ListResult[ScannedItem]{Items: out.Items, Pagination: out.Pagination, Meta: out.Meta}, err
}

// --- orders ---

type OrderResult struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// DestroyResult is the response returned by /order/destroy.
type DestroyResult struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// RequestScan calls POST /order/scan.
func (c *Client) RequestScan(itemIDs []int, destroyAfterScan bool) (OrderResult, error) {
	var out OrderResult
	err := c.doJSON(http.MethodPost, "/order/scan", map[string]interface{}{
		"items":   itemIDs,
		"destroy": destroyAfterScan,
	}, &out)
	if err == nil && out.StatusMessage == "" {
		err = fmt.Errorf("order outcome unknown: missing scan acknowledgement; check scan status before retrying")
	}
	return out, err
}

// PlaceAddress is the address object sent with a send order. Pointer
// fields marshal as JSON null when unset, matching the API's shape.
type PlaceAddress struct {
	SearchID      *string `json:"search_id"`
	ID            *int    `json:"id"`
	Address       *string `json:"address"`
	AddressDetail *string `json:"address_detail"`
	Suburb        *string `json:"suburb"`
	City          *string `json:"city"`
	CountryISO    string  `json:"country_iso"`
	State         *string `json:"state"`
	PostCode      *string `json:"post_code"`
}

type SendOrderRequest struct {
	Items         []int        `json:"items"`
	ReceiversName string       `json:"receivers_name"`
	Address       PlaceAddress `json:"address"`
	ServiceID     int          `json:"service_id"`
	AddNew        bool         `json:"add_new"`
}

type SendOrderResult struct {
	AddressVerified bool `json:"address_verified"`
	Destination     struct {
		Address       string         `json:"address"`
		AddressDetail string         `json:"address_detail"`
		City          string         `json:"city"`
		CountryISO    string         `json:"country_iso"`
		PostCode      StringOrNumber `json:"post_code"`
		State         string         `json:"state"`
		Suburb        string         `json:"suburb"`
	} `json:"destination"`
	Estimate    float64 `json:"estimate"`
	EstimateMax float64 `json:"estimate_max"`
	EstimateMin float64 `json:"estimate_min"`
	HandlingFee float64 `json:"handling_fee"`
	Service     struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"service"`
}

// CreateSendOrder calls POST /order/send.
func (c *Client) CreateSendOrder(reqBody SendOrderRequest) (SendOrderResult, error) {
	var out SendOrderResult
	err := c.doJSON(http.MethodPost, "/order/send", reqBody, &out)
	return out, err
}

// RequestDestroy calls POST /order/destroy.
func (c *Client) RequestDestroy(itemIDs []int) (DestroyResult, error) {
	var out DestroyResult
	err := c.doJSON(http.MethodPost, "/order/destroy", map[string]interface{}{
		"items":   itemIDs,
		"destroy": true,
	}, &out)
	return out, err
}

// --- send cost ---

type SendAddress struct {
	Address       string `json:"address"`
	AddressDetail string `json:"address_detail"`
	Suburb        string `json:"suburb"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostCode      string `json:"post_code"`
	CountryISO    string `json:"country_iso"`
}

type SendCostRequest struct {
	Country string      `json:"country"`
	Items   []int       `json:"items"`
	Address SendAddress `json:"address"`
}

type SendCostService struct {
	Service struct {
		Carrier     string `json:"carrier"`
		CarrierCode string `json:"carrier_code"`
		ID          int    `json:"id"`
		Name        string `json:"name"`
	} `json:"service"`
	QualifiesForFreeSending bool    `json:"qualifies_for_free_sending"`
	Estimate                float64 `json:"estimate"`
	EstimateBeforeDiscount  float64 `json:"estimate_before_discount"`
	EstimateMin             float64 `json:"estimate_min"`
	EstimateMax             float64 `json:"estimate_max"`
	HandlingFee             float64 `json:"handling_fee"`
	Description             string  `json:"description"`
	Message                 string  `json:"message"`
}

type SendCostResult struct {
	Services []SendCostService `json:"services"`
	Meta     Meta
}

// GetSendCost calls POST /order/send/cost to estimate shipping services.
func (c *Client) GetSendCost(req SendCostRequest) (SendCostResult, error) {
	var out struct {
		Services []SendCostService `json:"services"`
		Meta
	}
	err := c.doJSON(http.MethodPost, "/order/send/cost", req, &out)
	return SendCostResult{Services: out.Services, Meta: out.Meta}, err
}

// --- meta / reference data ---

type Country struct {
	ISO           string `json:"iso"`
	Name          string `json:"name"`
	PrintableName string `json:"printable_name"`
}

// GetCountries calls GET /countries.
func (c *Client) GetCountries() ([]Country, error) {
	var out struct {
		Countries []Country `json:"countries"`
	}
	err := c.doJSON(http.MethodGet, "/countries", nil, &out)
	return out.Countries, err
}

type Frequency struct {
	ID   StringOrNumber `json:"id"`
	Name string         `json:"name"`
}

// GetFrequencies calls GET /frequency.
func (c *Client) GetFrequencies() ([]Frequency, error) {
	var out struct {
		Frequencies []Frequency `json:"Frequencies"`
	}
	err := c.doJSON(http.MethodGet, "/frequency", nil, &out)
	return out.Frequencies, err
}

// CanonicalBaseURL keeps credentials scoped to an exact API endpoint, including
// its base path. Only loopback development servers may use plaintext HTTP.
func CanonicalBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid API URL")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1")) {
		return "", fmt.Errorf("API URL must use HTTPS (HTTP is permitted only on loopback for development)")
	}
	u.Host = strings.ToLower(u.Host)
	if u.Scheme == "https" && u.Port() == "443" {
		u.Host = host
		if strings.Contains(host, ":") {
			u.Host = "[" + host + "]"
		}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return strings.TrimRight(u.String(), "/"), nil
}
