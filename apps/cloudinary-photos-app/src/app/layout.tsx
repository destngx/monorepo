import './globals.css';
import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { Analytics } from '@vercel/analytics/react';
import { AuthLayout, MainLayout } from '@cloudinary-photos-app/components/layout';
import cloudinary from 'cloudinary';
import { Folder } from '@nx-pnpm-monorepo/cloudinary-photos-app/types';
import { Button } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/ui';
import Link from 'next/link';
import { Heart } from '@nx-pnpm-monorepo/cloudinary-photos-app/components/icons';

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
