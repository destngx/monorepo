# ⚡ Quick Start - API Key Integration

Your vnstock-server is now configured with API key authentication for higher rate limits and better reliability.

## 🚀 Start Server (30 seconds)

```bash
cd /Users/ez2/projects/personal/monorepo/apps/vnstock-server
source .venv/bin/activate
uvicorn src.main:app --reload
```

Expected output:

```
INFO:main:✓ API key loaded and authenticated successfully
Bạn đang sử dụng Phiên bản cộng đồng (60 requests/phút)
```

## 🔗 Available Endpoints

| Endpoint                                   | Purpose         | Example                                                            |
| ------------------------------------------ | --------------- | ------------------------------------------------------------------ |
| `GET /`                                    | Welcome         | `curl http://localhost:8000/`                                      |
| `GET /health`                              | Health check    | `curl http://localhost:8000/health`                                |
| `GET /api/v1/health/auth`                  | **Auth status** | `curl http://localhost:8000/api/v1/health/auth`                    |
| `GET /api/v1/stocks/listing`               | All stocks      | `curl http://localhost:8000/api/v1/stocks/listing`                 |
| `GET /api/v1/stocks/quote?symbol=VCB`      | Latest price    | `curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"`      |
| `GET /api/v1/stocks/historical?symbol=VCB` | Historical data | `curl "http://localhost:8000/api/v1/stocks/historical?symbol=VCB"` |
| `GET /docs`                                | **Swagger UI**  | http://localhost:8000/docs                                         |

## ✅ Verify Authentication

```bash
curl http://localhost:8000/api/v1/health/auth
```

Response:

```json
{
  "api_key_configured": true,
  "tier": "Community (with API key)",
  "benefits": {
    "with_key": ["Higher rate limits", "Priority access", "Better reliability"],
    "without_key": ["Limited to free tier", "Standard rate limits"]
  }
}
```

## 📊 What Changed?

| Before                          | After                            |
| ------------------------------- | -------------------------------- |
| Free tier (standard rate limit) | Community tier (60 req/min)      |
| No auth tracking                | Auth status visible in responses |
| Hardcoded VCI source            | Environment-based configuration  |
| Generic errors                  | Clear error messages             |

## 📁 Files Created/Modified

- ✅ `src/config.py` - Config management
- ✅ `src/main.py` - Updated with API key integration
- ✅ `.env.example` - Environment template
- ✅ `API_KEY_SETUP.md` - Complete guide
- ✅ `IMPLEMENTATION.md` - Technical details

## 🔐 Your API Key

- Status: ✅ Loaded from `.env.local`
- Tier: Community (60 requests/minute)
- Format: `vnstock_33aa4657f59a449495db643c729dc0d2`

## 📚 Learn More

- **Setup Guide**: `API_KEY_SETUP.md`
- **Technical Details**: `IMPLEMENTATION.md`
- **Plan**: `.sisyphus/plan.md`

## 🆘 Troubleshooting

**API key not loading?**

```bash
# Check it's in .env.local
cat .env.local | grep VNSTOCK_API_KEY

# Load it manually
export $(cat .env.local | xargs)
```

**Want to debug?**

```bash
DEBUG=true uvicorn src.main:app --reload
```

---

**Ready?** Run: `uvicorn src.main:app --reload`

Then visit: http://localhost:8000/docs
