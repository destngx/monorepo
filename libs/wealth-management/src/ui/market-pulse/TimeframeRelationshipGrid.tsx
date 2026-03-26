'use client';

import { ArrowRight } from 'lucide-react';
import { Card } from '../card';
import { Badge } from '../badge';
import { GLASS_CARD } from './constants';

export function TimeframeRelationshipGrid({ relationships, entryScore }: { relationships?: any[]; entryScore?: any }) {
  if (!relationships) return null;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3">
        {relationships.map((rel, idx) => (
          <div
            key={idx}
            className={`${GLASS_CARD} p-4 flex items-center justify-between group hover:bg-zinc-800/30 transition-all`}
          >
            <div className="flex items-center gap-4">
              <div className="flex flex-col">
                <div className="text-xs font-bold font-mono tracking-tighter text-white">
                  {rel.pair} <ArrowRight className="inline w-3 h-3 mx-1 text-zinc-600" />
                </div>
                <div className="text-[10px] text-zinc-500 uppercase font-bold tracking-widest">{rel.relationship}</div>
              </div>
              <div className="h-8 w-px bg-zinc-800/50 hidden md:block" />
              <div className="flex flex-col">
                <div className="text-xs font-medium text-zinc-400">{rel.advice}</div>
                <div className="text-[10px] text-zinc-600 italic font-mono">{rel.adviceVi}</div>
              </div>
            </div>
            <Badge
              variant="outline"
              className={`text-[9px] font-mono border-zinc-800 ${rel.status === 'STRONG' ? 'text-emerald-500 bg-emerald-500/10' : 'text-zinc-500'}`}
            >
              {rel.status}
            </Badge>
          </div>
        ))}
      </div>

      {entryScore && (
        <Card className={`${GLASS_CARD} p-5`}>
          <div className="text-[10px] font-bold uppercase tracking-widest text-zinc-500 mb-4">
            Entry Timing Score (15m)
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="flex flex-col items-center justify-center p-3 border-r border-zinc-800/30">
              <div className="text-[9px] font-bold text-zinc-500 uppercase mb-1">Overall</div>
              <div className="text-3xl font-black text-rose-500">{entryScore.overall}/10</div>
              <div className="text-[9px] font-bold text-rose-400 uppercase">Poor</div>
            </div>
            <div className="space-y-1">
              <div className="text-[9px] font-bold text-zinc-500 uppercase">Higher TF Support</div>
              <div className="text-xl font-bold">/{entryScore.higherTfSupport || 6}</div>
              <div className="text-[9px] text-zinc-600">Parent timeframes aligned</div>
            </div>
            <div className="space-y-1">
              <div className="text-[9px] font-bold text-zinc-500 uppercase">Lower TF Confirm</div>
              <div className="text-xl font-bold">/{entryScore.lowerTfConfirm || 4}</div>
              <div className="text-[9px] text-zinc-600">Child timeframes confirming</div>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
