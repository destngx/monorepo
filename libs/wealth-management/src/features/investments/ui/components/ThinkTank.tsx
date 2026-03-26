import { Card, CardHeader, CardTitle, CardDescription, CardContent, Button } from '@wealth-management/ui';
import { Terminal, Sparkles, RefreshCcw, Globe, User, Bot, Send, Database } from 'lucide-react';
import { hasContent, MessageContent } from '@wealth-management/features/chat/ui';

interface ThinkTankProps {
  messages: any[];
  isAnalyzing: boolean;
  isChatBusy: boolean;
  isLoadingAccounts: boolean;
  onInitiateScan: () => void;
  error: string | null;
  inputContext: any;
  input: string;
  onInputChange: (val: string) => void;
  onFormSubmit: (e: React.FormEvent) => void;
}

export function ThinkTank({ 
  messages, 
  isAnalyzing, 
  isChatBusy,
  isLoadingAccounts, 
  onInitiateScan, 
  error, 
  inputContext,
  input,
  onInputChange,
  onFormSubmit
}: ThinkTankProps) {
  return (
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
          onClick={onInitiateScan}
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

          {inputContext && (
            <div className="mb-6 p-4 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-300">
              <h3 className="text-zinc-400 font-bold mb-3 flex items-center gap-2">
                <Database className="h-4 w-4 text-emerald-500" /> AI Input Data Context
              </h3>
              <details className="cursor-pointer group">
                <summary className="text-indigo-400 group-hover:text-indigo-300 font-medium select-none">
                  Show Raw AI Input Context (JSON & Search Data)
                </summary>
                <pre className="mt-3 p-3 bg-zinc-950 rounded overflow-x-auto text-[10px] text-zinc-500 whitespace-pre-wrap border border-zinc-800/50">
                  {JSON.stringify(inputContext, null, 2)}
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
                  {(m).name && (
                    <div className="text-xs font-bold text-emerald-400 mb-1">{(m).name}</div>
                  )}
                  <MessageContent message={m} />
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
          <form onSubmit={onFormSubmit} className="flex gap-2 relative">
            <input
              value={input}
              onChange={(e) => onInputChange(e.target.value)}
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
  );
}
