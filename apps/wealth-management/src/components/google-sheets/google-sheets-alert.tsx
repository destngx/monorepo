"use client";

import { AlertTriangle, Terminal } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface GoogleSheetsAlertProps {
  errorType: 'MISSING_CREDENTIALS' | 'OAUTH_EXPIRED' | 'API_ERROR';
}

export function GoogleSheetsAlert({ errorType }: GoogleSheetsAlertProps) {
  const isExpired = errorType === 'OAUTH_EXPIRED';
  
  return (
    <Card className="border-red-200 bg-red-50/50 dark:bg-red-950/10 shadow-sm overflow-hidden animate-in fade-in slide-in-from-top-4 duration-300">
      <div className="absolute top-0 left-0 w-1 h-full bg-red-500" />
      <CardHeader className="pb-2">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400">
            <AlertTriangle className="h-5 w-5" />
          </div>
          <CardTitle className="text-lg text-red-700 dark:text-red-400">
            {isExpired ? "Google Sheets Session Expired" : "Google Sheets Connection Error"}
          </CardTitle>
        </div>
        <CardDescription className="text-red-600/80 dark:text-red-400/80 pl-9">
          {isExpired 
            ? "Your authentication token has expired. This typically happens every 7 days if your Google Cloud project is in 'Testing' mode."
            : "The application is unable to connect to your Google Spreadsheet. Please verify your credentials."}
        </CardDescription>
      </CardHeader>
      <CardContent className="pl-9 space-y-4">
        <div className="rounded-lg bg-black/5 dark:bg-white/5 p-4 border border-black/10 dark:border-white/10">
          <div className="flex items-start gap-3">
            <Terminal className="h-4 w-4 mt-1 text-muted-foreground" />
            <div className="space-y-2">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Solution</p>
              <p className="text-sm">Run the following command in your terminal to re-authorize:</p>
              <code className="block w-full p-2 bg-black text-green-400 rounded text-xs">
                pnpm run auth:setup
              </code>
            </div>
          </div>
        </div>
        
        <div className="flex flex-col sm:flex-row gap-3">
          <p className="text-xs text-muted-foreground flex-1">
            <span className="font-semibold text-foreground">Pro-tip:</span> To prevent this from happening every week, set your Google Cloud Project to &quot;Production&quot; status in the Google Cloud Console.
          </p>
          <Button variant="outline" size="sm" className="gap-2 h-8 text-xs cursor-pointer" asChild>
            <a href="https://console.cloud.google.com/apis/credentials/consent" target="_blank" rel="noopener noreferrer">
              GCP Console
            </a>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
