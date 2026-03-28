# Specification: System Configuration Protocol

**Author**: Product Owner / Business Analyst  
**Intended Audience**: Dev / DevOps Engineering  
**Status**: Enforced for All Deployments  
**Keywords**: Environment Variables, Secret Management, Feature Flags, Provider Chains

---

## 1. Variable Categorization Strategy

**Objective**: To ensure categorized, secure, and self-documenting configuration across all micro-services.

---

## 2. Core Configuration Clusters (Required)

### 2.1 The "Data Heartbeat" (Google Sheets)

This cluster connects the application to the central Ledger.

- **Client Identity**: Unique OAuth2 identifiers for API interactions.
- **Client Secret**: Private credential for token exchange.
- **Refresh Token**: Persistent grant for continuous read/write access.
- **Ledger ID**: Canonical Sheet ID from the browser URL.

### 2.2 The "Intelligence Engine" (AI Provider)

These values define the brain of the Wealth Manager.

- **Model Key**: Primary access token for GPT-4o/Claude/Gemini.
- **Fallback Logic**: The app automatically cycles from Primary to Secondary if the main provider context overflows or fails.

### 2.3 The "Cache Grid" (Redis)

Mandatory for meeting the p95 < 300ms latency standard.

- **Rest Connection String**: URL to the serverless Redis instance.
- **Bearer Token**: Authorization string for REST-based caching.

---

## 3. Advanced Feature Control (Feature Flags)

**PO Logic**: Use flags to toggle high-compute features in non-production environments.

- **Advanced Search**: Enables/disables Tavily/Exa market searches.
- **AI Analytics**: Enables/disables the "Financial Health" GPT-4o analysis.
- **Debug Mode**: Toggles high-verbosity logging for troubleshooting.

---

## 4. Normalization Rules (Defaults)

The following configuration must be set in the system defaults:

- **Base Currency**: Default = `VND`.
- **Timezone**: Default = `Asia/Ho_Chi_Minh` (GMT+7).
- **TTL Window**: Cache resets at 5:00 AM local time daily for historical market data.

---

## 5. Security Architecture (Secrets Mastery)

**BA Directive**: NO production secrets or API keys are to be stored in the repository.

1.  **Local Dev**: Use `.env.local` files (Git Ignored).
2.  **Product/Staging**: Use a centralized Secret Manager (e.g., Vercel Environment Variables or AWS Secrets Manager).
3.  **Rotation**: OAuth Refresh Tokens should be re-verified or rotated every 6 months to ensure system longevity.
