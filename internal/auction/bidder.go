package auction

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/kaiho/openrtb-dsp-faker/internal/config"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// BidResult holds the result for a single impression.
type BidResult struct {
	Bid   *openrtb2.Bid
	NoBid bool
}

// Bidder generates bid responses based on configuration.
type Bidder struct {
	cfg *config.Config
}

// NewBidder creates a new Bidder with the given config.
func NewBidder(cfg *config.Config) *Bidder {
	return &Bidder{cfg: cfg}
}

// ShouldBid determines whether the DSP should bid on this request,
// considering imp-type-specific bid rates and domain overrides.
func (b *Bidder) ShouldBid(req *openrtb2.BidRequest) bool {
	bidRate := b.cfg.Behavior.BidRate

	// Domain/bundle override: check for no_bid or bid_rate override
	if override := b.findDomainOverride(req); override != nil {
		if override.NoBid {
			return false
		}
		if override.BidRate != nil {
			bidRate = *override.BidRate
		}
	}

	return rand.Float64() < bidRate
}

// GenerateBids creates bids for each impression in the request.
func (b *Bidder) GenerateBids(req *openrtb2.BidRequest) []BidResult {
	results := make([]BidResult, 0, len(req.Imp))
	override := b.findDomainOverride(req)

	for i := range req.Imp {
		imp := &req.Imp[i]
		result := b.bidForImp(imp, override)
		results = append(results, result)
	}
	return results
}

// bidForImp generates a bid for a single impression.
func (b *Bidder) bidForImp(imp *openrtb2.Imp, override *config.DomainOverride) BidResult {
	// Check imp-type-specific bid rate
	if !b.shouldBidForImpType(imp) {
		return BidResult{NoBid: true}
	}

	priceMin := b.cfg.Bidding.Price.Min
	priceMax := b.cfg.Bidding.Price.Max

	// Apply domain override prices if present
	if override != nil {
		if override.PriceMin != nil {
			priceMin = *override.PriceMin
		}
		if override.PriceMax != nil {
			priceMax = *override.PriceMax
		}
	}

	price := GeneratePrice(priceMin, priceMax)

	// Respect bidfloor
	floor := imp.BidFloor
	if floor > 0 && price < floor {
		return BidResult{NoBid: true}
	}

	w, h := b.resolveSize(imp)
	crid := fmt.Sprintf("crid-%s-%d", imp.ID, rand.IntN(100000))
	adm := strings.ReplaceAll(b.cfg.Bidding.ADMTemplate, "${CREATIVE_ID}", crid)

	bid := &openrtb2.Bid{
		ID:      fmt.Sprintf("bid-%s-%d", imp.ID, rand.IntN(100000)),
		ImpID:   imp.ID,
		Price:   price,
		AdM:     adm,
		CrID:    crid,
		W:       w,
		H:       h,
		ADomain: b.cfg.Bidding.AdvertiserDomains,
	}
	return BidResult{Bid: bid}
}

// shouldBidForImpType checks imp-type-specific bid rate from custom config.
func (b *Bidder) shouldBidForImpType(imp *openrtb2.Imp) bool {
	rates := b.cfg.Custom.BidRateByImpType

	if imp.Banner != nil && rates.Banner != nil {
		return rand.Float64() < *rates.Banner
	}
	if imp.Video != nil && rates.Video != nil {
		return rand.Float64() < *rates.Video
	}
	// No type-specific override: always bid (request-level bid_rate already checked)
	return true
}

// resolveSize determines the creative size from the impression or custom config.
func (b *Bidder) resolveSize(imp *openrtb2.Imp) (int64, int64) {
	// Use imp's own size if available
	if imp.Banner != nil && imp.Banner.W != nil && imp.Banner.H != nil && *imp.Banner.W > 0 && *imp.Banner.H > 0 {
		return *imp.Banner.W, *imp.Banner.H
	}
	if imp.Video != nil && imp.Video.W != nil && imp.Video.H != nil && *imp.Video.W > 0 && *imp.Video.H > 0 {
		return *imp.Video.W, *imp.Video.H
	}

	// Pick from configured creative sizes if available
	if sizes := b.cfg.Custom.CreativeSizes; len(sizes) > 0 {
		s := sizes[rand.IntN(len(sizes))]
		return int64(s.W), int64(s.H)
	}

	// Fallback
	return 300, 250
}

// findDomainOverride returns the matching domain override, if any.
func (b *Bidder) findDomainOverride(req *openrtb2.BidRequest) *config.DomainOverride {
	for i := range b.cfg.Custom.DomainOverrides {
		o := &b.cfg.Custom.DomainOverrides[i]
		if o.Domain != "" && req.Site != nil && req.Site.Domain == o.Domain {
			return o
		}
		if o.Bundle != "" && req.App != nil && req.App.Bundle == o.Bundle {
			return o
		}
	}
	return nil
}

// GeneratePrice generates a random price in the given range, rounded to 2 decimal places.
func GeneratePrice(min, max float64) float64 {
	if min >= max {
		return min
	}
	price := min + rand.Float64()*(max-min)
	// Round to 2 decimal places
	return float64(int(price*100+0.5)) / 100
}

// BuildResponse builds a complete BidResponse from bid results.
func (b *Bidder) BuildResponse(requestID string, results []BidResult) *openrtb2.BidResponse {
	var bids []openrtb2.Bid
	for _, r := range results {
		if r.Bid != nil {
			bids = append(bids, *r.Bid)
		}
	}

	if len(bids) == 0 {
		return nil
	}

	return &openrtb2.BidResponse{
		ID: requestID,
		SeatBid: []openrtb2.SeatBid{
			{
				Bid:  bids,
				Seat: b.cfg.Bidding.Seat,
			},
		},
		Cur: b.cfg.Bidding.Currency,
	}
}
