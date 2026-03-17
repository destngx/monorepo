"use client";

import React from "react";
import { Bot, Sparkles } from "lucide-react";
import { ChatMessages } from "./chat-messages";
import { ChatInput } from "./chat-input";
import { useAISettings } from "@/hooks/use-ai-settings";
import { useChatMessages, useChatStream, getActiveModelLabel } from "../model";

export function ChatContainer() {
  const { messages, mounted, addMessage, updateLastMessage } = useChatMessages();
  const { isLoading, streamMessage, cancel } = useChatStream();
  const { settings } = useAISettings();

  const handleSendMessage = React.useCallback(
    async (text: string) => {
      // Add user message
      addMessage("user", text);

      try {
        // Add assistant message placeholder
        const assistantMsg = addMessage("assistant", "");

        // Stream the response
        await streamMessage([...messages, assistantMsg], settings.modelId, (content) => {
          updateLastMessage(content);
        });
      } catch (error: any) {
        if (error.name === "AbortError") {
          console.log("Request cancelled");
          return;
        }

        console.error("Failed to get response:", error);

        // Add error message
        addMessage(
          "assistant",
          `Sorry, I encountered an error: ${error.message || "Unknown error"}. Please try again.`
        );
      }
    },
    [messages, settings.modelId, addMessage, updateLastMessage, streamMessage]
  );

  if (!mounted) {
    return null; // Prevent hydration mismatch
  }

  const activeModelLabel = getActiveModelLabel(settings.modelId);

  return (
    <div className="flex flex-col h-full rounded-xl border bg-card shadow-lg overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b bg-gradient-to-r from-primary/5 to-transparent">
        <div className="flex items-center gap-3">
          <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center">
            <Bot className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-sm font-semibold">WealthOS AI Advisor</h2>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Sparkles className="h-3 w-3 text-amber-500" />
              {activeModelLabel}
            </p>
          </div>
        </div>
      </div>

      {/* Messages */}
      <ChatMessages messages={messages} isLoading={isLoading} />

      {/* Input */}
      <div className="flex-shrink-0 p-6 border-t bg-card">
        <ChatInput
          onSubmit={handleSendMessage}
          isLoading={isLoading}
          placeholder="Ask about your finances..."
        />
      </div>
    </div>
  );
}
