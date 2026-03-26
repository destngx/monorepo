'use client';

import { Send } from 'lucide-react';
import { Button } from '@wealth-management/ui';

interface Props {
  input: string;
  setInput: (val: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  isBusy: boolean;
}

export function ChatInputForm({ input, setInput, onSubmit, isBusy }: Props) {
  return (
    <div className="p-6 border-t bg-card">
      <form onSubmit={onSubmit} className="flex w-full items-end gap-3 max-w-3xl mx-auto">
        <div className="flex-1 relative">
          <textarea
            value={input}
            onChange={(e) => {
              const target = e.target;
              setInput(target.value);
              target.style.height = 'inherit';
              target.style.height = `${Math.min(target.scrollHeight, 200)}px`;
            }}
            placeholder="Message WealthOS AI..."
            className="w-full rounded-2xl border border-input bg-muted/30 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none min-h-[48px] max-h-[200px]"
            disabled={isBusy}
            rows={1}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                onSubmit(e as any);
              }
            }}
          />
        </div>
        <Button
          type="submit"
          size="icon"
          className="h-11 w-11 rounded-full shadow-lg transition-transform active:scale-95 bg-primary text-primary-foreground hover:opacity-90"
          disabled={isBusy || !input.trim()}
        >
          <Send className="h-5 w-5" />
        </Button>
      </form>
      <p className="text-[10px] text-center text-muted-foreground mt-3 uppercase tracking-tighter font-medium px-4">
        Analysis computed on live Google Sheet data — results may vary
      </p>
    </div>
  );
}
