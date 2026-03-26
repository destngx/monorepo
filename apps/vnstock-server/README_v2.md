# 🌟 vnstock-server v0.2.0 - API Key Integration

**Status**: ✅ Production Ready | **Version**: 0.2.0 | **Date**: 2026-03-26

A FastAPI wrapper for vnstock with authenticated access to Vietnamese stock market data.

## 🎯 What's New in v0.2.0

- ✅ **API Key Authentication** - Community tier (60 req/min)
- ✅ **Configuration Management** - Environment-based settings
- ✅ **New Endpoint** - `/api/v1/health/auth` for auth status
- ✅ **Enhanced Responses** - All responses include auth status
- ✅ **Production Ready** - Error handling, logging, validation

## 🚀 Quick Start

```bash
cd /Users/ez2/projects/personal/monorepo/apps/vnstock-server
source .venv/bin/activate
uvicorn src.main:app --reload
```

Visit: http://localhost:8000/docs

## 📚 Documentation

Choose based on your needs:

| Document              | Purpose                          | Read Time |
| --------------------- | -------------------------------- | --------- |
| **QUICKSTART.md**     | Get running in 30 seconds        | 2 min     |
| **API_KEY_SETUP.md**  | Complete setup & usage guide     | 10 min    |
| **IMPLEMENTATION.md** | Technical implementation details | 5 min     |
| **CHANGES.md**        | What's new & migration guide     | 3 min     |

## 📊 Key Metrics

| Metric            | Before    | After       |
| ----------------- | --------- | ----------- |
| **Rate Limit**    | Standard  | 60 req/min  |
| **Tier**          | Free      | Community   |
| **Auth Tracking** | ❌ No     | ✅ Yes      |
| **Config**        | Hardcoded | Environment |

## 🔗 API Endpoints

```
GET /                              Welcome page
GET /health                        Server health + auth status
GET /api/v1/health/auth    ⭐ NEW  Check authentication status
GET /api/v1/stocks/listing  🔐     List all Vietnamese stocks
GET /api/v1/stocks/quote    🔐     Get latest price for symbol
GET /api/v1/stocks/historical 🔐   Get historical OHLCV data
GET /docs                          Interactive Swagger UI
```

## ✅ Testing

All features verified:

- ✅ Python syntax validated
- ✅ Module imports successful
- ✅ API key loads from environment
- ✅ Authentication succeeds (Community tier)
- ✅ All endpoints accessible
- ✅ 100% backward compatible

## 🔐 Your API Key

- **Location**: `.env.local`
- **Status**: ✅ Loaded & Authenticated
- **Tier**: 🌟 Community (60 requests/minute)
- **Format**: `vnstock_33aa4657...` (masked for security)

## 📁 Files

```
src/config.py           Configuration & validation (NEW)
src/main.py             FastAPI app with API key integration (UPDATED)
.env.example            Environment template (NEW)
QUICKSTART.md           Quick start guide (NEW)
API_KEY_SETUP.md        Comprehensive guide (NEW)
IMPLEMENTATION.md       Technical details (NEW)
CHANGES.md              Version history (NEW)
```

## 🎯 Features

1. **Secure API Key Management**
   - Loads from VNSTOCK_API_KEY environment variable
   - Validated on startup
   - Clear error messages

2. **Authentication Integration**
   - Uses vnstock's `change_api_key()` function
   - Community tier access
   - Priority access during trading hours

3. **Enhanced API Responses**
   - All responses include `"authenticated"` field
   - Backward compatible
   - Client-friendly metadata

4. **Production Ready**
   - Comprehensive logging
   - Error handling
   - Non-blocking initialization

## 🚨 Troubleshooting

**API Key Not Loading?**

```bash
cat .env.local | grep VNSTOCK_API_KEY
export $(cat .env.local | xargs)
```

**Want Debug Output?**

```bash
DEBUG=true uvicorn src.main:app --reload
```

**Check Authentication:**

```bash
curl http://localhost:8000/api/v1/health/auth
```

## 📖 Usage Examples

### Get Stock Quote

```bash
curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
```

Response:

```json
{
  "success": true,
  "data": {
    "symbol": "VCB",
    "price": 98.5,
    "time": "2026-03-26 15:30:00"
  },
  "authenticated": true
}
```

### Check Authentication

```bash
curl http://localhost:8000/api/v1/health/auth
```

Response:

```json
{
  "api_key_configured": true,
  "tier": "Community (with API key)",
  "benefits": {
    "with_key": ["Higher rate limits", "Priority access", "Better reliability"]
  }
}
```

## 🔄 Migration Guide

**Existing clients**: No changes required! The new `"authenticated"` field is optional.

**For new deployments**:

1. Ensure `VNSTOCK_API_KEY` is in environment
2. Run `uvicorn src.main:app --reload`
3. Check `/api/v1/health/auth` to verify auth
4. All endpoints automatically use Community tier

## 🎓 Learn More

- vnstock Documentation: https://vnstocks.com/
- GitHub Repository: https://github.com/thinh-vu/vnstock
- Get API Key: https://vnstocks.com/onboard-member/

## 📞 Support

- **Quick Questions**: See QUICKSTART.md
- **Setup Help**: See API_KEY_SETUP.md
- **Technical Info**: See IMPLEMENTATION.md
- **Version Info**: See CHANGES.md

## 📝 License

Same as parent project

---

**Version**: 0.2.0 | **Updated**: 2026-03-26 | **Status**: ✅ Production Ready

Start using it now: `uvicorn src.main:app --reload`
