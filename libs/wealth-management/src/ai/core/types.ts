/**
 * Structured insight types for AI-generated analysis.
 * Used by all AI insight components for consistent rendering.
 */

export type InsightSeverity = 'info' | 'warning' | 'success' | 'critical';

export type InsightIconHint =
  | 'alert'
  | 'trend-up'
  | 'trend-down'
  | 'savings'
  | 'tip'
  | 'target'
  | 'shield'
  | 'chart'
  | 'wallet'
  | 'clock'
  | 'zap'
  | 'star';

export interface InsightSection {
  /** Semantic icon hint for the section */
  icon: InsightIconHint;
  /** Section heading */
  title: string;
  /** The insight text content */
  content: string;
  /** Visual severity level */
  severity: InsightSeverity;
}

export interface StructuredInsight {
  /** One-line key takeaway */
  summary: string;
  /** Individual insight sections */
  sections: InsightSection[];
  /** Quick-reference action items */
  actionItems?: string[];
}

/**
 * JSON instruction block to append to AI prompts for structured output.
 * Ensures the model returns a parseable StructuredInsight JSON.
 */
export const STRUCTURED_INSIGHT_FORMAT_INSTRUCTION = `
IMPORTANT: You MUST respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this exact format:
{
  "summary": "One concise sentence summarizing the key takeaway",
  "sections": [
    {
      "icon": "<one of: alert, trend-up, trend-down, savings, tip, target, shield, chart, wallet, clock, zap, star>",
      "title": "Section Title",
      "content": "Detailed insight text for this section. Keep each section focused on ONE specific recommendation or finding.",
      "severity": "<one of: info, warning, success, critical>"
    }
  ],
  "actionItems": [
    "Action item 1",
    "Action item 2"
  ]
}

Rules:
- Provide 3-5 sections, each focused on a different aspect
- Each section content should be 1-3 sentences
- Provide 2-4 actionable items
- severity "critical" = needs immediate attention, "warning" = should address soon, "success" = positive finding, "info" = informational
- Keep the summary under 20 words
`;
