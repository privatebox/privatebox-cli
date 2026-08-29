package api

import (
	"encoding/json"
	"testing"
)

func TestItemDecode(t *testing.T) {
	data := []byte(`{"items":[{"id":54661,"weight":802,"type":"Large Document","status":"Arrived","scan_status":"Queued","from":{"text":"CAMPING & CARAVANS MAG"},"to":"Bugs Bunny"},{"id":54662,"weight":"32g","type":"Parcel","status":"Arrived","from":{"text":"AMAZON","name":"Amazon"},"to":"Doc. Whats Up"}],"total":2}`)
	var out itemsResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out.Items))
	}
	if out.Items[0].ID != "54661" || out.Items[0].Weight != "802" {
		t.Fatalf("bad numeric decode: %v %v", out.Items[0].ID, out.Items[0].Weight)
	}
	if out.Items[1].Weight != "32g" {
		t.Fatalf("bad string decode: %q", out.Items[1].Weight)
	}
}

func TestItemDecodeMapShape(t *testing.T) {
	data := []byte(`{"items":{"10":{"box_id":6987,"folder_id":12231,"from":{"text":"PARCEL"},"id":204968,"received":"15th Apr 2017","scan_status":"","status":"Arrived","to":"Ishan Jayamanne","type":"Parcel","weight":"20g"},"11":{"box_id":6987,"folder_id":12231,"from":{"text":"LARGE LETTER"},"id":204967,"received":"15th Apr 2017","scan_status":"","status":"Arrived","to":"Ishan Jayamanne","type":"Parcel","weight":"10g"}},"current_page":2,"total":21}`)
	var out itemsResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out.Items))
	}
	if out.Items[0].ID != "204968" || out.Items[0].Received != "15th Apr 2017" || out.Items[0].BoxID != "6987" {
		t.Fatalf("bad map decode: %v %v %v", out.Items[0].ID, out.Items[0].Received, out.Items[0].BoxID)
	}
	if out.Pagination.CurrentPage != 2 || out.Pagination.Total != 21 {
		t.Fatalf("bad pagination decode: %d %d", out.Pagination.CurrentPage, out.Pagination.Total)
	}
}

func TestSentItemDecode(t *testing.T) {
	data := []byte(`{"items_sent":[{"destination":"123M Bell Road, Waiwhetū, Lower Hutt 5010, New Zealand","from":"AA","id":43424324,"sent_at":"2nd Aug 2025","status":"Sent","to":"Ishan Jayamanne","type":"Letter","weight":"2g"}],"total":1}`)
	var out sentItemsResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}
	s := out.Items[0]
	if s.ID != "43424324" || s.From != "AA" || s.Destination == "" || s.SentAt != "2nd Aug 2025" {
		t.Fatalf("bad sent decode: %v %v %v %v", s.ID, s.From, s.Destination, s.SentAt)
	}
}

func TestScannedItemDecode(t *testing.T) {
	data := []byte(`{"items_scanned":[{"arrived_date":"13th Jan 2016","folder":"Catch all","item":{"from":{"name":"Test","text":"TEST"},"id":204566,"scan_status":"complete","status":"Arrived","to":"Ishan Jayamanne","type":"Letter","weight":"1g"},"pages":1,"scanned_date":"28th Oct 2023","scan_id":162122,"url":"https://privatebox-test1.s3.ap-southeast-2.amazonaws.com/11684/scanned/162122.pdf?X-Amz-Expires=900"}]}`)
	var out struct {
		Items []ScannedItem `json:"items_scanned"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}
	s := out.Items[0]
	if s.ScanID != "162122" || s.Pages != "1" {
		t.Fatalf("bad scan decode: %v %v", s.ScanID, s.Pages)
	}
	if s.Item.Type != "Letter" || s.Item.Weight != "1g" || s.Item.From.Name != "Test" {
		t.Fatalf("bad nested item decode: %v %v %v", s.Item.Type, s.Item.Weight, s.Item.From.Name)
	}
	if s.URL == "" {
		t.Fatal("missing url")
	}
}

func TestSendOrderResultDecode(t *testing.T) {
	data := []byte(`{"address_verified":true,"destination":{"address":"15 Beaumonts Way","address_detail":"Unit 15","city":"Auckland","country_iso":"NZ","post_code":2102,"state":"","suburb":"Manurewa"},"estimate":4.6,"estimate_max":4.6,"estimate_min":4.6,"handling_fee":0,"service":{"id":25,"name":"Economy (NZ)"}}`)
	var out SendOrderResult
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.AddressVerified || out.Service.ID != 25 || out.Service.Name != "Economy (NZ)" || out.Estimate != 4.6 {
		t.Fatalf("bad send order decode: %v %v %v %v", out.AddressVerified, out.Service.ID, out.Service.Name, out.Estimate)
	}
	if out.Destination.PostCode != "2102" || out.Destination.City != "Auckland" {
		t.Fatalf("bad destination decode: %v %v", out.Destination.PostCode, out.Destination.City)
	}
}

func TestSendCostDecode(t *testing.T) {
	data := []byte(`{"services":[{"service":{"carrier":"CourierPost","carrier_code":"CPOLP","id":21,"name":"Courier Parcel"},"qualifies_for_free_sending":true,"estimate":0.95,"estimate_before_discount":10.95,"estimate_min":0.95,"estimate_max":0.95,"handling_fee":0,"description":"- Delivery time","message":"Qualifies for free sending discount of $10.00 NZD"}],"status_code":200,"status_message":"Successfully obtained an estimate to send items"}`)
	var out struct {
		Services []SendCostService `json:"services"`
		Meta
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(out.Services))
	}
	s := out.Services[0]
	if s.Service.Name != "Courier Parcel" || s.Estimate != 0.95 || !s.QualifiesForFreeSending {
		t.Fatalf("bad service decode: %v %v %v", s.Service.Name, s.Estimate, s.QualifiesForFreeSending)
	}
	if out.Meta.StatusMessage == "" || out.Meta.StatusCode != 200 {
		t.Fatalf("bad meta decode: %d %q", out.Meta.StatusCode, out.Meta.StatusMessage)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	cases := map[string]string{
		`{"status_code":400,"status_message":{"items":["Invalid list of items or already queued!"]}}`: `items: Invalid list of items or already queued!`,
		`{"status_code":401,"status_message":"Unauthenticated"}`:                                      `Unauthenticated`,
		`not json`: `not json`,
	}
	for in, want := range cases {
		if got := apiErrorMessage([]byte(in)); got != want {
			t.Errorf("apiErrorMessage(%q) = %q, want %q", in, got, want)
		}
	}
}
