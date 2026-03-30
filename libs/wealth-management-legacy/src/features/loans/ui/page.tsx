import { Suspense } from "react";
import { getLoans } from '../model/server-queries';

export const dynamic = 'force-dynamic';
export const revalidate = 0;
import { LoanSummary } from "./loan-summary";
import { LoanList } from "./loan-list";
import { LoanReviewAI } from "./loan-review-ai";
import { RefreshCw, Target } from "lucide-react";
import { AIDataInsight } from "@/components/dashboard/ai-data-insight";

async function LoanContent() {
  const loans = await getLoans();

  return (
    <div className="flex flex-col gap-8">
      <LoanSummary loans={loans} />
      <LoanReviewAI loans={loans} />
      
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-bold flex items-center gap-2">
              <RefreshCw className="h-5 w-5 text-primary animate-spin-slow" />
              Detailed Debt Tracker
            </h2>
            <AIDataInsight 
              type="Loan & Debt Analysis"
              description="Detailed breakdown of borrowing and lending progress."
              data={loans}
            />
          </div>
        </div>
        <LoanList loans={loans} />
      </div>
    </div>
  );
}

export default function LoansPage() {
  return (
    <div className="flex flex-col gap-8 p-4 md:p-8 max-w-6xl mx-auto min-h-screen">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2 text-primary">
          <Target className="h-6 w-6" />
          <h1 className="text-3xl font-bold tracking-tight">Loans & Debts</h1>
        </div>
        <p className="text-muted-foreground">
          Track your personal borrowing, lending, and monthly repayment progress.
        </p>
      </div>

      <Suspense fallback={<div className="h-96 flex items-center justify-center text-muted-foreground animate-pulse italic">Thinking...</div>}>
        <LoanContent />
      </Suspense>
    </div>
  );
}
