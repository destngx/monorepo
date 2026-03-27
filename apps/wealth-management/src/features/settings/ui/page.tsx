'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@wealth-management/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Save, Brain, Check, Github, Search, Filter } from 'lucide-react';
import { useAISettings, AIProvider } from '@/hooks/use-ai-settings';
import { AI_MODELS } from '@wealth-management/ai';
import { useState, useEffect, useMemo } from 'react';
import { getCredentialStatuses } from '@/app/actions/cache';

export default function SettingsPage() {
  const { settings, updateSettings, mounted } = useAISettings();
  const [selectedProvider, setSelectedProvider] = useState<AIProvider>(settings.provider);
  const [selectedModel, setSelectedModel] = useState(settings.modelId);
  const [isSaved, setIsSaved] = useState(false);
  const [credentials, setCredentials] = useState({ github: false, openai: false, anthropic: false, google: false });

  useEffect(() => {
    if (mounted) {
      setSelectedProvider(settings.provider);
      setSelectedModel(settings.modelId);
      void (async () => {
        const statuses = await getCredentialStatuses();
        setCredentials(statuses);
      })();
    }
  }, [mounted, settings.provider, settings.modelId]);

  const filteredModels = useMemo(() => {
    const entries = Object.entries(AI_MODELS);

    // Priority sorting: put github models first if provider is 'all' or 'github'
    entries.sort((a, b) => {
      if (a[1].provider === 'github' && b[1].provider !== 'github') return -1;
      if (a[1].provider !== 'github' && b[1].provider === 'github') return 1;
      return 0;
    });

    if (selectedProvider === 'all') return entries;
    return entries.filter(([_id, config]) => config.provider === selectedProvider);
  }, [selectedProvider]);

  // If selected model is not in the filtered list, we should change it or at least show warning
  const isModelAvailable = useMemo(() => {
    return filteredModels.some(([id]) => id === selectedModel);
  }, [filteredModels, selectedModel]);

  const handleSave = () => {
    updateSettings({
      provider: selectedProvider,
      modelId: selectedModel,
    });
    setIsSaved(true);
    setTimeout(() => setIsSaved(false), 2000);
  };

  const hasChanges = selectedProvider !== settings.provider || selectedModel !== settings.modelId;

  if (!mounted) return null;

  return (
    <div className="space-y-6 max-w-4xl mx-auto pb-10">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Settings</h2>
        <p className="text-muted-foreground text-sm">Manage your integrations and platform preferences.</p>
      </div>

      <div className="grid gap-6">
        <Card className="shadow-sm border-primary/10">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Brain className="h-5 w-5 text-primary" />
              AI Wealth Assistant
            </CardTitle>
            <CardDescription>Choose how you want to interact with your financial AI.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-8">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Provider Selection */}
              <div className="space-y-3">
                <Label className="text-sm font-semibold flex items-center gap-2">
                  <Filter className="h-3.5 w-3.5" /> Preferred Provider
                </Label>
                <Select value={selectedProvider} onValueChange={(val) => setSelectedProvider(val as AIProvider)}>
                  <SelectTrigger className="h-11">
                    <SelectValue placeholder="All Providers" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Any / All Providers</SelectItem>
                    <SelectItem value="github">
                      <div className="flex flex-col gap-0.5">
                        <div className="flex items-center gap-2">
                          <Github className="h-3 w-3" />
                          <span className="font-medium">GitHub Copilot</span>
                        </div>
                        <span className="text-[9px] text-muted-foreground ml-5">GPT-4o, GPT-o1, Grok</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="openai">OpenAI (Direct API)</SelectItem>
                    <SelectItem value="google">Google Gemini</SelectItem>
                    <SelectItem value="anthropic">Anthropic Claude</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-[10px] text-muted-foreground italic">
                  Filtering by provider helps you stay within your existing subscriptions.
                </p>
              </div>

              {/* Model Selection */}
              <div className="space-y-3">
                <Label className="text-sm font-semibold flex items-center gap-2">
                  <Search className="h-3.5 w-3.5" /> Specific Model
                </Label>
                <Select value={selectedModel} onValueChange={setSelectedModel}>
                  <SelectTrigger className="h-11 border-primary/20 bg-primary/5">
                    <SelectValue placeholder="Select an AI model" />
                  </SelectTrigger>
                  <SelectContent>
                    {filteredModels.map(([id, config]) => (
                      <SelectItem key={id} value={id} className="py-2.5">
                        <div className="flex flex-col gap-0.5">
                          <span className="font-medium text-xs">{config.label}</span>
                          <span className="text-[9px] text-muted-foreground truncate max-w-[200px]">
                            {config.description}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                    {filteredModels.length === 0 && (
                      <div className="p-4 text-center text-xs text-muted-foreground">
                        No models found for this provider.
                      </div>
                    )}
                  </SelectContent>
                </Select>
                {!isModelAvailable && selectedProvider !== 'all' && (
                  <p className="text-[10px] text-amber-600 font-medium">
                    ⚠️ Current model belongs to another provider. Please re-select.
                  </p>
                )}
              </div>
            </div>

            <div className="bg-muted/40 p-4 rounded-xl border border-dashed text-xs space-y-2">
              <p className="font-semibold text-muted-foreground uppercase text-[10px] tracking-wider">
                Credential Status
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">GitHub Copilot Token:</span>
                  {credentials.github ? (
                    <span className="text-emerald-500 font-medium">✓ Configured</span>
                  ) : (
                    <span className="text-rose-400 font-medium">✗ Missing</span>
                  )}
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">OpenAI API Key:</span>
                  {credentials.openai ? (
                    <span className="text-emerald-500 font-medium">✓ Configured</span>
                  ) : (
                    <span className="text-rose-400 font-medium">✗ Missing</span>
                  )}
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Anthropic Key:</span>
                  {credentials.anthropic ? (
                    <span className="text-emerald-500 font-medium">✓ Configured</span>
                  ) : (
                    <span className="text-rose-400 font-medium">✗ Missing</span>
                  )}
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Google (Gemini) Key:</span>
                  {credentials.google ? (
                    <span className="text-emerald-500 font-medium">✓ Configured</span>
                  ) : (
                    <span className="text-rose-400 font-medium">✗ Missing</span>
                  )}
                </div>
              </div>
            </div>
          </CardContent>
          <CardFooter className="border-t bg-muted/20 px-6 py-4 flex justify-end">
            <Button
              onClick={handleSave}
              className="gap-2 px-8"
              disabled={!hasChanges || (!isModelAvailable && selectedProvider !== 'all')}
            >
              {isSaved ? (
                <>
                  <Check className="h-4 w-4" /> Preference Saved
                </>
              ) : (
                <>
                  <Save className="h-4 w-4" /> Save Preferences
                </>
              )}
            </Button>
          </CardFooter>
        </Card>

        {/* Sync Settings Header */}
        <div className="pt-4 pb-2">
          <h3 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">Data Infrastructure</h3>
        </div>

        <Card className="shadow-sm border-emerald-500/10 bg-emerald-500/[0.02]">
          <CardHeader>
            <CardTitle className="text-base">Google Sheets Integration</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5 text-xs">
              <Label className="text-[10px] text-muted-foreground uppercase">Target Document</Label>
              <div className="h-9 flex items-center px-3 rounded border bg-background font-mono truncate">
                1BxiMVs0XRYFg2Uxyj-86xxxxxxxxxxxxxxxxxxxxxxxxxxxx
              </div>
            </div>

            <div className="flex gap-3 items-start p-3 rounded-lg bg-emerald-500/5 border border-emerald-500/20">
              <div className="h-5 w-5 rounded-full bg-emerald-500/20 flex items-center justify-center shrink-0 mt-0.5">
                <div className="h-2 w-2 rounded-full bg-emerald-500"></div>
              </div>
              <div>
                <h4 className="font-bold text-xs text-emerald-800">Connection Active</h4>
                <p className="text-[10px] text-emerald-700/70 mt-0.5 leading-relaxed">
                  WealthOS is successfully communicating with your spreadsheet. Transactions and accounts are synced
                  every 15 minutes.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
