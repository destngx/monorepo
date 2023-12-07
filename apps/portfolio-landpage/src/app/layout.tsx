import './globals.css';
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Analytics } from '@vercel/analytics/react';
import { Footer, Navbar } from '@portfolio-landpage/components';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'Dest Nguyxn Landpage',
  description: 'Nguyen Pham Quang Dinh portfolio landpage',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className={inter.className + ' flex'}>
        <Navbar />
        <div className=" h-screen w-full bg-black pl-0 p-20">
          {children}
          <Footer />
        </div>
      </body>
      <Analytics />
    </html>
  );
}
