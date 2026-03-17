"use client";

import { useState, useEffect, useCallback } from "react";

export function useScrollState() {
  const [isScrollingDown, setIsScrollingDown] = useState(false);
  const [isAtTop, setIsAtTop] = useState(true);
  const [lastScrollY, setLastScrollY] = useState(0);

  const handleScroll = useCallback(() => {
    const currentScrollY = window.scrollY;
    
    // Check if at top
    setIsAtTop(currentScrollY < 10);

    // Determine direction
    if (currentScrollY > lastScrollY && currentScrollY > 50) {
      setIsScrollingDown(true);
    } else if (currentScrollY < lastScrollY) {
      setIsScrollingDown(false);
    }

    setLastScrollY(currentScrollY);
  }, [lastScrollY]);

  useEffect(() => {
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, [handleScroll]);

  return { isScrollingDown, isAtTop, scrollY: lastScrollY };
}
