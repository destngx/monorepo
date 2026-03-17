'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card } from '@/components/ui/card';
import { ArrowRight, ArrowLeft, Check, Sparkles, Calendar } from 'lucide-react';
import { cn } from '@wealth-management/utils';
import { GoalType } from '@wealth-management/types';

const CATEGORIES: { type: GoalType; emoji: string; label: string }[] = [
  { type: 'SAVINGS_TARGET', emoji: '🏦', label: 'Emergency Fund' },
  { type: 'PURCHASE_GOAL', emoji: '🎁', label: 'Purchase' },
  { type: 'DEBT_PAYOFF', emoji: '💸', label: 'Debt Payoff' },
  { type: 'INVESTMENT_TARGET', emoji: '📈', label: 'Investment' },
  { type: 'INCOME_GOAL', emoji: '💰', label: 'Income Goal' },
  { type: 'NET_WORTH_MILESTONE', emoji: '🏆', label: 'Net Worth' },
];

export function CreateGoalFlow() {
  const [step, setStep] = useState(1);
  const [data, setData] = useState({
    type: 'SAVINGS_TARGET' as GoalType,
    name: '',
    targetAmount: '',
    deadline: '',
    fundingSource: '',
    contributionType: 'MANUAL',
  });

  const updateData = (key: string, value: any) => setData((prev) => ({ ...prev, [key]: value }));
  const next = () => setStep((s) => s + 1);
  const prev = () => setStep((s) => s - 1);

  return (
    <div className="max-w-xl mx-auto py-12">
      <div className="mb-8 flex justify-center gap-2">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={cn('h-1.5 w-12 rounded-full', i <= step ? 'bg-indigo-600' : 'bg-muted')} />
        ))}
      </div>

      <div className="min-h-[400px]">
        {step === 1 && (
          <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4">
            <div className="space-y-2 text-center">
              <h2 className="text-2xl font-black uppercase tracking-tight">What's the goal?</h2>
              <p className="text-muted-foreground">Select a category that best fits your intention.</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              {CATEGORIES.map((c) => (
                <button
                  key={c.type}
                  onClick={() => {
                    updateData('type', c.type);
                    updateData('name', c.label);
                  }}
                  className={cn(
                    'p-6 rounded-2xl border-2 transition-all text-left space-y-3 group',
                    data.type === c.type ? 'border-indigo-600 bg-indigo-50/50' : 'border-muted hover:border-indigo-200',
                  )}
                >
                  <span className="text-3xl grayscale group-hover:grayscale-0 transition-all">{c.emoji}</span>
                  <p className="font-bold text-sm uppercase tracking-tight">{c.label}</p>
                </button>
              ))}
            </div>
            <div className="pt-4">
              <Input
                placeholder="Give it a custom name..."
                value={data.name}
                onChange={(e) => updateData('name', e.target.value)}
                className="h-12 border-none bg-muted/50 rounded-xl px-6 font-medium"
              />
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4">
            <div className="space-y-2 text-center">
              <h2 className="text-2xl font-black uppercase tracking-tight">How much & when?</h2>
              <p className="text-muted-foreground">Define your target and your deadline.</p>
            </div>
            <div className="space-y-6">
              <div className="space-y-2">
                <label className="text-xs font-bold uppercase tracking-widest text-muted-foreground ml-1">
                  Target Amount (₫)
                </label>
                <Input
                  type="number"
                  placeholder="0"
                  value={data.targetAmount}
                  onChange={(e) => updateData('targetAmount', e.target.value)}
                  className="h-20 text-4xl font-black text-center border-none bg-muted/30 rounded-3xl"
                />
              </div>
              <div className="space-y-2 text-center p-4 bg-indigo-500/5 rounded-2xl border border-indigo-500/10">
                <div className="flex items-center justify-center gap-2 text-indigo-500 mb-1">
                  <Sparkles size={16} />
                  <span className="text-xs font-bold uppercase tracking-tighter italic">AI Advice</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Based on your savings history, a target of{' '}
                  <span className="text-foreground font-bold italic">₫50M</span> within 12 months is highly achievable.
                </p>
              </div>
              <div className="space-y-2">
                <label className="text-xs font-bold uppercase tracking-widest text-muted-foreground ml-1">
                  Deadline Date
                </label>
                <div className="relative">
                  <Calendar className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
                  <Input
                    type="date"
                    value={data.deadline}
                    onChange={(e) => updateData('deadline', e.target.value)}
                    className="h-14 pl-12 border-none bg-muted/30 rounded-2xl font-bold"
                  />
                </div>
              </div>
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4">
            <div className="space-y-2 text-center">
              <h2 className="text-2xl font-black uppercase tracking-tight">How will you fund it?</h2>
              <p className="text-muted-foreground">Linked source accounts and contribution type.</p>
            </div>
            <div className="space-y-4">
              {['MANUAL', 'AUTOMATIC', 'AI-MANAGED'].map((type) => (
                <button
                  key={type}
                  onClick={() => updateData('contributionType', type)}
                  className={cn(
                    'w-full p-6 rounded-2xl border-2 transition-all flex items-center justify-between text-left',
                    data.contributionType === type
                      ? 'border-indigo-600 bg-indigo-50/50'
                      : 'border-muted hover:border-indigo-200',
                  )}
                >
                  <div>
                    <p className="font-bold text-sm uppercase tracking-tight">{type.replace('-', ' ')}</p>
                    <p className="text-xs text-muted-foreground">
                      {type === 'AI-MANAGED' ? 'AI sweeps surplus cash automatically' : 'Fixed schedule or manual add'}
                    </p>
                  </div>
                  {data.contributionType === type && (
                    <div className="h-6 w-6 rounded-full bg-indigo-600 flex items-center justify-center text-white">
                      <Check size={14} strokeWidth={4} />
                    </div>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}

        {step === 4 && (
          <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 text-center">
            <div className="flex justify-center flex-col items-center gap-4">
              <div className="h-20 w-20 rounded-full bg-emerald-500/10 flex items-center justify-center text-emerald-500">
                <Check size={40} strokeWidth={3} />
              </div>
              <div className="space-y-2">
                <h2 className="text-3xl font-black italic uppercase tracking-tighter">Gold Standard!</h2>
                <p className="text-muted-foreground">
                  Your <span className="text-foreground font-bold italic">{data.name}</span> goal is ready to launch.
                </p>
              </div>
            </div>

            <Card className="border-none bg-indigo-600 text-white p-8 rounded-3xl overflow-hidden relative shadow-2xl shadow-indigo-600/30">
              <div className="absolute top-0 right-0 p-4 opacity-10">
                <Sparkles size={100} />
              </div>
              <div className="relative space-y-4">
                <p className="text-xs font-bold uppercase tracking-[0.2em] opacity-70">Projected Achievement</p>
                <p className="text-4xl font-black italic">December 2026</p>
                <div className="h-px w-full bg-white/20 my-4" />
                <div className="flex justify-between text-[10px] font-bold uppercase tracking-widest opacity-70">
                  <span>Monthly Contribution</span>
                  <span>Target Date Success</span>
                </div>
                <div className="flex justify-between font-black italic text-lg">
                  <span>₫2.5M/Month</span>
                  <span>94% PROBABILITY</span>
                </div>
              </div>
            </Card>

            <p className="text-xs text-muted-foreground px-12">
              We&apos;ll remind you if you fall behind and look for opportunities to accelerate your progress.
            </p>
          </div>
        )}
      </div>

      <div className="mt-12 flex justify-between gap-4">
        {step > 1 && step < 4 && (
          <Button
            variant="ghost"
            className="h-14 px-8 rounded-2xl font-bold uppercase tracking-widest text-xs gap-2"
            onClick={prev}
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
        )}
        {step < 4 ? (
          <Button
            className="h-14 px-8 rounded-2xl bg-indigo-600 hover:bg-indigo-700 text-white ml-auto font-black uppercase tracking-widest text-xs gap-2"
            onClick={next}
          >
            Continue
            <ArrowRight className="h-4 w-4" />
          </Button>
        ) : (
          <Button
            className="h-16 w-full rounded-2xl bg-indigo-600 hover:bg-indigo-700 text-white font-black uppercase tracking-widest text-lg shadow-xl shadow-indigo-600/20"
            onClick={() => (window.location.href = '/accounts/goals')}
          >
            Launch Goal
          </Button>
        )}
      </div>
    </div>
  );
}
