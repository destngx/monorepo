'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
  PieChart as RePieChart,
  Pie,
} from 'recharts';
import { Compass, TrendingUp, Activity, Zap, Flame, ArrowUpRight, ArrowDownRight, Layers } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '../card';
import { Badge } from '../badge';
import { AIDataInsight } from '../ai-data-insight';
import { GLASS_CARD, TERMINAL_FONT } from './constants';
import { SeasonalityStatsTable } from './SeasonalityStatsTable';

export function TechnicalAnalysisView({
  technicals,
  market,
  showSeasonality = true,
}: {
  technicals?: any;
  market: string;
  showSeasonality?: boolean;
}) {
  if (!technicals) return null;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-2 mb-2">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Compass className={`w-5 h-5 ${market === 'US' ? 'text-indigo-500' : 'text-emerald-500'}`} />
            <h3 className="text-lg font-bold tracking-tight uppercase underline decoration-zinc-800 underline-offset-8 decoration-2">
              Technical Alpha Suite: {market}
            </h3>
          </div>
          {technicals.n && (
            <Badge variant="outline" className="text-[9px] font-mono border-zinc-800 text-zinc-500 opacity-70">
              SỐ NẾN: {technicals.n}
            </Badge>
          )}
        </div>
        <AIDataInsight
          type="Technical Alpha suite"
          description={`Institutional grade technical analysis including ICT signatures, market cycles, and sentiment synthesis for ${market}.`}
          data={technicals}
          market={market}
          timeframe="1D"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Trend Analysis Card */}
        <Card className={`${GLASS_CARD} p-6 border-zinc-200 dark:border-zinc-800/50`}>
          <div className="flex items-center justify-between mb-6">
            <h4 className="text-sm font-bold text-zinc-400 uppercase tracking-widest flex items-center gap-2">
              <TrendingUp className="w-4 h-4" /> Xu hướng (Trend)
            </h4>
          </div>

          <div className="space-y-5">
            <div className="flex items-center justify-between pb-3 border-b border-zinc-200/10">
              <span className="text-xs text-zinc-400 font-medium">Hướng</span>
              <span
                className={`text-sm font-black ${technicals.trend?.directionEn === 'Up' ? 'text-emerald-500' : technicals.trend?.directionEn === 'Down' ? 'text-rose-500' : 'text-zinc-500'}`}
              >
                {technicals.trend?.direction || 'N/A'}
              </span>
            </div>

            <div className="flex items-center justify-between pb-3 border-b border-zinc-200/10">
              <span className="text-xs text-zinc-400 font-medium">Độ mạnh</span>
              <span className="text-sm font-black text-white">{technicals.trend?.strength}%</span>
            </div>

            <div className="flex items-center justify-between">
              <span className="text-xs text-zinc-400 font-medium">Độ tin cậy</span>
              <span className="text-sm font-black text-white">{technicals.trend?.confidence}%</span>
            </div>

            <div className="pt-4 mt-2">
              <div className="bg-zinc-900/40 p-3 rounded-xl border border-zinc-800/30 font-mono text-[9px] text-zinc-400">
                <span className="text-zinc-500">RSI(14):</span>{' '}
                <span className="text-zinc-300 mr-2">{technicals.indicators?.rsi?.toFixed(2)}</span>
                <span className="text-zinc-500 ml-2">EMA20:</span>{' '}
                <span className="text-zinc-300 mr-2">{technicals.indicators?.ema20?.toLocaleString()}</span>
                <span className="text-zinc-500 ml-2">EMA50:</span>{' '}
                <span className="text-zinc-300">{technicals.indicators?.ema50?.toLocaleString()}</span>
              </div>
            </div>
          </div>
        </Card>

        {/* Cycle Analysis Card */}
        <Card className={`${GLASS_CARD} p-6 border-zinc-200 dark:border-zinc-800/50`}>
          <div className="flex items-center justify-between mb-6">
            <h4 className="text-sm font-bold text-zinc-400 uppercase tracking-widest flex items-center gap-2">
              <Activity className="w-4 h-4" /> Chu kỳ thị trường (Cycle)
            </h4>
          </div>

          <div className="flex gap-8">
            <div className="flex-1 space-y-4">
              <div className="flex items-center justify-between border-b border-zinc-200/10 pb-2">
                <span className="text-[10px] text-zinc-400 font-medium uppercase">Pha hiện tại</span>
                <span
                  className={`text-[11px] font-black ${technicals.cycle?.phase === 'Markup' ? 'text-emerald-500' : 'text-rose-500'} uppercase`}
                >
                  {technicals.cycle?.descriptionVi?.includes('(')
                    ? technicals.cycle.descriptionVi.split('(')[0].trim()
                    : technicals.cycle?.descriptionVi || 'N/A'}
                  <span className="opacity-50 ml-1">({technicals.cycle?.phase || 'N/A'})</span>
                </span>
              </div>

              <div className="flex items-center justify-between border-b border-zinc-200/10 pb-2">
                <span className="text-[10px] text-zinc-400 font-medium uppercase">Độ tin cậy</span>
                <span className="text-xs font-black text-white">{technicals.cycle?.confidence || 0}%</span>
              </div>

              <div className="space-y-2 pt-2">
                {(technicals.cycle?.phases || []).map((p: any, i: number) => (
                  <div key={i} className="flex items-center justify-between text-[10px]">
                    <span className="text-zinc-500 font-medium">{p.label}</span>
                    <span className="font-mono font-bold text-zinc-300 tabular-nums">{p.value}.0%</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="w-[100px] h-[100px] relative min-w-0 min-h-0">
              <ResponsiveContainer width="100%" height="100%" minWidth={0}>
                <RePieChart>
                  <Pie
                    data={technicals.cycle?.phases || []}
                    cx="50%"
                    cy="50%"
                    innerRadius={30}
                    outerRadius={45}
                    paddingAngle={4}
                    dataKey="value"
                  >
                    {(technicals.cycle?.phases || []).map((entry: any, index: number) => (
                      <Cell key={index} fill={entry.color || '#3f3f46'} stroke="none" />
                    ))}
                  </Pie>
                </RePieChart>
              </ResponsiveContainer>
              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <div className="w-1.5 h-1.5 rounded-full bg-zinc-800 ring-2 ring-zinc-700/50" />
              </div>
            </div>
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* ICT Concepts Card */}
        <Card className={`${GLASS_CARD} p-6 border-zinc-200 dark:border-zinc-800/50`}>
          <div className="flex items-center justify-between mb-4">
            <h4 className="text-[10px] font-bold text-amber-500 uppercase tracking-[0.2em] flex items-center gap-2">
              <Zap className="w-3.5 h-3.5" /> ICT Concepts (FVG / OB)
            </h4>
            <Badge variant="outline" className="text-[9px] uppercase border-zinc-800 text-zinc-500">
              Institutional Signatures
            </Badge>
          </div>
          <div className="space-y-3">
            {technicals.ict?.fvgs?.length > 0 ? (
              technicals.ict.fvgs.map((fvg: any, idx: number) => (
                <div
                  key={idx}
                  className="flex items-center justify-between p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/40 group hover:border-amber-500/30 transition-all"
                >
                  <div className="flex items-center gap-3">
                    <div
                      className={`p-1.5 rounded-lg ${fvg.type === 'BULLISH' ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}`}
                    >
                      {fvg.type === 'BULLISH' ? (
                        <ArrowUpRight className="w-4 h-4" />
                      ) : (
                        <ArrowDownRight className="w-4 h-4" />
                      )}
                    </div>
                    <div>
                      <div className="text-xs font-bold text-white leading-tight">FVG: {fvg.type} GAP</div>
                      <div className="text-[10px] text-zinc-500 font-mono tracking-tighter">
                        Range: {fvg.bottom.toLocaleString()} - {fvg.top.toLocaleString()}
                      </div>
                    </div>
                  </div>
                  <div
                    className={`text-xs font-black font-mono ${fvg.type === 'BULLISH' ? 'text-emerald-500' : 'text-rose-500'}`}
                  >
                    {fvg.gap.toFixed(2)} pts
                  </div>
                </div>
              ))
            ) : (
              <div className="py-8 text-center text-zinc-500 text-xs italic opacity-50 border border-dashed border-zinc-800 rounded-2xl bg-zinc-900/20">
                No institutional imbalances detected in current window.
              </div>
            )}
          </div>
        </Card>

        {/* Sentiment Score Card */}
        <Card className={`${GLASS_CARD} p-6 border-zinc-200 dark:border-zinc-800/50`}>
          <div className="flex items-center justify-between mb-4">
            <h4 className="text-[10px] font-bold text-indigo-500 uppercase tracking-[0.2em] flex items-center gap-2">
              <Flame className="w-3.5 h-3.5" /> Sentiment Synthesis
            </h4>
            <Badge variant="outline" className="text-[9px] uppercase border-zinc-800 text-zinc-500">
              Crowd Psychology
            </Badge>
          </div>
          <div className="flex gap-6 items-center">
            <div className="relative w-24 h-24 flex items-center justify-center">
              <svg className="w-full h-full transform -rotate-90">
                <circle
                  cx="48"
                  cy="48"
                  r="40"
                  stroke="currentColor"
                  strokeWidth="8"
                  fill="transparent"
                  className="text-zinc-800"
                />
                <circle
                  cx="48"
                  cy="48"
                  r="40"
                  stroke="currentColor"
                  strokeWidth="8"
                  fill="transparent"
                  strokeDasharray={251.2}
                  strokeDashoffset={251.2 * (1 - (technicals.sentiment?.score || 50) / 100)}
                  strokeLinecap="round"
                  className={`${(technicals.sentiment?.score || 50) > 65 ? 'text-emerald-500' : (technicals.sentiment?.score || 50) < 35 ? 'text-rose-500' : 'text-indigo-500'} transition-all duration-1000`}
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className="text-xl font-black tabular-nums">{technicals.sentiment?.score || 0}</span>
                <span className="text-[8px] font-bold text-zinc-500 uppercase">Index</span>
              </div>
            </div>
            <div className="flex-1 space-y-2">
              <div
                className={`text-lg font-black tracking-tight ${(technicals.sentiment?.score || 50) > 65 ? 'text-emerald-400' : (technicals.sentiment?.score || 50) < 35 ? 'text-rose-400' : 'text-white'}`}
              >
                Psychology: {technicals.sentiment?.labelVi || 'Neutral'}
              </div>
              <p className="text-xs text-zinc-400 leading-relaxed">
                Synthesis of VIX volatility, crowd RSI positioning, and trend velocity. Current state suggests
                <span className="font-bold ml-1 text-zinc-200">
                  {(technicals.sentiment?.score || 50) > 65
                    ? 'excessive optimism'
                    : (technicals.sentiment?.score || 50) < 35
                      ? 'extreme capitulation'
                      : 'balanced equilibrium'}
                  .
                </span>
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Support/Resistance */}
      <Card className={`${GLASS_CARD} p-0 overflow-hidden border-zinc-200 dark:border-zinc-800/50`}>
        <div className="bg-zinc-50 dark:bg-zinc-800/20 p-4 border-b border-zinc-100 dark:border-zinc-800/50">
          <div className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">Key Levels (S/R)</div>
        </div>
        <div className="p-4 space-y-4">
          {(technicals.supportResistance || []).slice(0, 2).map((item: any, idx: number) => (
            <div key={idx} className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold font-mono">{item.symbol}</span>
                <span className="text-[10px] text-zinc-400">
                  BOLLINGER WIDTH:{' '}
                  {(((item.bollingerUpper - item.bollingerLower) / item.bollingerMid) * 100).toFixed(1)}%
                </span>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="bg-rose-500/5 p-2 rounded border border-rose-500/10">
                  <div className="text-[9px] uppercase text-rose-500 font-bold mb-1">Resistance</div>
                  <div className="flex flex-wrap gap-1">
                    {(item.resistance || []).map((r: number, i: number) => (
                      <span key={i} className="text-[10px] font-mono">
                        {r.toLocaleString()}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="bg-emerald-500/5 p-2 rounded border border-emerald-500/10">
                  <div className="text-[9px] uppercase text-emerald-500 font-bold mb-1">Support</div>
                  <div className="flex flex-wrap gap-1">
                    {(item.support || []).map((s: number, i: number) => (
                      <span key={i} className="text-[10px] font-mono">
                        {s.toLocaleString()}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </Card>

      {showSeasonality && technicals.seasonality && (
        <>
          {/* Seasonality Table */}
          <SeasonalityStatsTable seasonality={technicals.seasonality} />

          {/* Seasonality Chart (Small footer version) */}
          <Card className={`${GLASS_CARD} p-5 border-zinc-200 dark:border-zinc-800/50`}>
            <div className="text-[10px] font-bold uppercase tracking-widest text-zinc-500 mb-4 opacity-50">
              Visual Pattern
            </div>
            <div className="h-[80px] w-full mt-2 min-w-0 min-h-0">
              <ResponsiveContainer width="100%" height="100%" minWidth={0}>
                <BarChart data={technicals.seasonality.filter((s: any) => s.type === 'day')}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(128,128,128,0.1)" />
                  <Bar dataKey="avgReturn" radius={[2, 2, 0, 0]}>
                    {technicals.seasonality
                      .filter((s: any) => s.type === 'day')
                      .map((entry: any, index: number) => (
                        <Cell key={index} fill={entry.avgReturn >= 0 ? '#10b981' : '#f43f5e'} opacity={0.6} />
                      ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
