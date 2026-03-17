import type { Metadata } from "next";
import { Inter } from "next/font/google"; // Changing to Inter for modern look
import "./globals.css";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { ThemeProvider } from "@/components/theme-provider";
import { AIChatWidget } from "@/components/chat/ai-chat-widget";

import { SidebarProvider } from "@/components/layout/sidebar-provider";
import { LayoutWrapper } from "@/components/layout/layout-wrapper";
import { MaskProvider } from "@/components/mask-provider";
import { AIContextProvider } from "@/components/chat/ai-context-provider";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "WealthOS - Personal Finance",
  description: "AI-powered wealth management platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.className} antialiased`}>
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <MaskProvider>
            <AIContextProvider>
              <SidebarProvider>
                <div className="flex min-h-screen bg-background">
                  <Sidebar />
                  <LayoutWrapper>
                    <Header />
                    <main className="flex-1 p-4 md:p-6 overflow-x-hidden">
                      {children}
                    </main>
                  </LayoutWrapper>
                  <AIChatWidget />
                </div>
              </SidebarProvider>
            </AIContextProvider>
          </MaskProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
