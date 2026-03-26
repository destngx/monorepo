"use client";

import * as React from "react";
import { cn } from "@wealth-management/utils";
import { formatVND, formatUSD } from "@wealth-management/utils";
import { useMask } from "./mask-provider";

interface MaskedBalanceProps {
  amount?: number;
  value?: string | number;
  currency?: "VND" | "USD" | "none";
  unit?: string;
  className?: string;
  isInitiallyMasked?: boolean;
}

export function MaskedBalance({
  amount,
  value,
  currency = "VND",
  unit,
  className,
}: MaskedBalanceProps) {
  const { isMasked, toggleMask } = useMask();

  let displayValue = "";
  const numericValue = value !== undefined ? (typeof value === 'string' ? parseFloat(value.replace(/,/g, '')) : value) : amount;

  if (currency === "none") {
    displayValue = value?.toString() || amount?.toString() || "";
  } else {
    displayValue = currency === "VND" ? formatVND(numericValue || 0) : formatUSD(numericValue || 0);
  }

  if (unit && !isMasked) {
    displayValue = `${displayValue} ${unit}`;
  }
  
  // Create a display string that matches the length of the formatted amount roughly
  const maskChar = "•";
  const maskedString = maskChar.repeat(8);

  return (
    <span
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleMask();
      }}
      className={cn(
        "cursor-pointer transition-all duration-200 select-none inline-flex items-center gap-1",
        isMasked ? "font-mono tracking-widest opacity-80" : "tabular-nums",
        className
      )}
      title={isMasked ? "Click to show" : "Click to hide"}
    >
      {isMasked ? maskedString : displayValue}
      {isMasked && unit && <span className="text-[10px] opacity-50 tracking-normal">{unit}</span>}
    </span>
  );
}
