# openrtb-dsp-faker

An OpenRTB 2.6-compliant DSP mock server. A tool for SSP/Ad Exchange developers to test and debug DSP integrations.

Reproduces real DSP behavior (latency, error rates, bidding patterns) via configuration files, providing a realistic test environment.

## Quick Start

```bash
# Build
make build

# Start with default config
./bin/faker

# Start with custom config
./bin/faker -config ./configs/custom.yaml
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/openrtb2/bid` | Receives Bid Request, returns Bid Response |
| GET | `/health` | Health check (`{"status":"ok"}`) |

## Test Request

```bash
curl -X POST http://localhost:8081/openrtb2/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-req-001",
    "imp": [
      {
        "id": "imp-1",
        "banner": {"w": 300, "h": 250},
        "bidfloor": 1.00
      }
    ],
    "site": {"domain": "example-publisher.com"},
    "tmax": 100
  }'
```

## Configuration

See `configs/default.yaml`. Key parameters:

### server

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | int | - | Listening port |
| `read_timeout_ms` | int | 5000 | HTTP read timeout (ms) |
| `write_timeout_ms` | int | 5000 | HTTP write timeout (ms) |

### behavior

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `bid_rate` | float | - | Bid probability (0.0-1.0) |
| `latency.min_ms` | int | - | Minimum response latency (ms) |
| `latency.max_ms` | int | - | Maximum response latency (ms) |
| `error.rate` | float | - | HTTP error rate (0.0-1.0) |
| `error.codes` | []int | - | HTTP status codes to return |
| `timeout_rate` | float | - | Timeout simulation rate (0.0-1.0) |

### bidding

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `price.min` | float | - | Minimum bid price (CPM) |
| `price.max` | float | - | Maximum bid price (CPM) |
| `currency` | string | USD | Currency |
| `seat` | string | faker-dsp-seat-1 | Seat name |
| `advertiser_domains` | []string | ["example-advertiser.com"] | Advertiser domains |
| `adm_template` | string | (HTML template) | Ad markup template |

### custom (extended parameters)

| Parameter | Type | Description |
|-----------|------|-------------|
| `bid_rate_by_imp_type.banner` | float | Banner imp-specific bid rate |
| `bid_rate_by_imp_type.video` | float | Video imp-specific bid rate |
| `domain_overrides` | []object | Behavior overrides for specific domains/bundles |
| `creative_sizes` | []object | Creative size variations |
| `deal_support.enabled` | bool | Enable PMP deal support |
| `deal_support.deals` | []object | Deal definitions (deal_id, bidfloor, seat) |
| `no_bid_reason.enabled` | bool | Enable No-Bid Reason (nbr) response |
| `no_bid_reason.codes` | []int | NBR codes to return |

#### domain_overrides example

```yaml
domain_overrides:
  - domain: "blocked-publisher.com"
    no_bid: true                    # Always no-bid
  - domain: "premium-publisher.com"
    bid_rate: 0.95                  # High bid rate
    price_min: 5.00                 # High price range
    price_max: 25.00
  - bundle: "com.example.gameapp"
    bid_rate: 0.6
    price_min: 1.00
    price_max: 8.00
```

## Processing Flow

```
1. JSON decode (failure → 400)
2. Check imp array (empty → 400)
3. Error simulation check (triggered → 500/503 etc.)
4. Timeout simulation check (triggered → tmax+100ms delay)
5. Latency simulation (min-max ms)
6. Bid decision (no-bid → 204)
7. Generate Bid per imp (price below bidfloor → skip that imp)
8. All imps no-bid → 204
9. Return JSON response
```

## Development

```bash
# Test
make test

# Lint (go vet)
make lint

# Build
make build

# Clean
make clean
```

## Tech Stack

- Go 1.22+
- net/http (standard library)
- slog (standard library)
- gopkg.in/yaml.v3
