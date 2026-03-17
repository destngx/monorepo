"use client";

import { ChatContainer } from "@/features/chat/ui";

export default function ChatPage() {
  return (
    <div className="flex flex-col h-[calc(100vh-8rem)] max-w-5xl mx-auto gap-6 px-4">
      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col relative min-w-0">
        <div className="mb-4">
          <h1 className="text-2xl font-bold tracking-tight">AI Wealth Advisor</h1>
          <p className="text-muted-foreground text-sm">
            Get real-time insights powered by your financial data
          </p>
        </div>
        <div className="flex-1 w-full relative">
          <ChatContainer />
        </div>
      </div>
    </div>
  );
}
