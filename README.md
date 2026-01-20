# quant-read-api

High-performance **market data read API** built in Go, backed by ClickHouse.

This service provides **Index, Futures, and Options** market data with:
- raw (tick / second) data
- resampled OHLC data
- consistent response metadata
- session-aware offsets (IST market hours)

---

## 🚀 Features

- Ultra-fast ClickHouse queries
- Columnar OHLC responses (cache + chart friendly)
- Supports raw + resampled data
- Offset-based resampling (e.g. 1m candles starting at 09:15:30)
- Consistent `meta` object across all APIs

---

## 📦 API Versioning

All endpoints are **versioned** under:

/api/v1

This version is **frozen** once released (no breaking changes).

---

## 📡 API Endpoints (v1)

### 1️⃣ Index Data

**Endpoint**

GET /api/v1/index/data

**Query Parameters**

| Name       | Required | Description                         | Example |
|------------|----------|-------------------------------------|---------|
| underlying | ✅       | Index symbol                        | NIFTY   |
| from       | ✅       | Start datetime (IST)                | 2025-11-03T09:15:00 |
| to         | ✅       | End datetime (IST)                  | 2025-11-03T15:30:00 |
| tf         | ❌       | Resample timeframe (1s,5s,1m)       | 1m      |
| offset     | ❌       | Offset seconds                      | 30      |

**Raw (seconds)**

 ```curl -s "http://localhost:8081/api/v1/index/data?underlying=NIFTY&from=2025-11-03T09:15:00&to=2025-11-03T09:20:00"```

**Resampled (1m, offset 30s)**

 ```curl -s "http://localhost:8081/api/v1/index/data?underlying=NIFTY&from=2025-11-03T09:15:00&to=2025-11-03T15:30:00&tf=1m&offset=30"```

### 2️⃣ Futures Data

**Endpoint**
GET /api/v1/futures/data

**Query Parameters**

| Name        | Required | Description                     | Example             |
|-------------|----------|---------------------------------|---------------------|
| underlying  | ✅       | Symbol                          | NIFTY               |
| series      | ✅       | Contract series (numeric)       | 1                   |
| from        | ✅       | Start datetime (IST)            | 2025-11-03T09:15:00 |
| to          | ✅       | End datetime (IST)              | 2025-11-03T15:30:00 |
| tf          | ❌       | Resample timeframe (1s,5s,1m)   | 1m                  |
| offset      | ❌       | Offset seconds                  | 30                  |

**Raw**

 ```curl -s "http://localhost:8081/api/v1/futures/data?underlying=NIFTY&series=1&from=2025-11-03T09:15:00&to=2025-11-03T09:20:00"```

**Resampled**

 ```curl -s "http://localhost:8081/api/v1/futures/data?underlying=NIFTY&series=1&from=2025-11-03T09:15:00&to=2025-11-03T15:30:00&tf=1m&offset=30"```

### 3️⃣ Options Contract Data

**Endpoint**
GET /api/v1/options/contract

Query Parameters
| Name        | Required | Description           | Example             |
|-------------|----------|-----------------------|---------------------|
| underlying  | ✅       | Symbol                | NIFTY               |
| expiry      | ✅       | Expiry date           | 2025-11-18          |
| strike      | ✅       | Strike price          | 25000               |
| option_type | ✅       | CE / PE               | CE                  |
| from        | ✅       | Start datetime (IST)  | 2025-11-03T09:15:00 |
| to          | ✅       | End datetime (IST)    | 2025-11-03T15:30:00 |
| tf          | ❌       | Resample timeframe    | 1m                  |
| offset      | ❌       | Offset seconds        | 30                  |

**Raw**

 ```curl -s "http://localhost:8081/api/v1/options/contract?underlying=NIFTY&expiry=2025-11-18&strike=25000&option_type=CE&from=2025-11-03T09:15:00&to=2025-11-03T09:40:00"```

**Resampled**

 ```curl -s "http://localhost:8081/api/v1/options/contract?underlying=NIFTY&expiry=2025-11-18&strike=25000&option_type=CE&from=2025-11-03T09:15:00&to=2025-11-03T15:30:00&tf=1m&offset=30"```

## 📦 Response Format

All APIs return:

 {
  "data": { ... },
  "meta": {
    "underlying": "NIFTY",
    "from": "2025-11-03T09:15:00+05:30",
    "to": "2025-11-03T15:30:00+05:30",
    "tf": "1m",
    "offset": 30,
    "first_ts": "2025-11-03T09:16:30+05:30",
    "last_ts": "2025-11-03T15:28:30+05:30"
  }
}

Metadata is returned once per response (never per row).

## 🧱 Project Structure

components/   → DB query logic
controllers/ → HTTP handlers
models/      → Response & data models
routes/      → Router setup
services/    → ClickHouse, compression

## 🏷 Versioning
	•	v1 is stable
	•	New versions will be released as /api/v2
	•	No breaking changes inside a version

## 🧠 Philosophy
	•	Columnar data > row-based for analytics
	•	Metadata belongs to response, not rows
	•	Offset-based candles are first-class
	•	APIs should be deterministic & reproducible
## 🧑‍💻 Author

Prakul Jaiswal
