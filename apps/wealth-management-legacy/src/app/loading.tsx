"use client";

import { Loading } from "@wealth-management/ui/loading";

export default function GlobalLoading() {
  return (
    <div className="flex h-[80vh] w-full items-center justify-center">
      <Loading message="Thinking..." />
    </div>
  );
}
