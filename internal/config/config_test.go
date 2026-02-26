package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	yaml := `
server:
  port: 8081
behavior:
  bid_rate: 0.7
  latency:
    min_ms: 10
    max_ms: 80
  error:
    rate: 0.02
    codes: [500, 503]
  timeout_rate: 0.01
bidding:
  price:
    min: 0.50
    max: 15.00
  currency: "USD"
  seat: "test-seat"
`
	path := writeTestConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8081 {
		t.Errorf("expected port 8081, got %d", cfg.Server.Port)
	}
	if cfg.Behavior.BidRate != 0.7 {
		t.Errorf("expected bid_rate 0.7, got %f", cfg.Behavior.BidRate)
	}
	if cfg.Bidding.Price.Min != 0.50 {
		t.Errorf("expected price.min 0.50, got %f", cfg.Bidding.Price.Min)
	}
	if cfg.Bidding.Seat != "test-seat" {
		t.Errorf("expected seat test-seat, got %s", cfg.Bidding.Seat)
	}
}

func TestLoad_Defaults(t *testing.T) {
	yaml := `
server:
  port: 9090
behavior:
  bid_rate: 0.5
  latency:
    min_ms: 0
    max_ms: 0
  error:
    rate: 0
  timeout_rate: 0
bidding:
  price:
    min: 1.0
    max: 10.0
`
	path := writeTestConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.ReadTimeoutMS != 5000 {
		t.Errorf("expected default read_timeout_ms 5000, got %d", cfg.Server.ReadTimeoutMS)
	}
	if cfg.Server.WriteTimeoutMS != 5000 {
		t.Errorf("expected default write_timeout_ms 5000, got %d", cfg.Server.WriteTimeoutMS)
	}
	if cfg.Bidding.Currency != "USD" {
		t.Errorf("expected default currency USD, got %s", cfg.Bidding.Currency)
	}
	if cfg.Bidding.Seat != "faker-dsp-seat-1" {
		t.Errorf("expected default seat faker-dsp-seat-1, got %s", cfg.Bidding.Seat)
	}
	if cfg.Bidding.ADMTemplate == "" {
		t.Error("expected non-empty default adm_template")
	}
	if len(cfg.Bidding.AdvertiserDomains) == 0 {
		t.Error("expected non-empty default advertiser_domains")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "invalid port (0)",
			yaml: `
server:
  port: 0
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "invalid port (70000)",
			yaml: `
server:
  port: 70000
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "bid_rate > 1",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 1.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "bid_rate < 0",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: -0.1
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "latency max < min",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 100, max_ms: 50}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "error_rate > 1",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 2.0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
		{
			name: "price max < min",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 10.0, max: 1.0}
`,
		},
		{
			name: "negative price min",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: -1.0, max: 10.0}
`,
		},
		{
			name: "timeout_rate > 1",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 1.5
bidding:
  price: {min: 1.0, max: 10.0}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestConfig(t, tt.yaml)
			_, err := Load(path)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTestConfig(t, "{{invalid yaml}}")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoad_CustomBidRateValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid banner bid rate",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
custom:
  bid_rate_by_imp_type:
    banner: 0.8
`,
			wantErr: false,
		},
		{
			name: "invalid banner bid rate",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
custom:
  bid_rate_by_imp_type:
    banner: 1.5
`,
			wantErr: true,
		},
		{
			name: "invalid video bid rate",
			yaml: `
server:
  port: 8081
behavior:
  bid_rate: 0.5
  latency: {min_ms: 0, max_ms: 0}
  error: {rate: 0}
  timeout_rate: 0
bidding:
  price: {min: 1.0, max: 10.0}
custom:
  bid_rate_by_imp_type:
    video: -0.1
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestConfig(t, tt.yaml)
			_, err := Load(path)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
