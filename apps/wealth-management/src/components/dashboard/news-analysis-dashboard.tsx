'use client';

import { useState, useEffect } from 'react';
import useSWR from 'swr';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip as ReTooltip, 
  ResponsiveContainer, 
  Cell 
} from 'recharts';
import { 
  Newspaper, 
  TrendingUp, 
  TrendingDown, 
  Minus, 
  RefreshCcw, 
  Search, 
  ExternalLink, 
  Clock, 
  ShieldCheck,
  Sparkles,
  ChevronDown,
  ChevronUp
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { GLASS_CARD, TERMINAL_FONT } from './market-pulse-dashboard';

const fetcher = (url: string, body: any) => 
  fetch(url, { 
    method: 'POST', 
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body) 
  }).then((res) => res.json());

interface NewsItem {
  title: string;
  source: string;
  url: string;
  description: string;
  sentiment: 'Positive' | 'Negative' | 'Neutral';
  confidence: number;
  impactScore: number;
  summary: string;
}

interface NewsAnalysisReport {
  topic: string;
  overallSentiment: 'BULLISH' | 'BEARISH' | 'NEUTRAL';
  counts: {
    positive: number;
    negative: number;
    neutral: number;
  };
  articles: NewsItem[];
  timestamp: string;
}

export function NewsAnalysisDashboard({ 
    topic, 
    onAnalyzed 
}: { 
    topic: string, 
    onAnalyzed?: (data: NewsAnalysisReport) => void 
}) {
  const [showArticles, setShowArticles] = useState(false);
  const { data, error, isLoading, mutate } = useSWR<NewsAnalysisReport>(
    ['/api/news/analyze', topic],
    ([url, t]) => fetcher(url, { topic: t }),
    { revalidateOnFocus: false }
  );

  useEffect(() => {
    if (data && onAnalyzed) onAnalyzed(data);
  }, [data, onAnalyzed]);

  if (error || (data as any)?.error) {
    return (
       <Card className={`${GLASS_CARD} border-rose-500/20 shadow-rose-500/10`}>
          <CardContent className="pt-6 pb-6 text-center space-y-3">
             <div className="p-3 rounded-full bg-rose-500/10 text-rose-500 w-fit mx-auto">
                <Minus className="w-5 h-5" />
             </div>
             <div className="space-y-1">
                <h4 className="text-sm font-bold text-zinc-200 uppercase tracking-tight">Analysis Unavailable</h4>
                <p className="text-[10px] text-zinc-500 font-mono italic">
                   {error?.message || (data as any)?.error || 'The news intelligence engine encountered an unexpected error.'}
                </p>
             </div>
             <Button 
                variant="outline" 
                size="sm" 
                onClick={() => mutate()} 
                className="mt-2 h-8 border-rose-500/30 text-rose-500 hover:bg-rose-500/5 text-[10px] uppercase font-bold tracking-widest"
              >
                Retry Analysis
             </Button>
          </CardContent>
       </Card>
    );
  }

  const chartData = data?.counts ? [
    { name: 'Positive', count: data.counts.positive || 0, color: '#10b981' },
    { name: 'Negative', count: data.counts.negative || 0, color: '#f43f5e' },
    { name: 'Neutral', count: data.counts.neutral || 0, color: '#f59e0b' },
  ] : [
    { name: 'Positive', count: 0, color: '#10b981' },
    { name: 'Negative', count: 0, color: '#f43f5e' },
    { name: 'Neutral', count: 0, color: '#f59e0b' },
  ];

  const sentimentIcon = data?.overallSentiment === 'BULLISH' ? <TrendingUp className="w-4 h-4" /> : 
                       data?.overallSentiment === 'BEARISH' ? <TrendingDown className="w-4 h-4" /> : 
                       <Minus className="w-4 h-4" />;

  const sentimentColor = data?.overallSentiment === 'BULLISH' ? 'text-emerald-500' :
                        data?.overallSentiment === 'BEARISH' ? 'text-rose-500' :
                        'text-amber-500';

  const sentimentBg = data?.overallSentiment === 'BULLISH' ? 'bg-emerald-500/10 border-emerald-500/20' :
                     data?.overallSentiment === 'BEARISH' ? 'bg-rose-500/10 border-rose-500/20' :
                     'bg-amber-500/10 border-amber-500/20';

  return (
    <Card className={`${GLASS_CARD} border-indigo-500/20 shadow-indigo-500/10 group`}>
      <CardHeader className="pb-4 border-b border-zinc-200 dark:border-zinc-800 space-y-3">
        <div className="flex items-center justify-between">
           <CardTitle className="text-lg font-black tracking-tighter uppercase flex items-center gap-2">
              <span className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
                <Newspaper className="w-4 h-4" />
              </span>
              {topic} SENTIMENT
           </CardTitle>
           <Button 
             variant="ghost" 
             size="sm" 
             onClick={() => mutate()} 
             disabled={isLoading}
             className="h-8 w-8 p-0 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
             <RefreshCcw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
           </Button>
        </div>

        {data && (
          <div className={`p-2 rounded-xl border flex items-center justify-center gap-2 font-black text-xs tracking-widest ${sentimentBg} ${sentimentColor}`}>
             {sentimentIcon} {data.overallSentiment} BIAS
          </div>
        )}
      </CardHeader>

      <CardContent className="pt-6 space-y-6">
         <div className="grid grid-cols-3 gap-3">
            {[
              { label: 'Positive', count: data?.counts?.positive, color: 'text-emerald-500', bg: 'bg-emerald-500/5' },
              { label: 'Negative', count: data?.counts?.negative, color: 'text-rose-500', bg: 'bg-rose-500/5' },
              { label: 'Neutral', count: data?.counts?.neutral, color: 'text-amber-500', bg: 'bg-amber-500/5' },
            ].map((stat, idx) => (
              <div key={idx} className={`${stat.bg} p-3 rounded-2xl flex flex-col items-center justify-center border border-zinc-200 dark:border-zinc-800/50`}>
                 <span className={`text-xl font-black ${TERMINAL_FONT} ${stat.color}`}>
                   {isLoading ? (
                     <div className="h-6 w-8 bg-zinc-200 dark:bg-zinc-800 animate-pulse rounded" />
                   ) : (
                     stat.count ?? 0
                   )}
                 </span>
                 <span className="text-[9px] uppercase font-bold text-zinc-500 tracking-tighter">{stat.label}</span>
              </div>
            ))}
         </div>

         {/* Mini Chart visualization */}
         <div className="h-[120px] w-full min-w-0">
            <ResponsiveContainer width="100%" height="100%">
               <BarChart data={chartData}>
                  <CartesianGrid strokeDasharray="2 2" vertical={false} stroke="#27272a" opacity={0.3} />
                  <XAxis dataKey="name" hide />
                  <YAxis hide />
                  <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                    {chartData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} fillOpacity={0.8} />
                    ))}
                  </Bar>
                  <ReTooltip 
                    cursor={false}
                    contentStyle={{ backgroundColor: '#09090b', border: 'none', borderRadius: '8px', fontSize: '10px' }}
                  />
               </BarChart>
            </ResponsiveContainer>
         </div>

         <div className="flex flex-col items-center gap-4">
            <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
               {isLoading ? 'Scanning sources...' : `Analyzed ${data?.articles?.length ?? 0} Global Source Feeds`}
            </p>
            
            <Button 
                variant="outline" 
                size="sm" 
                onClick={() => setShowArticles(!showArticles)}
                className="w-full h-10 border-indigo-500/20 hover:bg-indigo-500/5 text-zinc-600 dark:text-zinc-400 font-bold uppercase text-[10px] tracking-widest gap-2 rounded-xl"
            >
                {showArticles ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                {showArticles ? 'Hide AI Analysis Details' : 'View Detailed Article Insights'}
            </Button>
         </div>

         {showArticles && data && (
            <div className="space-y-3 pt-4 border-t border-zinc-200 dark:border-zinc-800 animate-in slide-in-from-top-2 duration-300">
               {data.articles.map((article, idx) => (
                 <Card key={idx} className="bg-zinc-50/50 dark:bg-zinc-900/30 border-zinc-200 dark:border-zinc-800/50 hover:border-indigo-500/30 transition-all p-4 space-y-3">
                    <div className="flex items-start justify-between gap-3">
                       <div className="space-y-1">
                          <a 
                             href={article.url} 
                             target="_blank" 
                             rel="noopener noreferrer"
                             className="text-xs font-bold leading-tight line-clamp-2 hover:underline hover:text-indigo-500 transition-colors"
                          >
                             {article.title}
                          </a>
                          <p className="text-[10px] text-zinc-500 flex items-center gap-1.5 font-mono">
                             <a 
                               href={article.url} 
                               target="_blank" 
                               rel="noopener noreferrer"
                               className="hover:text-indigo-400 hover:underline"
                             >
                               {article.source}
                             </a>
                             • <Clock className="w-3 h-3" /> {new Date().toLocaleDateString()}
                          </p>
                       </div>
                       <Badge className={`${
                         article.sentiment === 'Positive' ? 'bg-emerald-500/10 text-emerald-500' :
                         article.sentiment === 'Negative' ? 'bg-rose-500/10 text-rose-500' :
                         'bg-amber-500/10 text-amber-500'
                       } border-none text-[9px] font-black tracking-widest px-2 py-0.5`}>
                         {article.sentiment}
                       </Badge>
                    </div>
                    
                    <p className="text-[11px] text-zinc-600 dark:text-zinc-400 leading-relaxed font-medium bg-white/50 dark:bg-zinc-950/40 p-2 rounded-lg border border-zinc-100 dark:border-zinc-800/30">
                       <Sparkles className="w-3 h-3 text-emerald-500 inline mr-1.5 mb-0.5" />
                       {article.summary}
                    </p>

                    <div className="flex items-center justify-between">
                       <div className="flex items-center gap-3">
                          <div className="flex flex-col">
                             <span className="text-[8px] uppercase font-bold text-zinc-500 tracking-tighter">Confidence</span>
                             <span className="text-[10px] font-black font-mono text-indigo-500">{article.confidence}%</span>
                          </div>
                          <div className="flex flex-col">
                             <span className="text-[8px] uppercase font-bold text-zinc-500 tracking-tighter">Market Impact</span>
                             <span className="text-[10px] font-black font-mono text-emerald-500">{article.impactScore}/100</span>
                          </div>
                       </div>
                       <a 
                         href={article.url} 
                         target="_blank" 
                         rel="noopener noreferrer"
                         className="p-1.5 rounded-lg hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-400 hover:text-indigo-500 transition-colors"
                        >
                         <ExternalLink className="w-3.5 h-3.5" />
                       </a>
                    </div>
                 </Card>
               ))}
            </div>
         )}
      </CardContent>
    </Card>
  );
}
