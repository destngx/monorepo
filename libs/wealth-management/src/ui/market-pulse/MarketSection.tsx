import { Globe, Search, Zap, AlertCircle } from 'lucide-react';
import { Badge } from '../../ui/badge';
import { Card, CardHeader, CardContent, CardTitle } from '../../ui/card';
import { AIDataInsight } from '../ai-data-insight';
import { IntelligenceBanner } from './IntelligenceBanner';
import { MarketSignalCard } from './MarketSignalCard';
import { TimeframeRelationshipGrid } from './TimeframeRelationshipGrid';
import { MarketBarChart } from './MarketBarChart';
import { MarketSnapshotTable } from './MarketSnapshotTable';
import { CorrelationHeatmap } from './CorrelationHeatmap';
import { TechnicalAnalysisView } from './TechnicalAnalysisView';
import { AssetValuationTerminal } from './AssetValuationTerminal';
import { GLASS_CARD } from './constants';

interface MarketSectionProps {
  market: 'US' | 'VN';
  marketData: any;
  timeframe: string;
}

export function MarketSection({ market, marketData, timeframe }: MarketSectionProps) {
  const isUs = market === 'US';
  const accentColor = isUs ? 'indigo' : 'emerald';
  const flag = isUs ? '🇺🇸' : '🇻🇳';

  return (
    <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500">
      <IntelligenceBanner
        scenarios={marketData?.scenarios}
        capitalFlow={marketData?.capitalFlow}
        market={market}
        timeframe={timeframe}
      />

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pb-4">
        <div className="lg:col-span-8">
          <MarketSignalCard signals={marketData?.technicals?.signals} market={market} />
        </div>
        <div className="lg:col-span-4">
          <TimeframeRelationshipGrid
            relationships={marketData?.technicals?.timeframeRelationships}
            entryScore={marketData?.technicals?.entryTimingScore}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pb-4">
        <div className="lg:col-span-8 space-y-6">
          <Card className={`${GLASS_CARD} border-zinc-700/30 group relative overflow-hidden`}>
            <div className={`absolute inset-0 bg-gradient-to-br from-${isUs ? 'indigo' : 'emerald'}-500/5 via-transparent to-transparent pointer-events-none`} />
            <CardHeader className="pb-2 border-b border-zinc-800/50 flex flex-row items-center justify-between">
              <div className="flex items-center gap-2">
                <Globe className={`w-3.5 h-3.5 ${isUs ? 'text-indigo-400' : 'text-emerald-400'}`} />
                <span className="text-[11px] font-bold tracking-widest text-zinc-400 uppercase">
                  Velocity Spectrum {flag}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-[9px] font-mono border-zinc-800 text-zinc-500 uppercase">
                  {timeframe.toUpperCase()} Active
                </Badge>
                <AIDataInsight
                  type="Velocity Bar Chart"
                  description={`Bar chart showing percentage change of ${market} market assets.`}
                  data={marketData?.assets || []}
                  market={market}
                  timeframe={timeframe}
                />
              </div>
            </CardHeader>
            <CardContent className="pt-8">
              <div className="h-[280px] w-full min-w-0 min-h-0">
                <MarketBarChart data={marketData?.assets || []} />
              </div>
            </CardContent>
          </Card>

          <Card className={`${GLASS_CARD}`}>
            <CardHeader className="bg-zinc-900/40 border-b border-zinc-800/50 py-3 px-4 flex flex-row items-center justify-between">
              <CardTitle className="text-xs flex items-center gap-2 font-mono text-zinc-400 uppercase">
                <Search className="w-3.5 h-3.5" />
                Snapshot Terminal
              </CardTitle>
              <AIDataInsight
                type="Snapshot Table"
                description={`Table showing ${market} market instruments with price and performance.`}
                data={marketData?.assets || []}
                market={market}
                timeframe={timeframe}
              />
            </CardHeader>
            <CardContent className="p-0">
              <MarketSnapshotTable assets={marketData?.assets || []} />
            </CardContent>
          </Card>
        </div>

        <div className="lg:col-span-4 space-y-6">
          <div className="relative">
            <div className="absolute top-3 right-3 z-10">
              <AIDataInsight
                type="Correlation Heatmap"
                description={`Correlation matrix for ${market} market assets.`}
                data={{ assets: marketData?.assetList || [], matrix: marketData?.correlationMatrix || [] }}
                market={market}
                timeframe={timeframe}
              />
            </div>
            <CorrelationHeatmap
              title={`${market} Heatmatrix`}
              assets={marketData?.assetList || []}
              matrix={marketData?.correlationMatrix || []}
            />
          </div>

          <Card className={`${GLASS_CARD} p-4 space-y-4`}>
            <div className="flex items-center justify-between pb-2 border-b border-zinc-800/50">
              <div className={`flex items-center gap-2 text-zinc-400 text-[10px] font-mono uppercase tracking-widest`}>
                <Zap className={`w-3 h-3 text-${isUs ? 'orange' : 'emerald'}-400`} />
                Dominant Driver
              </div>
              {marketData?.drivers?.capitalFlowSignal && (
                <Badge
                  variant="outline"
                  className={`text-[8px] font-mono border-none ${
                    marketData.drivers.capitalFlowSignal === 'RISK-ON'
                      ? 'bg-emerald-500/10 text-emerald-500'
                      : marketData.drivers.capitalFlowSignal === 'DEFENSIVE'
                        ? 'bg-rose-500/10 text-rose-500'
                        : 'bg-amber-500/10 text-amber-500'
                  }`}
                >
                  {marketData.drivers.capitalFlowSignal}
                </Badge>
              )}
            </div>
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="text-lg font-bold tracking-tight text-white leading-tight">
                  {marketData?.drivers?.summaryEn || '...'}
                </div>
                <div className="text-xs text-zinc-500 italic font-mono leading-relaxed">
                  {marketData?.drivers?.summaryVi}
                </div>
              </div>

              {marketData?.drivers?.correlationSignalEn && (
                <div className="pt-3 border-t border-zinc-800/30 space-y-2">
                  <div className={`flex items-center gap-1.5 text-[9px] font-bold ${isUs ? 'text-amber-500' : 'text-emerald-500'} uppercase tracking-tighter`}>
                    <AlertCircle className="w-3 h-3" /> Correlation Signal
                  </div>
                  <p className="text-[11px] text-zinc-400 leading-snug font-mono italic">
                    {marketData.drivers.correlationSignalEn}
                  </p>
                  <p className="text-[10px] text-zinc-600 leading-snug font-mono italic">
                    {marketData.drivers.correlationSignalVi}
                  </p>
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>

      <div className="pt-6 border-t border-zinc-200 dark:border-zinc-800/50 mt-10">
        <TechnicalAnalysisView technicals={marketData?.technicals} market={market} />
        <AssetValuationTerminal valuation={marketData?.valuation} market={market} />
      </div>
    </div>
  );
}
