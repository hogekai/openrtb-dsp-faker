package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kaiho/openrtb-dsp-faker/internal/config"
	"github.com/kaiho/openrtb-dsp-faker/internal/handler"
)

func main() {
	configPath := flag.String("config", "configs/default.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("starting faker DSP",
		"port", cfg.Server.Port,
		"bid_rate", cfg.Behavior.BidRate,
		"latency_min_ms", cfg.Behavior.Latency.MinMS,
		"latency_max_ms", cfg.Behavior.Latency.MaxMS,
		"error_rate", cfg.Behavior.Error.Rate,
		"timeout_rate", cfg.Behavior.TimeoutRate,
	)

	mux := http.NewServeMux()
	bidHandler := handler.NewBidHandler(cfg, logger)
	mux.Handle("POST /openrtb2/bid", bidHandler)
	mux.HandleFunc("GET /health", handler.HealthHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutMS) * time.Millisecond,
	}

	logger.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
