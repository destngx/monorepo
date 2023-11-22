import './globals.css';
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Analytics } from '@vercel/analytics/react';
import { AuthLayout, MainLayout } from '@cloudinary-photos-app/components/layout';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'DestNgX Photos',
  description: 'DestNgX Cloudinary Photos',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className={inter.className}>
        <AuthLayout>
          <MainLayout>{children}</MainLayout>
          <Analytics />
        </AuthLayout>
      </body>
    </html>
  );
}
