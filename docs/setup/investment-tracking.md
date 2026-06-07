# Investment & Asset Tracking Setup

Track investment portfolio value (stocks, ETFs, crypto) and asset appreciation/depreciation over time.

**Data sources:**
- **Finnhub** — stocks, ETFs (free, 60 req/min)
- **CoinGecko** — crypto (free, 30 req/min, no API key required)
- **Manual valuation** — real estate, vehicles, collectibles

---

## Part 1: Finnhub (Stocks & ETFs)

### Step 1: Get API key

1. Go to [finnhub.io](https://finnhub.io) → **Get free API key**
2. Sign up with email
3. Copy API key from dashboard

### Step 2: Store credentials

Add to `.env.local` and Railway:
```
FINNHUB_API_KEY=your_api_key_here
```

### Step 3: Key endpoints

**Quote (current price):**
```
GET https://finnhub.io/api/v1/quote?symbol=AAPL&token=YOUR_KEY

Response:
{
  "c": 189.30,   // current price
  "d": -0.45,    // change
  "dp": -0.24,   // % change
  "h": 190.12,   // high
  "l": 188.50,   // low
  "pc": 189.75   // previous close
}
```

**Symbol search:**
```
GET https://finnhub.io/api/v1/search?q=apple&token=YOUR_KEY
```

**Supported markets:** US stocks (NYSE, NASDAQ), some international exchanges
For Vietnamese stocks (VNM, etc.) — Finnhub has limited coverage; use manual valuation instead.

### Step 4: Rate limits

Free tier: 60 API calls/minute, 30 calls/second.
With daily price fetch for 20 holdings = 20 calls/day — well within limits.
Do NOT fetch real-time on every page load. Fetch once daily via background job and cache in `investment_prices` table.

---

## Part 2: CoinGecko (Crypto)

### No API key required for basic use

**Current price:**
```
GET https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum&vs_currencies=usd

Response:
{
  "bitcoin": { "usd": 67420.00 },
  "ethereum": { "usd": 3521.00 }
}
```

**Find coin ID by symbol:**
```
GET https://api.coingecko.com/api/v3/coins/list
```
Returns full list — search for your coin's `id` field (e.g. `bitcoin`, `ethereum`, `solana`).

**Historical price (for charts):**
```
GET https://api.coingecko.com/api/v3/coins/bitcoin/market_chart?vs_currency=usd&days=30
```

### Rate limits

Free tier: 30 calls/min, ~10k calls/month.
Add API key (free registration) to raise limit:
```
COINGECKO_API_KEY=CG-your_key_here
```
Pass as header: `x-cg-demo-api-key: CG-your_key_here`

---

## Part 3: Database Schema

Create `migrations/004_investments.sql`:

```sql
-- Investment holdings
CREATE TABLE investments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('stock', 'etf', 'crypto', 'fund', 'other')),
    quantity DECIMAL(18, 8) NOT NULL,
    avg_buy_price DECIMAL(18, 4) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Daily price snapshots (fetched by background job)
CREATE TABLE investment_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    price DECIMAL(18, 4) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    source TEXT NOT NULL CHECK (source IN ('finnhub', 'coingecko', 'manual')),
    fetched_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_investment_prices_symbol_time ON investment_prices(symbol, fetched_at DESC);

-- Asset valuation history (for appreciation/depreciation tracking)
ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS purchase_price DECIMAL(18, 2),
    ADD COLUMN IF NOT EXISTS purchase_date DATE,
    ADD COLUMN IF NOT EXISTS depreciation_rate DECIMAL(5, 4), -- annual rate e.g. 0.15 = 15%
    ADD COLUMN IF NOT EXISTS depreciation_method TEXT CHECK (depreciation_method IN ('straight_line', 'reducing_balance', 'manual'));

CREATE TABLE asset_valuations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    value DECIMAL(18, 2) NOT NULL,
    note TEXT,
    valued_at DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_asset_valuations_asset_time ON asset_valuations(asset_id, valued_at DESC);
```

**RLS policies** (add to `002_rls.sql` or new migration):
```sql
ALTER TABLE investments ENABLE ROW LEVEL SECURITY;
CREATE POLICY investments_user ON investments USING (user_id = auth.uid());

ALTER TABLE asset_valuations ENABLE ROW LEVEL SECURITY;
CREATE POLICY asset_valuations_user ON asset_valuations
    USING (asset_id IN (SELECT id FROM assets WHERE user_id = auth.uid()));
```

---

## Part 4: API Endpoints

Add to `backend/cmd/server/main.go`:

```
GET    /api/v1/investments              — list holdings with current value
POST   /api/v1/investments              — add holding
PATCH  /api/v1/investments/{id}         — update quantity/avg price
DELETE /api/v1/investments/{id}         — remove holding

GET    /api/v1/investments/summary      — total portfolio value, P&L, allocation breakdown
POST   /api/v1/investments/sync-prices  — trigger manual price fetch

GET    /api/v1/assets/{id}/valuations   — valuation history for an asset
POST   /api/v1/assets/{id}/valuations   — add manual valuation
```

### Portfolio summary response shape

```json
{
  "total_value": 45230.00,
  "total_cost": 38500.00,
  "total_pnl": 6730.00,
  "total_pnl_pct": 17.48,
  "currency": "USD",
  "holdings": [
    {
      "id": "uuid",
      "symbol": "AAPL",
      "name": "Apple Inc.",
      "type": "stock",
      "quantity": 10,
      "avg_buy_price": 150.00,
      "current_price": 189.30,
      "current_value": 1893.00,
      "pnl": 393.00,
      "pnl_pct": 26.20
    }
  ]
}
```

---

## Part 5: Price Fetch Background Job

Add to `backend/cmd/server/main.go` after router setup:

```go
go func() {
    // Fetch prices once at startup, then every 24h
    fetchAndStorePrices(pool, cfg)
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        if err := fetchAndStorePrices(pool, cfg); err != nil {
            log.Printf("price fetch error: %v", err)
        }
    }
}()
```

**Fetch logic:**
1. Query distinct symbols from `investments` table grouped by type
2. Batch stock/ETF symbols → Finnhub quote API (one call per symbol)
3. Batch crypto coin IDs → CoinGecko simple/price API (all in one call)
4. Insert rows into `investment_prices`
5. Keep last 365 days, delete older rows

---

## Part 6: Asset Depreciation Calculation

No external API needed — calculate server-side.

**Straight-line depreciation:**
```
current_value = purchase_price × (1 - depreciation_rate × years_elapsed)
```

**Reducing balance (common for vehicles):**
```
current_value = purchase_price × (1 - depreciation_rate) ^ years_elapsed
```

Example — car bought for $20,000 with 15% annual reducing balance depreciation:
- Year 1: $17,000
- Year 2: $14,450
- Year 3: $12,282

If `depreciation_method = 'manual'`, skip formula — use latest row from `asset_valuations` table.

Backend calculates current estimated value on the fly when returning asset data; no need to store calculated values.

---

## Part 7: Net Worth Integration

Update `GET /api/v1/dashboard/summary` to include:

```json
{
  "net_worth": {
    "investments": 45230.00,
    "assets": 285000.00,
    "liabilities": 120000.00,
    "total": 210230.00
  }
}
```

Liabilities = sum of negative-value transactions tagged as `liability` category, or a separate liabilities table if needed later.

---

## Security Checklist

- [ ] `FINNHUB_API_KEY` only in backend env (Railway), never Vercel
- [ ] `COINGECKO_API_KEY` only in backend env
- [ ] Price fetch job runs server-side only — never expose raw API keys to frontend
- [ ] Investment endpoints scoped to authenticated user via RLS
- [ ] Asset valuation endpoint checks asset belongs to authenticated user before insert
