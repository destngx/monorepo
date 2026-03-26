'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { HeartPulse, CheckCircle2, AlertCircle } from 'lucide-react';
import { AIDataInsight } from '@wealth-management/ui';

export default function HealthPage() {
  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2">
          <h2 className="text-2xl font-bold tracking-tight">Financial Health</h2>
          <AIDataInsight
            type="Financial Health Analysis"
            description="Deep dive into financial strengths and areas for improvement."
            data={{ score: 85 }} // We can pass more context later if needed
          />
        </div>
        <p className="text-muted-foreground text-sm">AI analysis of your financial standing.</p>
        <div className="bg-primary text-primary-foreground font-bold text-2xl px-6 py-2 rounded-full shadow-md flex items-center gap-2">
          <HeartPulse className="h-6 w-6" />
          85 / 100
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="shadow-sm border-emerald-200 bg-emerald-50/50">
          <CardHeader>
            <CardTitle className="text-emerald-800 flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-emerald-600" />
              Strengths
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="bg-white p-4 rounded-lg shadow-sm border border-emerald-100">
              <h4 className="font-semibold text-sm mb-1">Excellent Savings Rate</h4>
              <p className="text-xs text-muted-foreground">
                You are saving over 30% of your income consistently over the last 3 months.
              </p>
            </div>
            <div className="bg-white p-4 rounded-lg shadow-sm border border-emerald-100">
              <h4 className="font-semibold text-sm mb-1">Low Debt Utilization</h4>
              <p className="text-xs text-muted-foreground">
                Your outstanding credit card balances are extremely low compared to your assets.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="shadow-sm border-amber-200 bg-amber-50/50">
          <CardHeader>
            <CardTitle className="text-amber-800 flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-amber-600" />
              Areas of Improvement
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="bg-white p-4 rounded-lg shadow-sm border border-amber-100">
              <h4 className="font-semibold text-sm mb-1">High Food Delivery Expense</h4>
              <p className="text-xs text-muted-foreground">
                Your spending on Food Delivery has increased by 45% this month. Consider cooking at home to stay within
                budget.
              </p>
            </div>
            <div className="bg-white p-4 rounded-lg shadow-sm border border-amber-100">
              <h4 className="font-semibold text-sm mb-1">Emergency Fund Gap</h4>
              <p className="text-xs text-muted-foreground">
                Your emergency fund currently covers 2 months of expenses. The target is 6 months.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
