'use client';

import { AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@wealth-management/ui/card';

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html>
      <body>
        <div className="flex items-center justify-center min-h-screen p-4">
          <Card className="w-full max-w-md border-red-500/20 bg-red-500/5">
            <CardHeader>
              <div className="flex items-center gap-3">
                <AlertCircle className="h-6 w-6 text-red-600" />
                <div>
                  <CardTitle className="text-red-600">Critical Error</CardTitle>
                  <CardDescription className="text-sm mt-1">
                    An unexpected error occurred. Please try refreshing the page.
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="bg-red-950/20 rounded-lg p-3 text-sm text-foreground/80 font-mono break-words">
                {error.message || 'Unknown error'}
              </div>
              <Button onClick={() => reset()} variant="outline" className="w-full gap-2">
                <RefreshCw className="h-4 w-4" />
                Try Again
              </Button>
              <Button onClick={() => (window.location.href = '/')} variant="ghost" className="w-full">
                Go Home
              </Button>
            </CardContent>
          </Card>
        </div>
      </body>
    </html>
  );
}
