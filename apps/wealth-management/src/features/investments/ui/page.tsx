'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { useChat } from '@ai-sdk/react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { MaskedBalance } from '@/components/ui/masked-balance';
import {
  TrendingUp,
  Globe,
  Sparkles,
  Terminal,
  Database,
  RefreshCcw,
  Bitcoin,
  LayoutDashboard,
  Send,
  User,
  Bot,
  Layers,
} from 'lucide-react';
import { Button } from '@/components/ui/button';

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { MarketPulseDashboard } from '@/components/dashboard/market-pulse-dashboard';
import { MultiTimeframeDashboard } from '@/components/dashboard/multi-timeframe-dashboard';
import { SeasonalPatternsDashboard } from '../../../components/dashboard/seasonal-patterns-dashboard';
import { NewsAnalysisDashboard } from '@/components/dashboard/news-analysis-dashboard';
import { TickerAnalysisDashboard } from '@/components/dashboard/ticker-analysis-dashboard';
import { FmarketDashboard } from './fmarket-dashboard';

import { renderMessageContent, hasContent } from '@/features/chat/ui/chat-interface';

import { AIDataInsight } from '@/components/dashboard/ai-data-insight';

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
  const { messages, setMessages, sendMessage, status } = useChat({
    api: '/api/chat',
    body: { context: inputContext },
  } as any);
  const isChatBusy = status === 'streaming' || status === 'submitted';
 
  const [totalVND, setTotalVND] = useState(0);
  const [totalCrypto, setTotalCrypto] = useState(0);
  const [totalIFC, setTotalIFC] = useState(0);

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isChatBusy) return;
    void sendMessage({ text: input });
    setInput('');
  };

  const [error, setError] = useState<string | null>(null);
  const [newsAnalysis, setNewsAnalysis] = useState<Record<string, any>>({});
  const abortControllerRef = useRef<AbortController | null>(null);

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

        // --- Post-fetch: AI Price Extraction ---
        const symbolsToFetch: { symbol: string; type: 'crypto' | 'fund' }[] = [];
        const seenSymbols = new Set<string>();

        if (assetsData.crypto?.holdings) {
          assetsData.crypto.holdings.forEach((row: any) => {
            const symbol = row['Token'] || row['Currency'] || row['Asset'] || row['Symbol'];
            const priceKey = Object.keys(row).find(
              (k) =>
                k.toLowerCase().includes('price') ||
                k.toLowerCase().includes('rate') ||
                k.toLowerCase().includes('cost'),
            );
            if (
              symbol &&
              !seenSymbols.has(String(symbol)) &&
              (!row[priceKey || ''] || row[priceKey || ''] === '0' || row[priceKey || ''] === 0)
            ) {
              symbolsToFetch.push({ symbol: String(symbol), type: 'crypto' });
              seenSymbols.add(String(symbol));
            }
          });
        }

        if (assetsData.funds?.holdings) {
          assetsData.funds.holdings.forEach((row: any) => {
            const symbol = row['Certificate'] || row['Index'] || row['Asset'] || row['Symbol'];
            const priceKey = Object.keys(row).find(
              (k) =>
                k.toLowerCase().includes('price') ||
                k.toLowerCase().includes('rate') ||
                k.toLowerCase().includes('cost'),
            );
            if (
              symbol &&
              !seenSymbols.has(String(symbol)) &&
              (!row[priceKey || ''] || row[priceKey || ''] === '0' || row[priceKey || ''] === 0)
            ) {
              symbolsToFetch.push({ symbol: String(symbol), type: 'fund' });
              seenSymbols.add(String(symbol));
            }
          });
        }

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

  // Effect to re-calculate when prices arrive
  useEffect(() => {
    if (!accounts.length) return;

    // Calculate Account Total
    const accTotal = accounts.reduce((sum, acc) => sum + (acc.balance * (acc.currency === 'VND' ? 1 : exchangeRate)), 0);
    
    // Calculate Asset Totals
    let cryptoTotal = 0;
    if (assets?.crypto?.holdings) {
      assets.crypto.holdings.forEach((row: any) => {
        const symbol = row['Token'] || row['Currency'] || row['Asset'] || row['Symbol'];
        const amountKey = Object.keys(row).find(k => k.toLowerCase().includes('amount') || k.toLowerCase().includes('balance') || k.toLowerCase().includes('holding') || k.toLowerCase().includes('qty') || k.toLowerCase().includes('quantity'));
        const priceKey = Object.keys(row).find(k => k.toLowerCase().includes('price') || k.toLowerCase().includes('rate') || k.toLowerCase().includes('cost'));
        
        const amount = amountKey ? parseFloat(String(row[amountKey]).replace(/,/g, '')) : 0;
        let price = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
        if (!price && symbol && prices[String(symbol)]) price = prices[String(symbol)];
        
        if (!isNaN(amount) && !isNaN(price)) {
          let val = amount * price;
          if (price < 100000) val *= exchangeRate; // Assume USD/USDT if low price
          cryptoTotal += val;
        }
      });
    }

    let ifcTotal = 0;
    if (assets?.funds?.holdings) {
      assets.funds.holdings.forEach((row: any) => {
        const symbol = row['Certificate'] || row['Index'] || row['Asset'] || row['Symbol'];
        const amountKey = Object.keys(row).find(k => k.toLowerCase().includes('amount') || k.toLowerCase().includes('balance') || k.toLowerCase().includes('holding') || k.toLowerCase().includes('qty') || k.toLowerCase().includes('quantity'));
        const priceKey = Object.keys(row).find(k => k.toLowerCase().includes('price') || k.toLowerCase().includes('rate') || k.toLowerCase().includes('cost'));
        
        const amount = amountKey ? parseFloat(String(row[amountKey]).replace(/,/g, '')) : 0;
        let price = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
        if (!price && symbol && prices[String(symbol)]) price = prices[String(symbol)];
        
        if (!isNaN(amount) && !isNaN(price)) {
          ifcTotal += amount * price;
        }
      });
    }

    setTotalVND(accTotal);
    setTotalCrypto(cryptoTotal);
    setTotalIFC(ifcTotal);
  }, [prices, assets, accounts, exchangeRate]);

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

      let reading = true;
      while (reading) {
        const { done: isDone, value } = await reader.read();
        if (isDone) {
          reading = false;
          break;
        }

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.trim()) continue;
          try {
            const data = JSON.parse(line);

            if (data.type === 'error') {
              throw new Error(data.details || data.error);
            }

            if (data.type === 'message') {
              const msg = data.message;
              let textToRender = msg.content;

              if (msg.id === 'actions') {
                try {
                  const parsed = typeof msg.content === 'string' ? JSON.parse(msg.content) : msg.content;
                  if (parsed.executable_commands && parsed.executable_commands.length > 0) {
                    let md = `## ⚡ Alpha Generation Engine (Executable Commands)\n\n`;
                    parsed.executable_commands.forEach((cmd: any, idx: number) => {
                      const actionIcon =
                        cmd.action === 'ALLOCATE'
                          ? '💰'
                          : cmd.action === 'HEDGE'
                            ? '🛡️'
                            : cmd.action === 'OBSERVE'
                              ? '👁️'
                              : '🔧';
                      const targetName = cmd.target || cmd.exact_target || 'Target';
                      const rationale =
                        cmd.rationale || cmd.execution_logic || cmd.macro_hedge_reason || 'See context for details.';

                      md += `### Command ${idx + 1}: ${actionIcon} \`${cmd.action}\` -> **${targetName}**\n`;
                      md += `> **Rationale**: ${rationale}\n\n`;
                    });
                    textToRender = md;
                  }
                } catch (e) {
                  // Keep raw text
                }
              }

              const formattedMessage = {
                id: Date.now().toString() + '-' + msg.id,
                role: 'assistant' as const,
                name: msg.name,
                content: textToRender,
                parts: [{ type: 'text', text: textToRender }],
              } as any;

              setMessages((prev) => [...prev, formattedMessage]);
            }

            if (data.type === 'context') {
              setInputContext(data.context);
            }
          } catch (e) {
            console.error('Failed to parse stream line:', e);
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError(err.message || 'Failed to generate investment analysis');
        console.error('Analysis error:', err);
      }
    } finally {
      setIsAnalyzing(false);
    }
  };

  const renderAssetTable = (title: string, data?: AssetData, icon?: React.ReactNode) => {
    if (!data || data.holdings.length === 0) {
      return (
        <Card className="shadow-sm border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              {icon} {title}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex items-center justify-center p-8 text-muted-foreground bg-muted/20 border-t rounded-b-xl border-dashed">
            No specific data tracked in sheets.
          </CardContent>
        </Card>
      );
    }

    // Check if we already have a VND Value column
    const hasVndColumn = data.headers.some((h) => {
      const lowH = h.toLowerCase();
      return lowH.includes('vnd') && (lowH.includes('value') || lowH.includes('total') || lowH.includes('balance'));
    });

    const displayHeaders = hasVndColumn ? data.headers : [...data.headers, 'Est. VND Value'];

    // Calculate total for the VND/Estimated Value column only
    const columnTotals: (number | null)[] = displayHeaders.map((header) => {
      const isVirtualVnd = header === 'Est. VND Value';
      const isVndTotalCol = isVirtualVnd;

      // Only sum the VND/estimated value column
      if (!isVndTotalCol) return null;
      console.log(`Summing values for column: ${header}`);

      let sum = 0;
      let hasAny = false;

      for (const row of data.holdings) {
        const val = row[header];

        const amountKey = Object.keys(row).find((k) => {
          const lk = k.toLowerCase();
          return (
            lk.includes('amount') ||
            lk.includes('qty') ||
            lk.includes('quantity') ||
            lk.includes('unit') ||
            lk.includes('holding') ||
            lk.includes('balance') ||
            lk.includes('total') ||
            lk.includes('coin') ||
            lk.includes('coins') ||
            lk.includes('units') ||
            k.match(/^(amount|qty|quantity|holding|balance|total|coin|coins|units)$/i)
          );
        });
        const priceKey = Object.keys(row).find((k) => {
          const lk = k.toLowerCase();
          return lk.includes('price') || lk.includes('rate') || lk.includes('cost');
        });
        const symbol =
          row['Token'] || row['Currency'] || row['Asset'] || row['Certificate'] || row['Index'] || row['Symbol'];
        const sheetPrice = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
        const aiPrice = prices[String(symbol)] || 0;
        const finalPrice = !isNaN(sheetPrice) && sheetPrice !== 0 ? sheetPrice : aiPrice;

        let displayVal = val;
        if (!val || val === '0' || val === 0 || isVirtualVnd) {
          if (amountKey && row[amountKey] && finalPrice > 0) {
            const amount = parseFloat(String(row[amountKey]).replace(/,/g, ''));
            if (!isNaN(amount)) {
              let calculated = amount * finalPrice;
              if (title.toLowerCase().includes('crypto') && finalPrice < 100000) {
                calculated *= exchangeRate;
              }
              displayVal = calculated;
            }
          }
        }

        const num =
          displayVal !== undefined && displayVal !== null && displayVal !== ''
            ? parseFloat(String(displayVal).replace(/,/g, ''))
            : NaN;

        if (!isNaN(num) && num !== 0) {
          sum += num;
          hasAny = true;
        }
      }

      return hasAny ? parseFloat(sum.toFixed(2)) : null;
    });

    const isCryptoTable = title.toLowerCase().includes('crypto');
    const isIfcTable =
      title.toLowerCase().includes('fund') ||
      title.toLowerCase().includes('certificate') ||
      title.toLowerCase().includes('ifc');

    return (
      <Card className="shadow-sm border-border/50 overflow-hidden text-zinc-900 dark:text-zinc-100">
        <CardHeader className="bg-muted/30 border-b flex flex-row items-center justify-between py-3">
          <div className="flex items-center gap-2">
            <CardTitle className="text-lg flex flex-row items-center gap-2 m-0 p-0">
              {icon} {title}
            </CardTitle>
            <AIDataInsight
              type={`${title} Table`}
              description={`Detailed list of ${title.toLowerCase()} including balance, price, and estimated value.`}
              data={data.holdings}
            />
          </div>
          {isFetchingPrices && (
            <div className="flex items-center gap-2 text-[10px] text-primary animate-pulse font-medium bg-primary/10 px-2 py-0.5 rounded-full">
              <RefreshCcw className="w-3 h-3 animate-spin" />
              Refreshing AI Prices...
            </div>
          )}
        </CardHeader>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            {displayHeaders && displayHeaders.length > 0 && displayHeaders[0] !== '' && (
              <thead className="bg-muted/50 border-b text-muted-foreground uppercase tracking-wider text-[10px] font-semibold">
                <tr>
                  {displayHeaders.map((h, i) => (
                    <th key={i} className="text-left py-3 px-4 whitespace-nowrap">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
            )}
            <tbody className="divide-y">
              {data.holdings.map((row, i) => (
                <tr key={i} className="hover:bg-muted/25 transition-colors">
                  {displayHeaders.map((header, j) => (
                    <td key={j} className="py-3 px-4 whitespace-nowrap">
                      {(() => {
                        const h = header.toLowerCase();
                        const val = row[header] as string | number | undefined;

                        // Handle the virtual column or sensitive numeric columns
                        const isVirtualVnd = header === 'Est. VND Value';
                        const sensitiveHeaders = [
                          'amount',
                          'balance',
                          'vnd',
                          'unit',
                          'holding',
                          'qty',
                          'quantity',
                          'price',
                          'cost',
                          'value',
                          'usd',
                          'total',
                        ];

                        let displayVal = val;
                        const isVndTotalCell = h.includes('vnd') || h === 'total' || h === 'value' || isVirtualVnd;

                        const amountKey = Object.keys(row).find((k) => {
                          const lk = k.toLowerCase();
                          return (
                            lk.includes('amount') ||
                            lk.includes('qty') ||
                            lk.includes('quantity') ||
                            lk.includes('unit') ||
                            lk.includes('holding') ||
                            lk.includes('balance') ||
                            lk.includes('total') ||
                            lk.includes('coin') ||
                            lk.includes('coins') ||
                            lk.includes('units') ||
                            k.match(/^(amount|qty|quantity|holding|balance|spot|coin|coins|units)$/i)
                          );
                        });
                        const priceKey = Object.keys(row).find((k) => {
                          const lk = k.toLowerCase();
                          return lk.includes('price') || lk.includes('rate') || lk.includes('cost');
                        });

                        const symbol =
                          row['Token'] ||
                          row['Currency'] ||
                          row['Asset'] ||
                          row['Certificate'] ||
                          row['Index'] ||
                          row['Symbol'];
                        const sheetPrice = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
                        const aiPrice = prices[String(symbol)] || 0;
                        const finalPrice = !isNaN(sheetPrice) && sheetPrice !== 0 ? sheetPrice : aiPrice;
                        const isPriceAI = (!sheetPrice || sheetPrice === 0) && aiPrice > 0;

                        // If it's a total cell but empty, OR it's our virtual column, calculate it
                        if (isVndTotalCell && (!val || val === '0' || val === 0 || isVirtualVnd)) {
                          if (amountKey && row[amountKey] && finalPrice > 0) {
                            const amount = parseFloat(String(row[amountKey]).replace(/,/g, ''));
                            if (!isNaN(amount)) {
                              let calculated = amount * finalPrice;
                              // If it's crypto and price is likely in USD/USDT, multiply by exchange rate
                              if (title.toLowerCase().includes('crypto') && finalPrice < 100000) {
                                calculated *= exchangeRate;
                              }
                              displayVal = calculated;
                            }
                          }
                        }

                        const isNum =
                          displayVal &&
                          (typeof displayVal === 'number' ||
                            (typeof displayVal === 'string' && /^-?\d+(,\d+)*(\.\d+)?$/.test(displayVal.trim())));

                        if (isNum || sensitiveHeaders.some((sh) => h.includes(sh))) {
                          const unit =
                            row['Token'] ||
                            row['Currency'] ||
                            row['Asset'] ||
                            row['Certificate'] ||
                            row['Index'] ||
                            row['Symbol'];
                          const isCalculated = isVndTotalCell && (!val || val === '0' || val === 0 || isVirtualVnd);

                          const isVndCell = h.includes('vnd') || isVirtualVnd;
                          const content = (
                            <MaskedBalance
                              value={displayVal}
                              unit={!isVndCell && typeof unit === 'string' && unit !== displayVal ? unit : undefined}
                              currency={isVndCell ? 'VND' : h.includes('usd') ? 'USD' : 'none'}
                            />
                          );

                          if (isCalculated && (isVirtualVnd || isPriceAI)) {
                            const amount = amountKey ? parseFloat(String(row[amountKey]).replace(/,/g, '')) : NaN;
                            const isCrypto = title.toLowerCase().includes('crypto');
                            const isFund =
                              title.toLowerCase().includes('fund') || title.toLowerCase().includes('certificate');
                            const rateUsed = isCrypto && finalPrice < 100000 ? exchangeRate : 1;
                            const priceLabel = isFund ? 'NAV Rate' : isPriceAI ? 'AI Fetched Price' : priceKey;
                            const priceIsLikelyWrong = finalPrice === 1;

                            return (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="cursor-help border-b border-dotted border-primary/30 pb-0.5">
                                      {content}
                                    </span>
                                  </TooltipTrigger>
                                  <TooltipContent className="text-[10px] font-mono p-2">
                                    <div className="flex flex-col gap-1">
                                      <div className="text-muted-foreground uppercase text-[9px] font-bold">
                                        Calculation Formula
                                      </div>
                                      {!isNaN(amount) && amountKey ? (
                                        <div>
                                          {amount.toLocaleString()} ({amountKey})
                                        </div>
                                      ) : (
                                        <div className="text-yellow-500">Amount: N/A (no quantity column found)</div>
                                      )}
                                      <div
                                        className={`${isPriceAI ? 'text-indigo-500 font-bold' : ''} ${priceIsLikelyWrong ? 'text-yellow-500' : ''}`}
                                      >
                                        × {finalPrice.toLocaleString()} ({priceLabel})
                                        {priceIsLikelyWrong ? ' ⚠ price may be incorrect' : ''}
                                      </div>
                                      {rateUsed !== 1 && <div>× {rateUsed.toLocaleString()} (USDT Rate)</div>}
                                      <div className="border-t pt-1 mt-1 font-bold text-indigo-600 dark:text-indigo-400">
                                        = {Number(displayVal).toLocaleString()} VND
                                      </div>
                                    </div>
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            );
                          }

                          return content;
                        }
                        return displayVal || '';
                      })()}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
            <tfoot
              className={`border-t-2 font-semibold text-sm ${isCryptoTable ? 'bg-orange-500/10 border-orange-500/30' : isIfcTable ? 'bg-emerald-500/10 border-emerald-500/30' : 'bg-muted/40'}`}
            >
              <tr>
                {displayHeaders.map((header, j) => {
                  const isVirtualVnd = header === 'Est. VND Value';
                  const isVndTotalCol = isVirtualVnd;
                  const total = columnTotals[j];

                  if (j === 0) {
                    return (
                      <td key={j} className="py-3 px-4 whitespace-nowrap">
                        <span
                          className={`font-bold text-xs uppercase tracking-wide ${isCryptoTable ? 'text-orange-600 dark:text-orange-400' : isIfcTable ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground'}`}
                        >
                          {isCryptoTable ? 'Total Crypto' : isIfcTable ? 'Total IFC' : 'Total'}
                        </span>
                      </td>
                    );
                  }

                  // Only show total in VND/Est column, leave other columns empty
                  if (!isVndTotalCol) {
                    return <td key={j} className="py-3 px-4" />;
                  }

                  if (total !== null) {
                    return (
                      <td key={j} className="py-3 px-4 whitespace-nowrap font-bold">
                        <MaskedBalance value={total} currency="VND" />
                      </td>
                    );
                  }

                  return <td key={j} className="py-3 px-4" />;
                })}
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>
    );
  };

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


      <Tabs defaultValue="terminal" className="space-y-6">
        <TabsList className="bg-muted/50 p-1 border w-full justify-start overflow-x-auto overflow-y-hidden no-scrollbar">

          <TabsTrigger value="terminal" className="gap-2 px-4">
            <Terminal className="h-4 w-4" /> Think Tank Terminal
          </TabsTrigger>
          <TabsTrigger value="market-pulse" className="gap-2 px-4">
            <LayoutDashboard className="h-4 w-4" /> Market Pulse
          </TabsTrigger>
          <TabsTrigger value="multi-tf" className="gap-2 px-4">
            <TrendingUp className="h-4 w-4" /> Multi-TF Analysis
          </TabsTrigger>
          <TabsTrigger value="seasonality" className="gap-2 px-4">
            <Layers className="h-4 w-4" /> Seasonal Patterns
          </TabsTrigger>
          <TabsTrigger value="ticker-analyze" className="gap-2 px-4 text-amber-500 hover:text-amber-400">
            <Sparkles className="h-4 w-4" /> Ticker Analyze
          </TabsTrigger>
          <TabsTrigger value="fmarket" className="gap-2 px-4 text-indigo-500 hover:text-indigo-400">
            <TrendingUp className="h-4 w-4" /> Fmarket Insights
          </TabsTrigger>
          <TabsTrigger value="ledgers" className="gap-2 px-4">
            <Database className="h-4 w-4" /> Asset Ledgers
          </TabsTrigger>
        </TabsList>

        <TabsContent value="market-pulse" className="mt-0">
          <MarketPulseDashboard />
        </TabsContent>

        <TabsContent value="multi-tf" className="mt-0">
          <MultiTimeframeDashboard />
        </TabsContent>

        <TabsContent value="seasonality" className="mt-0">
          <SeasonalPatternsDashboard />
        </TabsContent>

        <TabsContent value="ticker-analyze" className="mt-0">
          <TickerAnalysisDashboard />
        </TabsContent>

        <TabsContent value="fmarket" className="mt-0">
          <FmarketDashboard />
        </TabsContent>


        <TabsContent value="terminal" className="mt-0 space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
             <NewsAnalysisDashboard 
                topic="Vietnam Market & Policy" 
                onAnalyzed={(data) => handleNewsAnalyzed('Vietnam', data)} 
             />
             <NewsAnalysisDashboard 
                topic="Interest Rates & Macro" 
                onAnalyzed={(data) => handleNewsAnalyzed('Macro', data)} 
             />
             <NewsAnalysisDashboard 
                topic="Global Geopolitics" 
                onAnalyzed={(data) => handleNewsAnalyzed('Geopolitics', data)} 
             />
          </div>

          <Card className="shadow-xl border-border/50 overflow-hidden bg-zinc-950 dark:bg-zinc-950 text-zinc-50">
            <CardHeader className="border-b border-zinc-800 bg-zinc-900/50 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2 text-zinc-100 mb-1">
                  <Terminal className="h-5 w-5 text-emerald-400" />
                  Live Think Tank Feed
                </CardTitle>
                <CardDescription className="text-zinc-400">
                  Real-time debate between 5 specialized Nobel-level AI analysts.
                </CardDescription>
              </div>
              <Button
                onClick={handleInitiateScan}
                disabled={isAnalyzing || isLoadingAccounts}
                className="gap-2 shadow-lg shadow-indigo-500/20 bg-indigo-600 hover:bg-indigo-700 text-white shrink-0"
              >
                {isAnalyzing ? <RefreshCcw className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                {isAnalyzing ? 'Synthesizing Global Data...' : 'Initiate Global Macro Scan'}
              </Button>
            </CardHeader>

            <CardContent className="p-0">
              <div className="h-[600px] break-words whitespace-pre-wrap overflow-y-auto p-6 font-mono text-sm leading-relaxed prose prose-invert prose-emerald max-w-[100%] overflow-x-hidden">
                {error && (
                  <div className="text-red-400 p-4 border border-red-500/30 rounded-lg bg-red-500/10 mb-4">
                    <p className="font-bold">⚠️ Error</p>
                    <p className="text-sm">{error}</p>
                  </div>
                )}

                {!messages.length && !isAnalyzing && !error && (
                  <div className="flex flex-col items-center justify-center h-full text-zinc-500 gap-4">
                    <Globe className="h-12 w-12 opacity-20" />
                    <p>System standing by. Click "Initiate Global Macro Scan" to begin the briefing.</p>
                  </div>
                )}

                {!!inputContext && (
                  <div className="mb-6 p-4 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300">
                    <h3 className="text-zinc-400 font-bold mb-3 flex items-center gap-2">
                      <Database className="h-4 w-4 text-emerald-500" /> AI Input Data Context
                    </h3>
                    <details className="cursor-pointer group">
                      <summary className="text-indigo-400 group-hover:text-indigo-300 font-medium select-none">
                        Show Raw AI Input Context (JSON & Search Data)
                      </summary>
                      <pre className="mt-3 p-3 bg-zinc-950 rounded overflow-x-auto text-[10px] text-zinc-500 whitespace-pre-wrap border border-zinc-800/50">
                        {inputContext ? JSON.stringify(inputContext, null, 2) : ''}
                      </pre>
                    </details>
                  </div>
                )}

                {messages.map((m) => {
                  if (!hasContent(m)) return null;
                  return (
                    <div
                      key={m.id}
                      className={`mb-6 flex gap-4 ${m.role === 'user' ? 'flex-row-reverse' : 'flex-row'}`}
                    >
                      <div
                        className={`flex h-8 w-8 shrink-0 select-none items-center justify-center rounded-lg border shadow-sm ${m.role === 'user' ? 'bg-indigo-600 border-indigo-500 text-white' : 'bg-emerald-950 border-emerald-800 text-emerald-400'}`}
                      >
                        {m.role === 'user' ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
                      </div>
                      <div
                        className={`flex flex-col gap-2 rounded-2xl px-5 py-3.5 text-sm max-w-[85%] leading-relaxed overflow-x-auto ${m.role === 'user' ? 'bg-indigo-600 text-white rounded-tr-none' : 'bg-zinc-900 border border-zinc-800 text-zinc-300 rounded-tl-none shadow-sm'}`}
                      >
                        {(m as any).name && (
                          <div className="text-xs font-bold text-emerald-400 mb-1">{(m as any).name}</div>
                        )}
                        {renderMessageContent(m)}
                      </div>
                    </div>
                  );
                })}

                {(isAnalyzing || isChatBusy) && !messages.length && (
                  <div className="flex items-center gap-2 text-emerald-400">
                    <span className="inline-block w-2 h-4 bg-emerald-400 animate-pulse" />
                    <span className="text-sm">Analyzing global macro landscape...</span>
                  </div>
                )}
              </div>
            </CardContent>
            {messages.length > 0 && (
              <div className="p-4 bg-zinc-900 border-t border-zinc-800">
                <form onSubmit={handleFormSubmit} className="flex gap-2 relative">
                  <input
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder="Connect to Think Tank (e.g. 'How should I rebalance based on this?')"
                    className="flex-1 bg-zinc-950 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-100 focus:outline-none focus:ring-1 focus:ring-emerald-500"
                    disabled={isChatBusy}
                  />
                  <Button
                    type="submit"
                    className="bg-emerald-600 hover:bg-emerald-700 text-white px-6 w-auto"
                    disabled={isChatBusy || !input.trim()}
                  >
                    <Send className="h-4 w-4" />
                  </Button>
                </form>
              </div>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="ledgers" className="space-y-6 mt-0">
          {renderAssetTable('Crypto Holdings', assets?.crypto, <Bitcoin className="h-5 w-5 text-orange-500" />)}
          {renderAssetTable(
            'Investment Fund Certificates',
            assets?.funds,
            <TrendingUp className="h-5 w-5 text-emerald-500" />,
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
