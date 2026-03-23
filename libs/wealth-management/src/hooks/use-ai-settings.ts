'use client';

import { useState, useEffect, useCallback } from 'react';
import { AppError, isAppError } from '../utils/errors';

const SETTINGS_KEY = 'wealthos-ai-settings';
const SYNC_EVENT = 'wealthos-ai-settings-sync';

export type AIProvider = 'all' | 'openai' | 'github' | 'google' | 'anthropic';

interface AISettings {
  provider: AIProvider;
  modelId: string;
}

const DEFAULT_SETTINGS: AISettings = {
  provider: 'all',
  modelId: 'gpt-4o-mini',
};

export function useAISettings() {
  const [settings, setSettings] = useState<AISettings>(DEFAULT_SETTINGS);
  const [mounted, setMounted] = useState(false);

  const loadSettings = useCallback(() => {
    if (typeof window === 'undefined') return;
    const saved = localStorage.getItem(SETTINGS_KEY);
    if (saved) {
      try {
        setSettings(JSON.parse(saved));
      } catch (e) {
        if (isAppError(e)) {
          console.error('Failed to parse AI settings:', e.message);
        } else {
          const error = new AppError(e instanceof Error ? e.message : 'Failed to parse AI settings');
          console.error('Failed to parse AI settings:', error.message);
        }
      }
    }
  }, []);

  useEffect(() => {
    setMounted(true);
    loadSettings();

    // Sync across components in the same tab
    const handleSync = () => loadSettings();
    window.addEventListener(SYNC_EVENT, handleSync);

    // Sync across tabs
    const handleStorage = (e: StorageEvent) => {
      if (e.key === SETTINGS_KEY) loadSettings();
    };
    window.addEventListener('storage', handleStorage);

    return () => {
      window.removeEventListener(SYNC_EVENT, handleSync);
      window.removeEventListener('storage', handleStorage);
    };
  }, [loadSettings]);

  const updateSettings = (newSettings: Partial<AISettings>) => {
    const updated = { ...settings, ...newSettings };
    setSettings(updated);
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(updated));

    // Notify other hook instances in the same window
    window.dispatchEvent(new CustomEvent(SYNC_EVENT));
  };

  return {
    settings,
    updateSettings,
    mounted,
  };
}
