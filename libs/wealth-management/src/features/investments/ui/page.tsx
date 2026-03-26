'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { useChat } from '@ai-sdk/react';
import { Globe, Sparkles, Terminal, Database, LayoutDashboard, TrendingUp, Layers, Newspaper } from 'lucide-react';

import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  MarketPulseDashboard,
  MultiTimeframeDashboard,
  SeasonalPatternsDashboard,
  NewsAnalysisDashboard,
  TickerAnalysisDashboard
} from '@wealth-management/ui';
import { FmarketDashboard } from './fmarket-dashboard';
import { ThinkTank } from './components/ThinkTank';
import { AssetLedgers } from './components/AssetLedgers';

interface Account {
  balance: number;
  currency: string;
  type: string;
  name: string;
}

interface AssetData {
  headers: string[];
  holdings: Record<string, unknown>[];
}

interface AssetResponse {
  crypto: AssetData;
  funds: AssetData;
}

export default function InvestmentsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [assets, setAssets] = useState<AssetResponse | null>(null);
  const [exchangeRate, setExchangeRate] = useState(25400);
  const [prices, setPrices] = useState<Record<string, number>>({});
  const [isLoadingAccounts, setIsLoadingAccounts] = useState(true);
  const [isFetchingPrices, setIsFetchingPrices] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [inputContext, setInputContext] = useState<unknown>(null);
  const [input, setInput] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [newsAnalysis, setNewsAnalysis] = useState<Record<string, any>>({});
  const [mounted, setMounted] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  const { messages, setMessages, sendMessage, status } = useChat({
    api: '/api/chat',
    body: { context: inputContext },
  } as any);
  const isChatBusy = status === 'streaming' || status === 'submitted';

  const handleNewsAnalyzed = useCallback((topic: string, data: any) => {
    setNewsAnalysis(prev => {
      if (prev[topic] === data) return prev;
      return { ...prev, [topic]: data };
    });
  }, []);

  useEffect(() => {
    async function fetchData() {
      try {
        setIsLoadingAccounts(true);
        const [accRes, assetsRes, rateRes] = await Promise.all([
          fetch('/api/accounts'),
          fetch('/api/investments/assets'),
          fetch('/api/exchange-rate'),
        ]);

        const accData = await accRes.json();
        const assetsData = await assetsRes.json();
        const { rate } = await rateRes.json();

        setAccounts(accData);
        setAssets(assetsData);
        setExchangeRate(rate || 25400);

        const symbolsToFetch: { symbol: string; type: 'crypto' | 'fund' }[] = [];
        const seenSymbols = new Set<string>();

        const extractSymbols = (holdings: any[], type: 'crypto' | 'fund') => {
          if (!holdings) return;
          holdings.forEach((row: any) => {
            const symbolKey = type === 'crypto' ? ['Token', 'Currency', 'Asset', 'Symbol'] : ['Certificate', 'Index', 'Asset', 'Symbol'];
            const symbol = symbolKey.map(k => row[k]).find(v => !!v);
            const priceKey = Object.keys(row).find(k => k.toLowerCase().includes('price') || k.toLowerCase().includes('rate') || k.toLowerCase().includes('cost'));
            if (symbol && !seenSymbols.has(String(symbol)) && (!row[priceKey || ''] || row[priceKey || ''] === '0' || row[priceKey || ''] === 0)) {
              symbolsToFetch.push({ symbol: String(symbol), type });
              seenSymbols.add(String(symbol));
            }
          });
        };

        if (assetsData.crypto) extractSymbols(assetsData.crypto.holdings, 'crypto');
        if (assetsData.funds) extractSymbols(assetsData.funds.holdings, 'fund');

        if (symbolsToFetch.length > 0) {
          setIsFetchingPrices(true);
          void fetch('/api/investments/prices', {
            method: 'POST',
            body: JSON.stringify({ symbols: symbolsToFetch }),
          })
            .then((res) => res.json() as Promise<{ prices: Record<string, number> }>)
            .then((data) => {
              if (data.prices) setPrices(data.prices);
            })
            .finally(() => setIsFetchingPrices(false));
        }
      } catch (err) {
        console.error('Failed to load data', err);
        setError('Failed to load account data');
      } finally {
        setIsLoadingAccounts(false);
      }
    }
    void fetchData();
  }, []);

  const handleInitiateScan = async () => {
    if (!accounts.length) {
      setError('No accounts loaded');
      return;
    }

    setIsAnalyzing(true);
    setMessages([]);
    setInputContext(null);
    setError(null);
    abortControllerRef.current = new AbortController();

    try {
      const response = await fetch('/api/ai/investment-analysis', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ accounts, prices, newsAnalysis }),
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.details || `API error: ${response.statusText}`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error('Failed to start response stream');

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.trim()) continue;
          try {
            const data = JSON.parse(line);
            if (data.type === 'error') throw new Error(data.details || data.error);
            if (data.type === 'message') {
              const msg = data.message;
              let textToRender = msg.content;
              if (msg.id === 'actions') {
                try {
                  const parsed = typeof msg.content === 'string' ? JSON.parse(msg.content) : msg.content;
                  if (parsed.executable_commands && parsed.executable_commands.length > 0) {
                    let md = `## ⚡ Alpha Generation Engine (Executable Commands)\n\n`;
                    parsed.executable_commands.forEach((cmd: any, idx: number) => {
                      const actionIcon = cmd.action === 'ALLOCATE' ? '💰' : cmd.action === 'HEDGE' ? '🛡️' : cmd.action === 'OBSERVE' ? '👁️' : '🔧';
                      md += `### Command ${idx + 1}: ${actionIcon} \`${cmd.action}\` -> **${cmd.target}**\n`;
                      md += `> **Rationale**: ${cmd.rationale}\n\n`;
                    });
                    textToRender = md;
                  }
                } catch (e) {
                  console.error('Failed to parse actions content', e);
                }
              }
              const formattedMessage = {
                id: Date.now().toString() + '-' + msg.id,
                role: 'assistant' as const,
                name: msg.name,
                content: textToRender,
              } as any;
              setMessages((prev: any) => [...prev, formattedMessage]);
            }
            if (data.type === 'context') setInputContext(data.context);
          } catch (e) {
            console.error('Failed to parse stream line:', e);
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError(err.message || 'Failed to generate investment analysis');
      }
    } finally {
      setIsAnalyzing(false);
    }
  };

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isChatBusy) return;
    sendMessage({ text: input });
    setInput('');
  };

  if (!mounted) return null;

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-10">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Globe className="h-6 w-6 text-indigo-500" />
            Global Macro War Room
          </h2>
          <p className="text-muted-foreground text-sm">
            AI-driven geopolitical and economic synthesis for your portfolio.
          </p>
        </div>
      </div>

      <Tabs defaultValue="think-tank" className="space-y-6">
        <TabsList className="bg-muted/50 p-1 border w-full justify-start overflow-x-auto no-scrollbar">
          <TabsTrigger value="think-tank" className="gap-2 px-4"><Terminal className="h-4 w-4" /> Think Tank</TabsTrigger>
          <TabsTrigger value="news-analyze" className="gap-2 px-4"><Newspaper className="h-4 w-4" /> News Analyze</TabsTrigger>
          <TabsTrigger value="market-pulse" className="gap-2 px-4"><LayoutDashboard className="h-4 w-4" /> Market Pulse</TabsTrigger>
          <TabsTrigger value="multi-tf" className="gap-2 px-4"><TrendingUp className="h-4 w-4" /> Multi-TF Analysis</TabsTrigger>
          <TabsTrigger value="seasonality" className="gap-2 px-4"><Layers className="h-4 w-4" /> Seasonal Patterns</TabsTrigger>
          <TabsTrigger value="ticker-analyze" className="gap-2 px-4 text-amber-500 hover:text-amber-400"><Sparkles className="h-4 w-4" /> Ticker Analyze</TabsTrigger>
          <TabsTrigger value="fmarket" className="gap-2 px-4 text-indigo-500 hover:text-indigo-400"><TrendingUp className="h-4 w-4" /> Fmarket Insights</TabsTrigger>
          <TabsTrigger value="ledgers" className="gap-2 px-4"><Database className="h-4 w-4" /> Asset Ledgers</TabsTrigger>
        </TabsList>

        <TabsContent value="think-tank" className="mt-0">
          <ThinkTank
            messages={messages}
            isAnalyzing={isAnalyzing}
            isChatBusy={isChatBusy}
            isLoadingAccounts={isLoadingAccounts}
            onInitiateScan={handleInitiateScan}
            error={error}
            inputContext={inputContext}
            input={input}
            onInputChange={setInput}
            onFormSubmit={handleFormSubmit}
          />
        </TabsContent>

        <TabsContent value="news-analyze" className="mt-0 space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
             <NewsAnalysisDashboard topic="Vietnam Market & Policy" onAnalyzed={(data) => handleNewsAnalyzed('Vietnam', data)} />
             <NewsAnalysisDashboard topic="Interest Rates & Macro" onAnalyzed={(data) => handleNewsAnalyzed('Macro', data)} />
             <NewsAnalysisDashboard topic="Global Geopolitics" onAnalyzed={(data) => handleNewsAnalyzed('Geopolitics', data)} />
          </div>
        </TabsContent>

        <TabsContent value="market-pulse" className="mt-0"><MarketPulseDashboard /></TabsContent>
        <TabsContent value="multi-tf" className="mt-0"><MultiTimeframeDashboard /></TabsContent>
        <TabsContent value="seasonality" className="mt-0"><SeasonalPatternsDashboard /></TabsContent>
        <TabsContent value="ticker-analyze" className="mt-0"><TickerAnalysisDashboard /></TabsContent>
        <TabsContent value="fmarket" className="mt-0"><FmarketDashboard /></TabsContent>
        <TabsContent value="ledgers" className="mt-0">
          <AssetLedgers
            assets={assets}
            isFetchingPrices={isFetchingPrices}
            prices={prices}
            exchangeRate={exchangeRate}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
