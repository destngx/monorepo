"use client";

import { CreateGoalFlow } from "./create-goal-flow";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

export default function CreateGoalPage() {
  return (
    <div className="space-y-6 max-w-2xl mx-auto pb-20">
      <div className="flex items-center">
        <Link href="/accounts/goals">
          <Button variant="ghost" size="sm" className="gap-2 text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-4 w-4" />
            Back to Goals
          </Button>
        </Link>
      </div>

      <div className="space-y-2 text-center mb-8">
        <h1 className="text-4xl font-black italic uppercase tracking-tighter">New Mission</h1>
        <p className="text-muted-foreground">Launch your next financial achievement with AI guidance.</p>
      </div>

      <CreateGoalFlow />
    </div>
  );
}
