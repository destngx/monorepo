# Specification: Environment & Infrastructure Requirements

**Author**: Product Owner / System Architect  
**Intended Audience**: DevOps / Initial Setup Engineer  
**Status**: Mandatory Prerequisites  
**Keywords**: Node.js, Python, Google Cloud, Upstash, OpenAI, VNStock

---

## 1. Core Runtime Runtimes

**Objective**: To ensure the execution environment supports both the Next.js frontend and the auxiliary Python data services.

- **Primary Runtime (JS/TS)**: Required for the main application and shared libraries.
- **Secondary Runtime (Python)**: Required for the specialized `vnstock-server` (Data Crawler).
- **Package Management**: A workspace-aware runner (e.g., Bun) is required to manage dependencies across the monorepo.

---

## 2. Persistent Storage & Database Requirements

The application follows a "Serverless-First" logic, relying on managed external services:

### 2.1 Google Sheets (Primary Database)

- **Service**: Google Sheets API v4.
- **Requirement**: A Google Cloud Project with the Sheets API enabled.
- **Access**: OAuth2 credentials (Client ID, Client Secret) and a long-lived Refresh Token.
- **Entity**: A target Spreadsheet ID to serve as the Transaction Ledger.

### 2.2 Redis (Global Cache)

- **Service**: Upstash Redis (REST-compatible).
- **Requirement**: Endpoint URL and Authorization Token.
- **Usage**: Mandatory for high-performance aggregates and session-level net worth tracking.

---

## 3. Intelligence & AI Service Requirements

The "Intelligence Center" requires active API access to one or more of the following providers:

- **OpenAI**: Primary for GPT-4o synthesis.
- **Anthropic**: Fallback for complex logical reasoning (Claude).
- **Google Generative AI**: Alternative for high-speed summarization (Gemini).
- **GitHub Copilot API**: Optional token-based fallback.

---

## 4. Connectivity & Search Integration

To enable real-time "Market Pulse" and news deep-dives:

- **Web Search API**: Tavily (Primary) or Exa (Secondary).
- **Data Crawling**: Local or remote access to the `vnstock-server` instance.

---

## 5. Security & Access Control

- **Environment Isolation**: Mandatory use of localized environment files (e.g., `.env.local`) for all sensitive keys.
- **OAuth Project Status**: For permanent token validity, the Google Cloud Project must be set to "Production" state (not "Testing").
