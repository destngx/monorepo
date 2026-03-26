import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { 
  Sidebar, 
  Header, 
  ThemeProvider, 
  SidebarProvider, 
  LayoutWrapper, 
  MaskProvider,
  ErrorNotificationsProvider,
  ErrorNotificationDisplay
} from '@wealth-management/ui';
import { AIChatWidget, AIContextProvider } from '@wealth-management/features/chat/ui';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'WealthOS - Personal Finance',
  description: 'AI-powered wealth management platform',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.className} antialiased`}>
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
          <ErrorNotificationsProvider>
            <MaskProvider>
              <AIContextProvider>
                <SidebarProvider>
                  <div className="flex min-h-screen bg-background text-foreground transition-colors duration-300">
                    <Sidebar />
                    <LayoutWrapper>
                      <Header />
                      <main className="flex-1 p-4 md:p-6 overflow-x-hidden">{children}</main>
                    </LayoutWrapper>
                    <AIChatWidget />
                  </div>
                </SidebarProvider>
              </AIContextProvider>
            </MaskProvider>
            <ErrorNotificationDisplay />
          </ErrorNotificationsProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
