"use client";

import React, { createContext, useContext, useState, useCallback, useMemo } from "react";

export interface AIInsight {
  id: string;
  type: "insight" | "alert" | "suggestion";
  title: string;
  content: string;
  timestamp: Date;
  metadata?: Record<string, unknown>;
  suggestedQuestions?: string[];
}

interface AIContextType {
  pageData: Record<string, unknown>;
  insights: AIInsight[];
  setPageData: (data: Record<string, unknown>) => void;
  addInsight: (insight: Omit<AIInsight, "id" | "timestamp">) => void;
  clearInsights: () => void;
  removeInsight: (id: string) => void;
}

const AIContext = createContext<AIContextType | undefined>(undefined);

export function AIContextProvider({ children }: { children: React.ReactNode }) {
  const [pageData, setPageDataState] = useState<Record<string, unknown>>({});
  const [insights, setInsights] = useState<AIInsight[]>([]);

  const setPageData = useCallback((data: Record<string, unknown>) => {
    setPageDataState(prev => ({ ...prev, ...data }));
  }, []);

  const addInsight = useCallback((insight: Omit<AIInsight, "id" | "timestamp">) => {
    const newInsight: AIInsight = {
      ...insight,
      id: Math.random().toString(36).substring(2, 9),
      timestamp: new Date(),
    };
    setInsights(prev => {
      // Avoid duplicates by title/content
      if (prev.some(i => i.title === insight.title && i.content === insight.content)) {
        return prev;
      }
      return [...prev, newInsight];
    });
  }, []);

  const removeInsight = useCallback((id: string) => {
    setInsights(prev => prev.filter(i => i.id !== id));
  }, []);

  const clearInsights = useCallback(() => {
    setInsights([]);
    setPageDataState({});
  }, []);

  const value = useMemo(() => ({
    pageData,
    insights,
    setPageData,
    addInsight,
    clearInsights,
    removeInsight
  }), [pageData, insights, setPageData, addInsight, clearInsights, removeInsight]);

  return <AIContext.Provider value={value}>{children}</AIContext.Provider>;
}

export function useAIContext() {
  const context = useContext(AIContext);
  if (context === undefined) {
    throw new Error("useAIContext must be used within an AIContextProvider");
  }
  return context;
}
