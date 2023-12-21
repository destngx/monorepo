import './globals.css';
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Analytics } from '@vercel/analytics/react';
import { AuroraBg, Footer, Navbar } from '@portfolio-landpage/components';
import Loading from './loading';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'Dest Nguyxn Landpage',
  description: 'Nguyen Pham Quang Dinh portfolio landpage',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className={inter.className + ' flex p-1'}>
        <Loading>
          {children}
          <Footer />
        </Loading>
      </body>
      <Analytics />
    </html>
  );
}
