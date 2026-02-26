package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kaiho/openrtb-dsp-faker/internal/config"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func newTestHandler(cfg *config.Config) *BidHandler {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewBidHandler(cfg, logger)
}

func defaultTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Port: 8081},
		Behavior: config.BehaviorConfig{
			BidRate: 1.0, // always bid
			Latency: config.LatencyConfig{MinMS: 0, MaxMS: 0},
			Error:   config.ErrorConfig{Rate: 0},
		},
		Bidding: config.BiddingConfig{
			Price:             config.PriceConfig{Min: 1.00, Max: 10.00},
			Currency:          "USD",
			Seat:              "test-seat",
			AdvertiserDomains: []string{"test.com"},
			ADMTemplate:       `<div>test</div>`,
		},
	}
}

func TestBidHandler_SuccessfulBid(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	body := `{
		"id": "req-1",
		"imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}, "bidfloor": 0.50}],
		"site": {"domain": "example.com"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp openrtb2.BidResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "req-1" {
		t.Errorf("expected response ID req-1, got %s", resp.ID)
	}
	if len(resp.SeatBid) == 0 {
		t.Fatal("expected at least one seatbid")
	}
	if len(resp.SeatBid[0].Bid) == 0 {
		t.Fatal("expected at least one bid")
	}
	bid := resp.SeatBid[0].Bid[0]
	if bid.ImpID != "imp-1" {
		t.Errorf("expected impid imp-1, got %s", bid.ImpID)
	}
	if bid.Price < 0.50 {
		t.Errorf("bid price %f should be >= bidfloor 0.50", bid.Price)
	}
}

func TestBidHandler_NoBid(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Behavior.BidRate = 0.0 // never bid
	h := newTestHandler(cfg)

	body := `{"id": "req-1", "imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}}]}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for no-bid, got %d", w.Code)
	}
}

func TestBidHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBidHandler_EmptyImp(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	body := `{"id": "req-1", "imp": []}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty imp, got %d", w.Code)
	}
}

func TestBidHandler_MissingImp(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	body := `{"id": "req-1"}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing imp, got %d", w.Code)
	}
}

func TestBidHandler_ErrorSimulation(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Behavior.Error.Rate = 1.0 // always error
	cfg.Behavior.Error.Codes = []int{503}
	h := newTestHandler(cfg)

	body := `{"id": "req-1", "imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}}]}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("expected 503 from error simulation, got %d", w.Code)
	}
}

func TestBidHandler_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	req := httptest.NewRequest(http.MethodGet, "/openrtb2/bid", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestBidHandler_AllImpsBelowFloor(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Bidding.Price.Min = 1.00
	cfg.Bidding.Price.Max = 2.00
	h := newTestHandler(cfg)

	body := `{
		"id": "req-1",
		"imp": [{"id": "imp-1", "banner": {"w": 300, "h": 250}, "bidfloor": 100.00}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 when all imps below floor, got %d", w.Code)
	}
}

func TestBidHandler_MultipleImps(t *testing.T) {
	h := newTestHandler(defaultTestConfig())

	body := `{
		"id": "req-1",
		"imp": [
			{"id": "imp-1", "banner": {"w": 300, "h": 250}},
			{"id": "imp-2", "banner": {"w": 728, "h": 90}},
			{"id": "imp-3", "banner": {"w": 320, "h": 50}}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/openrtb2/bid", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp openrtb2.BidResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.SeatBid[0].Bid) != 3 {
		t.Errorf("expected 3 bids, got %d", len(resp.SeatBid[0].Bid))
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}
