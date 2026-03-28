# Quick Reference: Web Search Roundrobin Setup

## ⚡ 5-Minute Setup

```bash
# 1. Copy environment template
cp .env.example .env.local

# 2. Add your API keys to .env.local
# Get Tavily key: https://tavily.com/api
# Use Exa key:
# DuckDuckGo: No key needed

# 3. Install dependencies
npm install

# 4. Start the app
npm run serve wealth-management

# 5. Test in chat: "What's the latest Bitcoin price?"
```

## 🔍 Provider Fallback Chain

```
Query
  └─ Tavily (Advanced)
       ├─ Success? ✅ Done
       ├─ Failed? ⚠️ Try Exa
       └─ No results? ⏭️ Try Exa

  └─ Exa (Neural Search)
       ├─ Success? ✅ Done
       ├─ Failed? ⚠️ Try DuckDuckGo
       └─ No results? ⏭️ Try DuckDuckGo

  └─ DuckDuckGo (Fallback)
       ├─ Success? ✅ Done
       └─ Failed? ❌ Error
```

## 📝 Configuration

**Required in `.env.local`:**

```env
TAVILY_API_KEY=tvly_your_key_here
EXA_API_KEY=
```

**Optional:** DuckDuckGo works without configuration (free fallback)

## 📊 Console Output Examples

**Success:**

```
[SearchService] Attempting tavily...
[SearchService] ✓ Success with tavily (5 results)
```

**Fallback:**

```
[SearchService] Attempting tavily...
[SearchService] ✗ tavily failed: Rate limit exceeded, trying next provider...
[SearchService] Attempting exa...
[SearchService] ✓ Success with exa (4 results)
```

**Final Fallback:**

```
[SearchService] Attempting tavily...
[SearchService] ✗ tavily failed: API key invalid, trying next provider...
[SearchService] Attempting exa...
[SearchService] ✗ exa failed: No results, trying next provider...
[SearchService] Attempting duckduckgo...
[SearchService] ✓ Success with duckduckgo (5 results)
```

## 🔧 How It's Used

The search service is **automatically integrated** with:

- ✅ Chat API (`/api/chat`)
- ✅ Financial tools (webSearch tool)
- ✅ Price service (as fallback for crypto/fund prices)

No additional changes needed—it just works!

## 🚨 Troubleshooting

| Issue                            | Check                                 |
| -------------------------------- | ------------------------------------- |
| "All providers failed"           | `.env.local` has correct API keys     |
| "tavily: API key not configured" | `TAVILY_API_KEY` is set and not empty |
| "exa: API key not configured"    | `EXA_API_KEY` is set and not empty    |
| Still using DuckDuckGo           | API keys might be rate-limited        |

## 📚 Files Modified

1. ✅ `libs/wealth-management/src/services/services/search-service.ts` — Core implementation (233 lines)
2. ✅ `package.json` — Added `exa-js` dependency
3. ✅ `.env.example` — Configuration template (80+ lines)
4. 📘 `IMPLEMENTATION_SUMMARY.md` — Full documentation

## 💡 Key Features

- ✅ Automatic fallback (no manual intervention)
- ✅ Detailed logging for debugging
- ✅ Health check endpoint available
- ✅ Zero breaking changes (backward compatible)
- ✅ Works offline with DuckDuckGo fallback
- ✅ Production-ready error handling

## 📞 Support

**Need help?**

1. Check console logs for `[SearchService]` messages
2. Review `.env.example` for configuration
3. See `IMPLEMENTATION_SUMMARY.md` for detailed docs
4. Test each provider independently

---

**Deployment Status:** ✅ Ready to test | ⏭️ Needs npm install | 🔐 Needs API keys in .env.local
