"use client";

import { useRef, useEffect, useState, useMemo, useCallback } from "react";
import { useChat } from "@ai-sdk/react";
import { usePathname } from "next/navigation";
import { useAISettings } from "@/hooks/use-ai-settings";
import { AI_MODELS } from "@wealth-management/ai";
import { useDebouncedChatPersistence } from "@/hooks/use-debounced-chat-persistence";
import { AIFab } from "./ai-fab";
import { AIDrawer } from "./ai-drawer";
import { AIInsightCard } from "./ai-insight-card";
import { useAIContext, AIInsight } from "./ai-context-provider";
import { ChatMessage } from "../model/types";



export function AIChatWidget() {
  const [open, setOpen] = useState(false);
  const [input, setInput] = useState("");
  const [showInsightCard, setShowInsightCard] = useState(false);
  const [aiSuggestions, setAiSuggestions] = useState<Array<{ label: string; prompt: string }>>([]);
  const [isBusySuggestions, setIsBusySuggestions] = useState(false);
  
  const pathname = usePathname();
  const { settings } = useAISettings();
  const { insights, pageData, removeInsight } = useAIContext();

  const { messages, setMessages, sendMessage, status } = useChat({
    api: "/api/chat",
    body: { 
      modelId: settings.modelId,
      context: {
        pathname,
        pageData,
        activeInsights: insights
      }
    },
    maxSteps: 100,
  } as any);

  // Fetch AI suggestions based on current context
  const fetchSuggestions = useCallback(async () => {
    setIsBusySuggestions(true);
    
    // Quick scrape of the page for more context
    const pageTitle = document.querySelector('h1')?.innerText || '';
    const sectionHeaders = Array.from(document.querySelectorAll('h2, h3'))
      .map(h => (h as HTMLElement).innerText)
      .slice(0, 5);
    
    try {
      const response = await fetch("/api/chat/suggestions", {
        method: "POST",
        body: JSON.stringify({
          modelId: settings.modelId,
          context: {
            pathname,
            pageTitle,
            sectionHeaders,
            pageData,
            activeInsights: insights
          }
        })
      });
      const data = (await response.json()) as { suggestions: Array<{ label: string; prompt: string }> };
      if (data.suggestions) {
        setAiSuggestions(data.suggestions);
      }
    } catch (err) {
      console.error("Failed to fetch suggestions:", err);
    } finally {
      setIsBusySuggestions(false);
    }
  }, [pathname, pageData, insights, settings.modelId]);

  // Trigger fetch when opening OR context changes significantly
  useEffect(() => {
    if (open && messages.length === 0) {
      void fetchSuggestions();
    }
  }, [open, pathname, insights.length, fetchSuggestions, messages.length]);

  // Load messages from localStorage on mount
  useEffect(() => {
    const savedMessages = localStorage.getItem("wealthos-chat-history");
    if (savedMessages) {
      try {
        setMessages(JSON.parse(savedMessages));
      } catch (e) {
        console.error("Failed to load chat history:", e);
      }
    }
  }, [setMessages]);

  useDebouncedChatPersistence(messages, status);

  const isBusy = status === "streaming" || status === "submitted";

  const currentContext = useMemo(() => {
    const now = new Date();
    const monthYear = now.toLocaleString("en-US", { month: "long", year: "numeric" });
    
    if (pathname.includes("budget")) return `Budget · ${monthYear}`;
    if (pathname.includes("goals")) return "Goals & Savings";
    if (pathname.includes("market") || pathname.includes("trade")) return "Market Pulse";
    if (pathname.includes("accounts")) return "Accounts Overview";
    if (pathname.includes("credit-cards")) return "Card Strategy";
    if (pathname.includes("loans")) return "Loan Analysis";
    if (pathname.includes("transactions")) return "Spending History";
    
    return "Wealth Overview";
  }, [pathname]);

  const activePrompts = useMemo(() => {
    // Priority 1: Insight-specific follow-ups
    const insightPrompts = insights.flatMap(insight => 
      (insight.suggestedQuestions || []).map(q => ({
        label: q.length > 20 ? q.substring(0, 20) + "..." : q,
        prompt: q
      }))
    );
    
    // Priority 2: AI-generated dynamic suggestions
    return [...insightPrompts, ...aiSuggestions].slice(0, 6);
  }, [insights, aiSuggestions]);

  const hasInsight = insights.length > 0;

  const handleFabClick = () => {
    if (hasInsight) {
      setShowInsightCard(true);
    } else {
      setShowInsightCard(false);
    }
    setOpen(true);
  };

  const handleClearChat = () => {
    setMessages([]);
    localStorage.removeItem("wealthos-chat-history");
  };

  const handlePromptClick = (prompt: string) => {
    setShowInsightCard(false);
    void sendMessage({ text: prompt });
  };

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isBusy) return;
    setShowInsightCard(false);
    void sendMessage({ text: input });
    setInput("");
  };

  return (
    <>
      <AIFab 
        onClick={handleFabClick}
        isLoading={isBusy}
        hasInsight={hasInsight}
        isOpen={open}
      />

      <AIDrawer
        isOpen={open}
        onClose={() => setOpen(false)}
        messages={messages as unknown as ChatMessage[]}
        input={input}
        onInputChange={setInput}
        onSubmit={handleFormSubmit}
        onPromptClick={handlePromptClick}
        onRegenerateSuggestions={fetchSuggestions}
        onClearChat={handleClearChat}
        isBusy={isBusy}
        isBusySuggestions={isBusySuggestions}
        activeContext={currentContext}
        suggestedPrompts={activePrompts}
      >
        {showInsightCard && insights.length > 0 && (
          <div className="space-y-4 mb-6">
            {insights.map((insight) => (
              <AIInsightCard 
                key={insight.id}
                title={insight.title}
                time={insight.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                content={insight.content}
                onViewBreakdown={() => {
                  handlePromptClick(`Tell me more about: ${insight.title}. ${insight.content}`);
                  removeInsight(insight.id);
                }}
                onAskFollowUp={() => {
                  setShowInsightCard(false);
                  removeInsight(insight.id);
                }}
                className="animate-in slide-in-from-bottom-4 duration-500"
              />
            ))}
          </div>
        )}
      </AIDrawer>
    </>
  );
}
