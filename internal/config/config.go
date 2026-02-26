package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for the faker DSP.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Behavior BehaviorConfig `yaml:"behavior"`
	Bidding  BiddingConfig  `yaml:"bidding"`
	Custom   CustomConfig   `yaml:"custom"`
}

// ServerConfig holds server-level settings.
type ServerConfig struct {
	Port           int `yaml:"port"`
	ReadTimeoutMS  int `yaml:"read_timeout_ms"`
	WriteTimeoutMS int `yaml:"write_timeout_ms"`
}

// BehaviorConfig controls DSP simulation behavior.
type BehaviorConfig struct {
	BidRate     float64       `yaml:"bid_rate"`
	Latency     LatencyConfig `yaml:"latency"`
	Error       ErrorConfig   `yaml:"error"`
	TimeoutRate float64       `yaml:"timeout_rate"`
}

// LatencyConfig controls simulated response latency.
type LatencyConfig struct {
	MinMS int `yaml:"min_ms"`
	MaxMS int `yaml:"max_ms"`
}

// ErrorConfig controls simulated HTTP errors.
type ErrorConfig struct {
	Rate  float64 `yaml:"rate"`
	Codes []int   `yaml:"codes"`
}

// BiddingConfig controls bid price and response content.
type BiddingConfig struct {
	Price              PriceConfig `yaml:"price"`
	Currency           string      `yaml:"currency"`
	Seat               string      `yaml:"seat"`
	AdvertiserDomains  []string    `yaml:"advertiser_domains"`
	ADMTemplate        string      `yaml:"adm_template"`
}

// PriceConfig controls bid price range.
type PriceConfig struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

// CustomConfig holds SSP-developer-friendly extended parameters.
type CustomConfig struct {
	// imp typeごとの入札率差分 (banner vs video)
	BidRateByImpType BidRateByImpType `yaml:"bid_rate_by_imp_type"`
	// 特定domain/bundleでの挙動オーバーライド
	DomainOverrides []DomainOverride `yaml:"domain_overrides"`
	// creative サイズのバリエーション
	CreativeSizes []CreativeSize `yaml:"creative_sizes"`
	// PMP deal対応
	DealSupport DealSupportConfig `yaml:"deal_support"`
	// No-Bid Reason (nbr)
	NoBidReason NoBidReasonConfig `yaml:"no_bid_reason"`
}

// BidRateByImpType allows different bid rates per impression type.
type BidRateByImpType struct {
	Banner *float64 `yaml:"banner"`
	Video  *float64 `yaml:"video"`
}

// DomainOverride allows overriding behavior for specific domains or app bundles.
type DomainOverride struct {
	// Match by site domain or app bundle
	Domain string  `yaml:"domain"`
	Bundle string  `yaml:"bundle"`
	// Override values (nil = use default)
	BidRate *float64 `yaml:"bid_rate"`
	PriceMin *float64 `yaml:"price_min"`
	PriceMax *float64 `yaml:"price_max"`
	NoBid   bool     `yaml:"no_bid"`
}

// CreativeSize represents a creative size option.
type CreativeSize struct {
	W int `yaml:"w"`
	H int `yaml:"h"`
}

// DealSupportConfig controls PMP deal behavior.
type DealSupportConfig struct {
	Enabled bool          `yaml:"enabled"`
	Deals   []DealConfig  `yaml:"deals"`
}

// DealConfig defines a single deal.
type DealConfig struct {
	DealID   string  `yaml:"deal_id"`
	BidFloor float64 `yaml:"bidfloor"`
	Seat     string  `yaml:"seat"`
}

// NoBidReasonConfig controls no-bid reason codes (OpenRTB nbr field).
type NoBidReasonConfig struct {
	Enabled bool  `yaml:"enabled"`
	// Codes to return when no-bidding (0=unknown, 1=technical error, 2=invalid request, etc.)
	Codes   []int `yaml:"codes"`
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Behavior.BidRate < 0 || c.Behavior.BidRate > 1 {
		return fmt.Errorf("behavior.bid_rate must be between 0.0 and 1.0, got %f", c.Behavior.BidRate)
	}
	if c.Behavior.Latency.MinMS < 0 {
		return fmt.Errorf("behavior.latency.min_ms must be >= 0, got %d", c.Behavior.Latency.MinMS)
	}
	if c.Behavior.Latency.MaxMS < c.Behavior.Latency.MinMS {
		return fmt.Errorf("behavior.latency.max_ms (%d) must be >= min_ms (%d)", c.Behavior.Latency.MaxMS, c.Behavior.Latency.MinMS)
	}
	if c.Behavior.Error.Rate < 0 || c.Behavior.Error.Rate > 1 {
		return fmt.Errorf("behavior.error.rate must be between 0.0 and 1.0, got %f", c.Behavior.Error.Rate)
	}
	if c.Behavior.TimeoutRate < 0 || c.Behavior.TimeoutRate > 1 {
		return fmt.Errorf("behavior.timeout_rate must be between 0.0 and 1.0, got %f", c.Behavior.TimeoutRate)
	}
	if c.Bidding.Price.Min < 0 {
		return fmt.Errorf("bidding.price.min must be >= 0, got %f", c.Bidding.Price.Min)
	}
	if c.Bidding.Price.Max < c.Bidding.Price.Min {
		return fmt.Errorf("bidding.price.max (%f) must be >= min (%f)", c.Bidding.Price.Max, c.Bidding.Price.Min)
	}
	// Validate custom bid rates
	if r := c.Custom.BidRateByImpType.Banner; r != nil {
		if *r < 0 || *r > 1 {
			return fmt.Errorf("custom.bid_rate_by_imp_type.banner must be between 0.0 and 1.0, got %f", *r)
		}
	}
	if r := c.Custom.BidRateByImpType.Video; r != nil {
		if *r < 0 || *r > 1 {
			return fmt.Errorf("custom.bid_rate_by_imp_type.video must be between 0.0 and 1.0, got %f", *r)
		}
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Server.ReadTimeoutMS == 0 {
		c.Server.ReadTimeoutMS = 5000
	}
	if c.Server.WriteTimeoutMS == 0 {
		c.Server.WriteTimeoutMS = 5000
	}
	if c.Bidding.Currency == "" {
		c.Bidding.Currency = "USD"
	}
	if c.Bidding.Seat == "" {
		c.Bidding.Seat = "faker-dsp-seat-1"
	}
	if c.Bidding.ADMTemplate == "" {
		c.Bidding.ADMTemplate = `<div class="faker-ad">Faker DSP Ad - ${CREATIVE_ID}</div>`
	}
	if len(c.Bidding.AdvertiserDomains) == 0 {
		c.Bidding.AdvertiserDomains = []string{"example-advertiser.com"}
	}
}
