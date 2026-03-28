# Investments Feature - One-Pager

## Overview

The Investments feature provides a comprehensive AI-powered investment management platform with global macro analysis, real-time market intelligence, and automated portfolio optimization. Users can track crypto and fund holdings across multiple platforms, analyze market trends across multiple timeframes, monitor seasonal patterns, and receive AI-generated investment strategies through an interactive think tank system. The feature integrates Vietnamese financial markets (Fmarket) for comprehensive local market coverage.

**Primary Use Cases**:

- Portfolio tracking and valuation (crypto + investment funds + Vietnamese markets)
- Real-time market intelligence and sentiment analysis
- Multi-timeframe technical analysis
- Seasonal pattern recognition for trading decisions
- Global macro geopolitical synthesis
- AI-driven investment recommendations and actionable commands
- News analysis for market impact assessment
- Vietnamese financial market integration (stocks, bonds, funds, gold, rates)

## User Flow

### Basic Investment Dashboard Flow

1. **Access**: Navigate to `/investments` page via sidebar
2. **Data Loading**: System fetches accounts, assets (crypto/funds), exchange rates, Vietnamese market data
3. **AI Price Enhancement**: Automatically fetches missing prices using AI market data
4. **Dashboard Exploration**: Switch between 8 specialized analysis tabs
5. **Think Tank Analysis**: Initiate AI-powered investment strategy generation
6. **Actionable Commands**: Receive executable investment recommendations

### Advanced Analysis Workflow

- **Market Pulse**: Real-time market sentiment and trend analysis
- **Multi-Timeframe**: Technical analysis across different chart periods
- **Seasonal Patterns**: Historical seasonal trends for assets
- **News Analysis**: Geopolitical and economic news impact assessment
- **Ticker Analysis**: Deep-dive technical analysis for specific symbols
- **Fmarket Insights**: Vietnamese market data integration (stocks, bonds, funds, gold, USD/VND rates)
- **Asset Ledgers**: Detailed holdings tables with real-time valuations
- **Vietnamese Market Drill-down**: Click any fund for detailed NAV history and issuer information

### AI Think Tank Process

1. **Data Extraction**: Analyzes user accounts, holdings, market data, and Vietnamese market context
2. **Market Intelligence**: Performs web searches and quantitative analysis
3. **Think Tank Debate**: 5 specialized AI analysts debate strategies
4. **Synthesis**: Combines analysis into coherent recommendations
5. **Action Generation**: Creates executable investment commands (ALLOCATE, HEDGE, OBSERVE)
6. **Streaming Response**: Real-time delivery of analysis and recommendations

## API Endpoints Used

### `/api/investments/assets` (GET)

**Fetches crypto and fund holdings data from Google Sheets**

- **Response**: `{ crypto: { headers: string[], holdings: any[] }, funds: { headers: string[], holdings: any[] } }`
- **Source**: Google Sheets integration via libs/wealth-management with intelligent header detection
- **Processing**: Smart column detection using keyword patterns, platform grouping, empty holding filtering
- **Caching**: Dual-layer (Redis + in-memory), portfolio data TTL
- **Features**: Flexible data structure support, robust parsing with fallback handling

### `/api/investments/prices` (POST)

**AI-powered price fetching for missing asset data**

- **Parameters**: `{ symbols: [{ symbol: string, type: 'crypto' | 'fund' }] }`
- **Response**: `{ prices: Record<string, number> }`
- **Features**: Dual pricing system (sheet price priority + AI fallback), market data API integration
- **Processing**: Identifies assets without prices, fetches real-time data, caches results
- **Caching**: Redis-based with `price_symbol:{symbol}:{type}` keys and market data TTL

### `/api/ai/investment-analysis` (POST)

**Main AI orchestration endpoint for investment strategy**

- **Parameters**: `{ accounts: any[], prices?: Record<string, number>, newsAnalysis?: Record<string, any> }`
- **Response**: Streaming NDJSON with phases: think-tank, synthesis, actions, context
- **Features**: 300-second timeout, multi-phase AI processing, 5-specialist think tank debate
- **Processing**: Data extraction → Market intelligence (web search + quantitative) → Think tank debate → Synthesis → Action commands

### `/api/fmarket` (POST)

**Vietnamese financial market data proxy with caching and header spoofing**

- **Parameters**: `{ action: string, params?: object }` with optional `?force=true` for cache bypass
- **Actions Supported**:
  - `getProductsFilterNav` - List funds (stock/bond/balanced/MMF) with filtering
  - `getIssuers` - Fund issuers list
  - `getBankInterestRates` - Bank interest rate products
  - `getProductDetails` - Fund/product details by code
  - `getNavHistory` - NAV history for specific fund with date ranges
  - `getGoldPriceHistory` - Gold prices over configurable date ranges
  - `getUsdRateHistory` - USD/VND exchange rates over time
  - `getGoldProducts` - Available gold investment products
- **Response**: JSON data from Fmarket API with structured formatting and error handling
- **Features**: Header spoofing to avoid blocking, Redis caching, configurable time ranges
- **Caching**: Key format `fmarket:{action}:{params}`, market data TTL with force refresh option

### `/api/accounts` (GET), `/api/exchange-rate` (GET)

**Supporting data for portfolio calculations**

- **Accounts**: Financial account balances and currencies for total portfolio calculation
- **Exchange Rate**: VND/USD conversion rates (default 25400 with real-time updates)
- **Integration**: Used for total portfolio valuation and crypto price conversions

## Key Components

### Frontend Components

- **InvestmentsPage**: Main dashboard with tabbed interface and AI think tank integration
- **FmarketDashboard**: Vietnamese financial market insights with interactive charts and drill-down
- **MarketPulseDashboard**: Real-time market sentiment and trend analysis
- **MultiTimeframeDashboard**: Technical analysis across multiple chart periods
- **SeasonalPatternsDashboard**: Historical seasonal trend analysis for assets
- **NewsAnalysisDashboard**: Geopolitical news impact assessment (3-topic instances)
- **TickerAnalysisDashboard**: Individual asset deep-dive technical analysis
- **Asset Tables**: Dynamic tables with masked balances, calculation tooltips, and real-time updates

### State Management

- **Asset State**: Crypto and fund holdings with platform grouping and dynamic headers
- **Price State**: Dual-source pricing (sheet + AI) with caching and refresh indicators
- **Analysis State**: AI think tank streaming responses with message persistence
- **Fmarket State**: Complex Vietnamese market data (9 data types, time ranges, drill-down)
- **Exchange Rate**: VND/USD conversion with fallback handling
- **News Analysis**: Cross-topic market sentiment data for AI context

### AI Orchestration Components

- **AIOrchestrator**: Core AI processing engine with model selection and streaming
- **Think Tank System**: 5 specialized AI analysts for multi-perspective strategy debate
- **Streaming Response Handler**: Real-time message processing with JSON parsing and error recovery
- **Action Command Parser**: Converts AI recommendations to executable commands with icons
- **Context Builder**: Aggregates user data, market intelligence, and news for AI prompts

## Data Flow

```
User Access → Data Loading (Accounts + Assets + Exchange Rates + Fmarket)
                    ↓
         Smart Column Detection (Header keywords → Column mapping)
                    ↓
         AI Price Enhancement (Missing detection → Dual pricing system)
                    ↓
         Portfolio Calculation (VND totals + Currency conversions)
                    ↓
Dashboard Rendering (8 Analysis Tabs + Vietnamese Market Integration)
                    ↓
Think Tank Initiation → Data Extraction → Market Intelligence Gathering
                    ↓
         Think Tank Debate (5 AI Analysts with Web Search + Quantitative)
                    ↓
         Synthesis Phase (Strategy Formation with Vietnamese Context)
                    ↓
         Action Commands (ALLOCATE 💰 | HEDGE 🛡️ | OBSERVE 👁️)
                    ↓
         Streaming Response → User Interface Updates + Context Preservation
```

## Edge Cases & Error Handling

### Data Loading Failures

- **Issue**: Google Sheets API unavailable or authentication expired
- **Handling**: Graceful degradation with cached data, clear error messages, retry options
- **User Experience**: "Data loading failed" notification with last successful data timestamp

### Smart Column Detection Issues

- **Issue**: Sheet columns don't match expected keyword patterns
- **Handling**: Fallback to alternative keywords, empty value handling, user notification
- **User Experience**: Automatic adaptation with "column detection may be inaccurate" warnings

### AI Price Fetching Errors

- **Issue**: Market data APIs fail or return invalid prices (e.g., price = 1.0 indicating error)
- **Handling**: Fallback to cached prices, skip unfetchable symbols, visual error indicators
- **User Experience**: Price tooltips show fetch status, calculation breakdowns with source attribution

### Streaming Analysis Timeouts

- **Issue**: Complex analysis exceeds 300-second timeout
- **Handling**: Graceful termination with partial results delivery, abort controller cleanup
- **User Experience**: "Analysis truncated" notification with available insights preserved

### Vietnamese Market API Issues

- **Issue**: Fmarket API blocking or data format changes
- **Handling**: Header spoofing, fallback data structures, alternative endpoints
- **User Experience**: "Market data temporarily unavailable" with cached data display

### Think Tank Analysis Failures

- **Issue**: AI orchestration fails at any phase (think tank, synthesis, actions)
- **Handling**: Fallback prompts and simplified analysis modes, phase isolation
- **User Experience**: Continues with available analysis phases, error boundaries prevent crashes

## Performance Considerations

### Optimization Strategies

- **Dual-Layer Caching**: Redis (persistent) + in-memory (fast) with configurable TTLs
- **Lazy Loading**: Dashboard components loaded on tab activation, Vietnamese market data on demand
- **Debounced Updates**: Price fetching and portfolio recalculation batched to reduce API calls
- **Streaming Responses**: Real-time AI delivery without blocking UI, efficient JSON parsing
- **Memoized Calculations**: Portfolio totals and asset valuations cached with dependency tracking
- **Smart Column Detection**: Keyword-based header mapping reduces processing overhead

### Resource Usage

- **Network**: Multiple API calls (sheets, prices, Fmarket, AI, exchange rates, web search)
- **Memory**: Large datasets for holdings, market analysis, and Vietnamese financial data
- **CPU**: Real-time calculations, AI streaming processing, chart rendering (Recharts)
- **Storage**: localStorage for chat history, Redis for price/fmarket caching

### Scalability Limits

- **Asset Holdings**: No hard limit, but large datasets impact table rendering performance
- **Vietnamese Market Data**: 9 parallel API calls on mount, cached for performance
- **Concurrent Users**: Isolated per-user sessions, no shared state conflicts
- **AI Analysis**: Rate-limited by AI provider quotas, 300-second streaming timeout
- **Market Data**: Cached for performance, real-time for critical price updates

## Configuration Options

### Dashboard Tabs Configuration

- **Think Tank**: Core AI analysis interface with streaming responses (always enabled)
- **News Analyze**: Geopolitical impact assessment with 3 configurable topics (Vietnam, Macro, Geopolitics)
- **Market Pulse**: Real-time sentiment analysis with timeframe selection
- **Multi-TF**: Technical analysis periods (customizable chart intervals)
- **Seasonality**: Seasonal pattern analysis with historical data depth
- **Ticker Analyze**: Individual asset deep-dive with symbol input validation
- **Fmarket**: Vietnamese market integration with time range controls (1M/6M/YTD/1Y/2Y/3Y/5Y/ALL)
- **Asset Ledgers**: Holdings tables with VND estimation and calculation transparency

### AI Analysis Settings

- **Model Selection**: GPT-4o (balanced), Claude (reasoning), Gemini (multimodal) for different analysis styles
- **Analysis Depth**: Quick (basic), Comprehensive (full think tank), Custom (specific focus)
- **Context Window**: Conversation history retention (short/medium/long term)
- **Fallback Modes**: Simplified analysis when advanced AI features fail
- **Vietnamese Context**: Include/exclude local market data in global analysis

### Data Integration Settings

- **Google Sheets**: Spreadsheet ID and authentication with OAuth refresh handling
- **Exchange Rate**: Primary API source with fallback to default rate (25400)
- **Market Data APIs**: Price fetching providers with priority ordering
- **Fmarket Integration**: API endpoint configuration and header spoofing settings
- **Caching TTL**: Configurable cache durations for different data types

## Testing Considerations

### Unit Tests Needed

- Asset data parsing and smart column detection logic
- Price calculation formulas and VND conversion algorithms
- AI streaming response parsing and error recovery
- Vietnamese market data transformation and chart rendering
- Dual pricing system (sheet + AI) resolution logic
- Fmarket hook state management and time range calculations

### Integration Tests Needed

- End-to-end asset data flow from Google Sheets to UI display
- Vietnamese market data pipeline from Fmarket API to dashboard
- AI analysis workflow with mock streaming responses
- Price fetching integration with external APIs and caching
- Multi-dashboard tab switching with state preservation
- Cross-feature data sharing (accounts, exchange rates, news analysis)

### E2E Test Scenarios

- Complete investment analysis workflow with AI think tank and streaming responses
- Vietnamese market drill-down (fund selection → NAV history → issuer details)
- Asset price updates and portfolio recalculation with dual pricing
- Error recovery when APIs are unavailable (sheets, Fmarket, AI)
- Cross-tab data consistency and state management
- Mobile responsiveness across all dashboards and Vietnamese market views
- Long conversation context management in AI think tank

## Future Enhancements

### Short Term

- Portfolio rebalancing recommendations with actionable allocation commands
- Risk assessment and diversification analysis with Vietnamese market context
- Historical performance tracking and comparison charts
- Alert system for price targets and market events
- Enhanced Vietnamese market analysis (sector breakdown, fund performance comparison)

### Long Term

- Multi-currency portfolio support with automatic conversion handling
- Automated trading integration with Vietnamese market connectivity
- Social trading and community insights for local market trends
- Advanced machine learning-based prediction models with seasonal patterns
- Multi-asset class optimization (stocks, bonds, crypto, gold, funds)

---

**Feature Complexity**: High (multi-AI orchestration, Vietnamese market integration, complex state management)
**User Impact**: Core investment management and AI strategy generation with comprehensive market coverage
**Technical Debt**: Moderate (complex dual pricing system, extensive Vietnamese market integration)
**Test Coverage**: Needs expansion for Vietnamese market integration and AI orchestration reliability
