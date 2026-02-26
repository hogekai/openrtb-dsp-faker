# openrtb-dsp-faker

OpenRTB 2.6準拠のDSPモックサーバー。SSP/Ad Exchange開発者がDSP接続のテスト・デバッグに使うためのツール。

実際のDSPの挙動（レイテンシ、エラー率、入札パターン）を設定ファイルベースで再現し、リアルなテスト環境を提供する。

## クイックスタート

```bash
# ビルド
make build

# デフォルト設定で起動
./bin/faker

# カスタム設定で起動
./bin/faker -config ./configs/custom.yaml
```

## エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| POST | `/openrtb2/bid` | Bid Request受信 → Bid Response返却 |
| GET | `/health` | ヘルスチェック (`{"status":"ok"}`) |

## テスト用リクエスト

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

## 設定ファイル

`configs/default.yaml` を参照。主な設定項目:

### server

| パラメーター | 型 | デフォルト | 説明 |
|---|---|---|---|
| `port` | int | - | 待受ポート |
| `read_timeout_ms` | int | 5000 | HTTP read timeout (ms) |
| `write_timeout_ms` | int | 5000 | HTTP write timeout (ms) |

### behavior

| パラメーター | 型 | デフォルト | 説明 |
|---|---|---|---|
| `bid_rate` | float | - | 入札確率 (0.0〜1.0) |
| `latency.min_ms` | int | - | 最小応答遅延 (ms) |
| `latency.max_ms` | int | - | 最大応答遅延 (ms) |
| `error.rate` | float | - | HTTPエラー返却率 (0.0〜1.0) |
| `error.codes` | []int | - | 返却するHTTPステータスコード |
| `timeout_rate` | float | - | タイムアウトシミュレーション率 (0.0〜1.0) |

### bidding

| パラメーター | 型 | デフォルト | 説明 |
|---|---|---|---|
| `price.min` | float | - | 最低入札価格 (CPM) |
| `price.max` | float | - | 最高入札価格 (CPM) |
| `currency` | string | USD | 通貨 |
| `seat` | string | faker-dsp-seat-1 | Seat名 |
| `advertiser_domains` | []string | ["example-advertiser.com"] | 広告主ドメイン |
| `adm_template` | string | (HTMLテンプレート) | Ad markup テンプレート |

### custom（拡張パラメーター）

| パラメーター | 型 | 説明 |
|---|---|---|
| `bid_rate_by_imp_type.banner` | float | Banner imp 固有の入札率 |
| `bid_rate_by_imp_type.video` | float | Video imp 固有の入札率 |
| `domain_overrides` | []object | 特定domain/bundleへの挙動オーバーライド |
| `creative_sizes` | []object | Creative サイズバリエーション |
| `deal_support.enabled` | bool | PMP deal対応の有効化 |
| `deal_support.deals` | []object | Deal定義 (deal_id, bidfloor, seat) |
| `no_bid_reason.enabled` | bool | No-Bid Reason (nbr) 返却の有効化 |
| `no_bid_reason.codes` | []int | 返却するnbrコード |

#### domain_overrides の設定例

```yaml
domain_overrides:
  - domain: "blocked-publisher.com"
    no_bid: true                    # 常にno-bid
  - domain: "premium-publisher.com"
    bid_rate: 0.95                  # 高い入札率
    price_min: 5.00                 # 高い価格帯
    price_max: 25.00
  - bundle: "com.example.gameapp"
    bid_rate: 0.6
    price_min: 1.00
    price_max: 8.00
```

## 処理フロー

```
1. JSONデコード (失敗 → 400)
2. imp空チェック (空 → 400)
3. エラーシミュレーション判定 (該当 → 500/503等)
4. タイムアウトシミュレーション判定 (該当 → tmax+100ms遅延)
5. レイテンシシミュレーション (min〜max ms)
6. 入札判定 (no-bid → 204)
7. imp毎のBid生成 (bidfloor以下 → そのimpはno-bid)
8. 全impがno-bid → 204
9. JSON応答返却
```

## 開発

```bash
# テスト
make test

# lint (go vet)
make lint

# ビルド
make build

# クリーン
make clean
```

## Tech Stack

- Go 1.22+
- net/http (標準ライブラリ)
- slog (標準ライブラリ)
- gopkg.in/yaml.v3
