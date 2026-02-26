package auction

import (
	"testing"

	"github.com/kaiho/openrtb-dsp-faker/internal/config"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func int64p(v int64) *int64 { return &v }

func newTestConfig() *config.Config {
	return &config.Config{
		Behavior: config.BehaviorConfig{
			BidRate: 1.0, // always bid for deterministic tests
		},
		Bidding: config.BiddingConfig{
			Price:             config.PriceConfig{Min: 1.00, Max: 10.00},
			Currency:          "USD",
			Seat:              "test-seat",
			AdvertiserDomains: []string{"test-advertiser.com"},
			ADMTemplate:       `<div>Ad ${CREATIVE_ID}</div>`,
		},
	}
}

func TestGeneratePrice_Range(t *testing.T) {
	tests := []struct {
		name string
		min  float64
		max  float64
	}{
		{"normal range", 1.00, 10.00},
		{"narrow range", 5.00, 5.50},
		{"same min max", 3.00, 3.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				price := GeneratePrice(tt.min, tt.max)
				if price < tt.min || price > tt.max {
					t.Errorf("price %f out of range [%f, %f]", price, tt.min, tt.max)
				}
			}
		})
	}
}

func TestGeneratePrice_TwoDecimals(t *testing.T) {
	for i := 0; i < 100; i++ {
		price := GeneratePrice(0.01, 99.99)
		rounded := float64(int(price*100+0.5)) / 100
		if price != rounded {
			t.Errorf("price %f is not rounded to 2 decimal places", price)
		}
	}
}

func TestBidder_GenerateBids_RespectsBidFloor(t *testing.T) {
	cfg := newTestConfig()
	cfg.Bidding.Price.Max = 2.00 // max price is $2
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID: "test-1",
		Imp: []openrtb2.Imp{
			{
				ID:       "imp-1",
				Banner:   &openrtb2.Banner{W: int64p(300), H: int64p(250)},
				BidFloor: 100.00, // floor way above max price
			},
		},
	}

	results := bidder.GenerateBids(req)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].NoBid {
		t.Error("expected no-bid when bidfloor exceeds max price")
	}
}

func TestBidder_GenerateBids_BidsAboveFloor(t *testing.T) {
	cfg := newTestConfig()
	cfg.Bidding.Price.Min = 5.00
	cfg.Bidding.Price.Max = 10.00
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID: "test-1",
		Imp: []openrtb2.Imp{
			{
				ID:       "imp-1",
				Banner:   &openrtb2.Banner{W: int64p(300), H: int64p(250)},
				BidFloor: 1.00, // floor below min price
			},
		},
	}

	bidCount := 0
	for i := 0; i < 100; i++ {
		results := bidder.GenerateBids(req)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].NoBid {
			bidCount++
			if results[0].Bid.Price < 1.00 {
				t.Errorf("bid price %f is below bidfloor 1.00", results[0].Bid.Price)
			}
		}
	}
	if bidCount == 0 {
		t.Error("expected at least some bids when floor is below price range")
	}
}

func TestBidder_GenerateBids_MultipleImps(t *testing.T) {
	cfg := newTestConfig()
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID: "test-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{W: int64p(300), H: int64p(250)}},
			{ID: "imp-2", Banner: &openrtb2.Banner{W: int64p(728), H: int64p(90)}},
			{ID: "imp-3", Banner: &openrtb2.Banner{W: int64p(320), H: int64p(50)}},
		},
	}

	results := bidder.GenerateBids(req)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestBidder_BuildResponse_AllNoBid(t *testing.T) {
	cfg := newTestConfig()
	bidder := NewBidder(cfg)

	results := []BidResult{
		{NoBid: true},
		{NoBid: true},
	}

	resp := bidder.BuildResponse("req-1", results)
	if resp != nil {
		t.Error("expected nil response when all imps are no-bid")
	}
}

func TestBidder_BuildResponse_WithBids(t *testing.T) {
	cfg := newTestConfig()
	bidder := NewBidder(cfg)

	results := []BidResult{
		{Bid: &openrtb2.Bid{ID: "bid-1", ImpID: "imp-1", Price: 5.00}},
		{NoBid: true},
		{Bid: &openrtb2.Bid{ID: "bid-3", ImpID: "imp-3", Price: 3.00}},
	}

	resp := bidder.BuildResponse("req-1", results)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "req-1" {
		t.Errorf("expected response ID req-1, got %s", resp.ID)
	}
	if len(resp.SeatBid) != 1 {
		t.Fatalf("expected 1 seatbid, got %d", len(resp.SeatBid))
	}
	if len(resp.SeatBid[0].Bid) != 2 {
		t.Errorf("expected 2 bids, got %d", len(resp.SeatBid[0].Bid))
	}
	if resp.SeatBid[0].Seat != "test-seat" {
		t.Errorf("expected seat test-seat, got %s", resp.SeatBid[0].Seat)
	}
}

func TestBidder_ShouldBid_AlwaysBid(t *testing.T) {
	cfg := newTestConfig()
	cfg.Behavior.BidRate = 1.0
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{ID: "test"}
	for i := 0; i < 100; i++ {
		if !bidder.ShouldBid(req) {
			t.Error("expected always bid with bid_rate=1.0")
		}
	}
}

func TestBidder_ShouldBid_NeverBid(t *testing.T) {
	cfg := newTestConfig()
	cfg.Behavior.BidRate = 0.0
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{ID: "test"}
	for i := 0; i < 100; i++ {
		if bidder.ShouldBid(req) {
			t.Error("expected never bid with bid_rate=0.0")
		}
	}
}

func TestBidder_ShouldBid_DomainOverrideNoBid(t *testing.T) {
	cfg := newTestConfig()
	cfg.Behavior.BidRate = 1.0
	cfg.Custom.DomainOverrides = []config.DomainOverride{
		{Domain: "blocked.com", NoBid: true},
	}
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID:   "test",
		Site: &openrtb2.Site{Domain: "blocked.com"},
	}
	for i := 0; i < 100; i++ {
		if bidder.ShouldBid(req) {
			t.Error("expected no-bid for blocked domain")
		}
	}
}

func TestBidder_BidRateDistribution(t *testing.T) {
	cfg := newTestConfig()
	cfg.Behavior.BidRate = 0.5
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{ID: "test"}
	bidCount := 0
	iterations := 10000
	for i := 0; i < iterations; i++ {
		if bidder.ShouldBid(req) {
			bidCount++
		}
	}

	ratio := float64(bidCount) / float64(iterations)
	// Allow 5% tolerance
	if ratio < 0.45 || ratio > 0.55 {
		t.Errorf("bid rate distribution out of expected range: got %f, expected ~0.5", ratio)
	}
}

func TestBidder_CreativeSize_FromImp(t *testing.T) {
	cfg := newTestConfig()
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID: "test",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{W: int64p(728), H: int64p(90)}},
		},
	}

	results := bidder.GenerateBids(req)
	if results[0].NoBid {
		t.Fatal("unexpected no-bid")
	}
	if results[0].Bid.W != 728 || results[0].Bid.H != 90 {
		t.Errorf("expected 728x90, got %dx%d", results[0].Bid.W, results[0].Bid.H)
	}
}

func TestBidder_CreativeSize_FromConfig(t *testing.T) {
	cfg := newTestConfig()
	cfg.Custom.CreativeSizes = []config.CreativeSize{
		{W: 300, H: 250},
	}
	bidder := NewBidder(cfg)

	req := &openrtb2.BidRequest{
		ID: "test",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{}}, // no size specified
		},
	}

	results := bidder.GenerateBids(req)
	if results[0].NoBid {
		t.Fatal("unexpected no-bid")
	}
	if results[0].Bid.W != 300 || results[0].Bid.H != 250 {
		t.Errorf("expected 300x250 from config, got %dx%d", results[0].Bid.W, results[0].Bid.H)
	}
}
