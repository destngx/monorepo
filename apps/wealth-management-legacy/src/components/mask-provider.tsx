"use client";

import * as React from "react";
import { usePathname } from "next/navigation";

interface MaskContextType {
  isMasked: boolean;
  setIsMasked: (value: boolean) => void;
  toggleMask: () => void;
}

const MaskContext = React.createContext<MaskContextType | undefined>(undefined);

export function MaskProvider({ children }: { children: React.ReactNode }) {
  const [isMasked, setIsMasked] = React.useState(true);
  const pathname = usePathname();

  // Reset mask state on pathname change
  React.useEffect(() => {
    setIsMasked(true);
  }, [pathname]);

  const toggleMask = React.useCallback(() => {
    setIsMasked((prev) => !prev);
  }, []);

  return (
    <MaskContext.Provider value={{ isMasked, setIsMasked, toggleMask }}>
      {children}
    </MaskContext.Provider>
  );
}

export function useMask() {
  const context = React.useContext(MaskContext);
  if (context === undefined) {
    throw new Error("useMask must be used within a MaskProvider");
  }
  return context;
}
