package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/kaiho/openrtb-dsp-faker/internal/auction"
	"github.com/kaiho/openrtb-dsp-faker/internal/config"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// BidHandler handles POST /openrtb2/bid requests.
type BidHandler struct {
	cfg    *config.Config
	bidder *auction.Bidder
	logger *slog.Logger
}

// NewBidHandler creates a new BidHandler.
func NewBidHandler(cfg *config.Config, logger *slog.Logger) *BidHandler {
	return &BidHandler{
		cfg:    cfg,
		bidder: auction.NewBidder(cfg),
		logger: logger,
	}
}

func (h *BidHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req openrtb2.BidRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Imp) == 0 {
		http.Error(w, "imp array is required and must not be empty", http.StatusBadRequest)
		return
	}

	// Error simulation
	if h.cfg.Behavior.Error.Rate > 0 && rand.Float64() < h.cfg.Behavior.Error.Rate {
		codes := h.cfg.Behavior.Error.Codes
		if len(codes) == 0 {
			codes = []int{500}
		}
		code := codes[rand.IntN(len(codes))]
		h.logger.Info("bid request",
			"request_id", req.ID,
			"action", "error",
			"status_code", code,
			"latency_ms", time.Since(start).Milliseconds(),
			"imp_count", len(req.Imp),
		)
		w.WriteHeader(code)
		return
	}

	// Timeout simulation
	if h.cfg.Behavior.TimeoutRate > 0 && rand.Float64() < h.cfg.Behavior.TimeoutRate {
		sleepMs := int64(100) // default extra delay
		if req.TMax > 0 {
			sleepMs = req.TMax + 100
		}
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		h.logger.Info("bid request",
			"request_id", req.ID,
			"action", "timeout",
			"latency_ms", time.Since(start).Milliseconds(),
			"imp_count", len(req.Imp),
		)
		// Still return 204 after the delay (client will have timed out)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Latency simulation
	if h.cfg.Behavior.Latency.MaxMS > 0 {
		minMs := h.cfg.Behavior.Latency.MinMS
		maxMs := h.cfg.Behavior.Latency.MaxMS
		delayMs := minMs
		if maxMs > minMs {
			delayMs = minMs + rand.IntN(maxMs-minMs+1)
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Bid/no-bid decision (request level)
	if !h.bidder.ShouldBid(&req) {
		h.logger.Info("bid request",
			"request_id", req.ID,
			"action", "no_bid",
			"latency_ms", time.Since(start).Milliseconds(),
			"imp_count", len(req.Imp),
		)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Generate bids
	results := h.bidder.GenerateBids(&req)
	resp := h.bidder.BuildResponse(req.ID, results)

	if resp == nil {
		// All imps resulted in no-bid (e.g. all below bidfloor)
		h.logger.Info("bid request",
			"request_id", req.ID,
			"action", "no_bid",
			"latency_ms", time.Since(start).Milliseconds(),
			"imp_count", len(req.Imp),
		)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Log bid prices
	for _, sb := range resp.SeatBid {
		for _, bid := range sb.Bid {
			h.logger.Info("bid request",
				"request_id", req.ID,
				"action", "bid",
				"imp_id", bid.ImpID,
				"bid_price", bid.Price,
				"latency_ms", time.Since(start).Milliseconds(),
				"imp_count", len(req.Imp),
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// HealthHandler handles GET /health requests.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
