'use client';

import { memo } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Zap, CheckCircle } from 'lucide-react';

interface Props {
  message: any;
}

export const MessageContent = memo(({ message }: Props) => {
  const parts = message.parts || [];
  const toolInvocations = message.toolInvocations || [];

  if (parts.length === 0 && toolInvocations.length === 0 && !message.content) return null;

  return (
    <div className="space-y-4">
      {/* 1. Legacy Content */}
      {message.content && parts.length === 0 && (
        <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 prose-headings:my-2 prose-headings:font-semibold prose-ul:my-2 prose-ol:my-2 prose-li:my-0">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
        </div>
      )}

      {/* 2. Parts */}
      {parts.map((part: any, idx: number) => {
        if (part.type === 'text' && part.text) {
          return (
            <div
              key={`text-${idx}`}
              className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 prose-headings:my-2 prose-headings:font-semibold prose-ul:my-2 prose-ol:my-2 prose-li:my-0"
            >
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{part.text}</ReactMarkdown>
            </div>
          );
        }

        if (typeof part.type === 'string' && part.type.startsWith('tool-')) {
          const toolName = part.type.split('-')[1];
          if (
            part.state === 'input-streaming' ||
            part.state === 'input-available' ||
            part.state === 'approval-requested'
          ) {
            return (
              <div
                key={`tool-call-${idx}`}
                className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded px-3 py-2 text-xs"
              >
                <p className="font-semibold text-blue-700 dark:text-blue-300 flex items-center gap-1">
                  <Zap className="h-3 w-3" /> Calling: {toolName}
                </p>
              </div>
            );
          }
          if (part.state === 'output-available') {
            return (
              <div
                key={`tool-result-${idx}`}
                className="bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded px-3 py-2 text-xs"
              >
                <p className="font-semibold text-green-700 dark:text-green-300 flex items-center gap-1">
                  <CheckCircle className="h-3 w-3" /> {toolName}
                </p>
                {typeof part.output === 'string' ? (
                  <p className="mt-1 whitespace-pre-wrap">{part.output}</p>
                ) : (
                  <pre className="text-xs overflow-auto mt-1 max-h-48 p-2 bg-white dark:bg-black rounded">
                    {JSON.stringify(part.output, null, 2)}
                  </pre>
                )}
              </div>
            );
          }
        }
        return null;
      })}

      {/* 3. Tool Invocations */}
      {toolInvocations.map((tool: any, idx: number) => {
        if (tool.state === 'call' || tool.state === 'partial-call') {
          return (
            <div
              key={`legacy-tool-call-${idx}`}
              className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded px-3 py-2 text-xs"
            >
              <p className="font-semibold text-blue-700 dark:text-blue-300 flex items-center gap-1">
                <Zap className="h-3 w-3" /> Calling: {tool.toolName}
              </p>
            </div>
          );
        }
        if (tool.state === 'result') {
          return (
            <div
              key={`legacy-tool-result-${idx}`}
              className="bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded px-3 py-2 text-xs"
            >
              <p className="font-semibold text-green-700 dark:text-green-300 flex items-center gap-1">
                <CheckCircle className="h-3 w-3" /> {tool.toolName}
              </p>
              {typeof tool.result === 'string' ? (
                <p className="mt-1 whitespace-pre-wrap">{tool.result}</p>
              ) : (
                <pre className="text-xs overflow-auto mt-1 max-h-48 p-2 bg-white dark:bg-black rounded">
                  {JSON.stringify(tool.result, null, 2)}
                </pre>
              )}
            </div>
          );
        }
        return null;
      })}
    </div>
  );
});

MessageContent.displayName = 'MessageContent';
