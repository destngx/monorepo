"use client";

import * as React from "react";
import { Sparkles } from "lucide-react";
import { useAISettings } from "@/hooks/use-ai-settings";
import { AI_MODELS } from "@wealth-management/ai";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "-management/ui/select";

export function ModelSwitcher() {
  const { settings, updateSettings, mounted } = useAISettings();

  if (!mounted) {
    return (
      <div className="h-4 w-24 bg-muted animate-pulse rounded" />
    );
  }

  const activeModel = AI_MODELS[settings.modelId] || AI_MODELS["gpt-4o-mini"];

  return (
    <Select
      value={settings.modelId}
      onValueChange={(value) => updateSettings({ modelId: value })}
    >
      <SelectTrigger 
        className="h-7 border-none bg-transparent hover:bg-muted/50 focus:ring-0 focus:ring-offset-0 text-left shadow-none group px-2 gap-1.5 transition-colors"
      >
        <Sparkles className="h-3 w-3 text-amber-500 shrink-0" />
        <SelectValue>
          <span className="text-xs text-muted-foreground group-hover:text-primary transition-colors truncate max-w-[100px]">
            {activeModel.label}
          </span>
        </SelectValue>
      </SelectTrigger>
      <SelectContent align="end" className="w-[240px]">
        {Object.entries(AI_MODELS).map(([id, config]) => (
          <SelectItem key={id} value={id} className="cursor-pointer py-2.5">
            <div className="flex flex-col gap-0.5 w-full">
              <span className="font-medium text-xs">{config.label}</span>
              <p className="text-[10px] text-muted-foreground leading-tight italic pr-2 whitespace-normal">
                {config.description}
              </p>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
