# vnstock-server: Leverage API Key for Better Performance

**Date**: 2026-03-26  
**Objective**: Integrate VNSTOCK_API_KEY into FastAPI wrapper for authenticated API access, higher rate limits, and improved data reliability

---

## Architecture

```
Current Flow:
  FastAPI Endpoint → Vnstock() (no auth) → VCI/KBS Free API

Improved Flow:
  FastAPI Endpoint → Vnstock(api_key=env) → VCI/KBS Community API
                  ↓
            Auth middleware (token validation)
            Rate limit context
```

---

## Tasks

### Phase 1: Environment & Configuration

- [ ] Create `.env.example` documenting VNSTOCK_API_KEY
- [ ] Create `config.py` to safely load and validate API key
- [ ] Add API key validation on app startup

### Phase 2: Authentication Integration

- [ ] Modify Vnstock initialization to use API key
- [ ] Create authentication middleware for FastAPI
- [ ] Add API key forwarding to all endpoints

### Phase 3: Endpoint Enhancements

- [ ] Update `/api/v1/stocks/listing` with authenticated source
- [ ] Update `/api/v1/stocks/quote` with authenticated source
- [ ] Update `/api/v1/stocks/historical` with authenticated source
- [ ] Add response metadata (rate limit info, data freshness)

### Phase 4: Error Handling & Rate Limiting

- [ ] Add rate limit awareness
- [ ] Implement exponential backoff for retries
- [ ] Add request tracking for monitoring

### Phase 5: Testing

- [ ] Test endpoints with API key
- [ ] Validate data quality improvement
- [ ] Verify rate limit headers in responses

---

## Implementation Details

### config.py

```python
import os
from typing import Optional

class VnstockConfig:
    api_key: Optional[str] = os.getenv('VNSTOCK_API_KEY')

    @staticmethod
    def validate():
        if not VnstockConfig.api_key:
            raise ValueError("VNSTOCK_API_KEY not set in environment")
        return VnstockConfig.api_key
```

### Updated main.py Pattern

```python
from vnstock import Vnstock
from config import VnstockConfig

# Initialize with API key
api_key = VnstockConfig.validate()

def get_vnstock_instance():
    return Vnstock(api_key=api_key)

@app.get("/api/v1/stocks/quote")
def get_quote(symbol: str):
    # Use authenticated instance
    df = get_vnstock_instance().stock(symbol=symbol, source='VCI').quote.history(...)
```

---

## Deliverables

1. `src/config.py` - Configuration management
2. `src/auth.py` - Authentication middleware
3. `src/main.py` - Updated endpoints with API key integration
4. `.env.example` - Environment template
5. Updated error handling & logging

---

## References

- vnstock AGENTS.md: API key authentication via environment
- vnstock source: `vnai/beam/auth.py` - Shows how library detects VNSTOCK_API_KEY
- Current endpoints: src/main.py lines 21-103
