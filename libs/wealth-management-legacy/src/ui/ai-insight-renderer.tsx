"use client";

import {
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  PiggyBank,
  Lightbulb,
  Target,
  Shield,
  BarChart3,
  Wallet,
  Clock,
  Zap,
  Star,
  Sparkles,
  CheckCircle2,
  type LucideIcon,
} from "lucide-react";
import type { StructuredInsight, InsightIconHint, InsightSeverity } from "../ai/core/types";

const ICON_MAP: Record<InsightIconHint, LucideIcon> = {
  alert: AlertTriangle,
  "trend-up": TrendingUp,
  "trend-down": TrendingDown,
  savings: PiggyBank,
  tip: Lightbulb,
  target: Target,
  shield: Shield,
  chart: BarChart3,
  wallet: Wallet,
  clock: Clock,
  zap: Zap,
  star: Star,
};

const SEVERITY_STYLES: Record<
  InsightSeverity,
  {
    bg: string;
    border: string;
    iconBg: string;
    iconColor: string;
    titleColor: string;
    badgeBg: string;
    badgeText: string;
  }
> = {
  info: {
    bg: "bg-blue-50/50 dark:bg-blue-950/20",
    border: "border-blue-100/50 dark:border-blue-900/30",
    iconBg: "bg-blue-500/10",
    iconColor: "text-blue-500",
    titleColor: "text-blue-900 dark:text-blue-200",
    badgeBg: "bg-blue-500/10",
    badgeText: "text-blue-600 dark:text-blue-400",
  },
  warning: {
    bg: "bg-amber-50/50 dark:bg-amber-950/20",
    border: "border-amber-100/50 dark:border-amber-900/30",
    iconBg: "bg-amber-500/10",
    iconColor: "text-amber-500",
    titleColor: "text-amber-900 dark:text-amber-200",
    badgeBg: "bg-amber-500/10",
    badgeText: "text-amber-600 dark:text-amber-400",
  },
  success: {
    bg: "bg-emerald-50/50 dark:bg-emerald-950/20",
    border: "border-emerald-100/50 dark:border-emerald-900/30",
    iconBg: "bg-emerald-500/10",
    iconColor: "text-emerald-500",
    titleColor: "text-emerald-900 dark:text-emerald-200",
    badgeBg: "bg-emerald-500/10",
    badgeText: "text-emerald-600 dark:text-emerald-400",
  },
  critical: {
    bg: "bg-rose-50/50 dark:bg-rose-950/20",
    border: "border-rose-100/50 dark:border-rose-900/30",
    iconBg: "bg-rose-500/10",
    iconColor: "text-rose-500",
    titleColor: "text-rose-900 dark:text-rose-200",
    badgeBg: "bg-rose-500/10",
    badgeText: "text-rose-600 dark:text-rose-400",
  },
};

const SEVERITY_LABELS: Record<InsightSeverity, string> = {
  info: "Info",
  warning: "Attention",
  success: "On Track",
  critical: "Critical",
};

function InsightSectionCard({
  section,
}: {
  section: StructuredInsight["sections"][number];
}) {
  const style = SEVERITY_STYLES[section.severity] || SEVERITY_STYLES.info;
  const Icon = ICON_MAP[section.icon] || Lightbulb;

  return (
    <div
      className={`flex items-start gap-3 p-3.5 rounded-xl ${style.bg} border ${style.border} shadow-sm transition-all hover:shadow-md group`}
    >
      <div className={`p-2 ${style.iconBg} rounded-lg shrink-0 transition-transform group-hover:scale-110`}>
        <Icon className={`h-4 w-4 ${style.iconColor}`} />
      </div>
      <div className="space-y-1.5 min-w-0 flex-1">
        <div className="flex items-center gap-2 flex-wrap">
          <p className={`text-xs font-bold ${style.titleColor} uppercase tracking-wider`}>
            {section.title}
          </p>
          <span
            className={`text-[9px] font-bold ${style.badgeBg} ${style.badgeText} px-1.5 py-0.5 rounded-full uppercase tracking-widest`}
          >
            {SEVERITY_LABELS[section.severity]}
          </span>
        </div>
        <p className="text-xs leading-relaxed text-foreground/80 font-medium">
          {section.content}
        </p>
      </div>
    </div>
  );
}

function ActionItemsList({ items }: { items: string[] }) {
  return (
    <div className="mt-1 pt-3 border-t border-primary/10">
      <div className="flex items-center gap-2 mb-2.5">
        <Zap className="h-3.5 w-3.5 text-indigo-500" />
        <p className="text-[10px] font-bold text-indigo-600 dark:text-indigo-400 uppercase tracking-widest">
          Action Items
        </p>
      </div>
      <div className="space-y-1.5">
        {items.map((item, i) => (
          <div key={i} className="flex items-start gap-2 group">
            <CheckCircle2 className="h-3.5 w-3.5 text-indigo-400/60 shrink-0 mt-0.5 group-hover:text-indigo-500 transition-colors" />
            <p className="text-[11px] leading-relaxed text-muted-foreground font-medium">
              {item}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

export interface AIInsightRendererProps {
  insight: StructuredInsight;
  className?: string;
}

export function AIInsightRenderer({ insight, className }: AIInsightRendererProps) {
  return (
    <div className={className}>
      {/* Summary Banner */}
      <div className="flex items-center gap-2.5 mb-4 px-1">
        <div className="p-1 bg-indigo-500/10 rounded-md">
          <Sparkles className="h-3.5 w-3.5 text-indigo-500" />
        </div>
        <p className="text-sm font-semibold text-foreground/90 leading-snug">
          {insight.summary}
        </p>
      </div>

      {/* Sections Grid */}
      <div className="grid grid-cols-1 gap-2.5">
        {insight.sections.map((section, i) => (
          <InsightSectionCard key={i} section={section} />
        ))}
      </div>

      {/* Action Items */}
      {insight.actionItems && insight.actionItems.length > 0 && (
        <ActionItemsList items={insight.actionItems} />
      )}
    </div>
  );
}
