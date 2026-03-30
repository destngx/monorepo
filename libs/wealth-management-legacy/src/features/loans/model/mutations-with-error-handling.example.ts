import { AppError } from '../../../utils/errors';
import { handleApiError, extractUserMessage } from '../../../utils/api-error-handler';

/**
 * Example: Using error handler in async functions
 *
 * This shows how to integrate error notifications into your API calls.
 * Components using this function would use useApiErrorHandler() hook to display errors.
 */

export async function createLoanWithErrorHandling(loan: any): Promise<any> {
  const response = await fetch('/api/loans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(loan),
  });

  if (!response.ok) {
    const error = await handleApiError(response);
    throw error;
  }

  return response.json();
}

/**
 * Example: Using in a component with useApiErrorHandler
 *
 * 'use client';
 *
 * import { useApiErrorHandler } from '@wealth-management/hooks';
 *
 * export function MyComponent() {
 *   const { withErrorHandling } = useApiErrorHandler();
 *
 *   const handleCreateLoan = async () => {
 *     const result = await withErrorHandling(
 *       () => createLoanWithErrorHandling(loanData),
 *       'Failed to create loan'
 *     );
 *     if (result) {
 *       // Success - result contains the created loan
 *     }
 *   };
 *
 *   return <button onClick={handleCreateLoan}>Create Loan</button>;
 * }
 */

/**
 * Example: Using in async mutations directly
 *
 * Make sure mutations throw AppError or convert errors:
 *
 * export async function sendChatMessage(...): Promise<ReadableStream> {
 *   const response = await fetch('/api/chat', { ... });
 *
 *   if (!response.ok) {
 *     const error = await handleApiError(response);
 *     throw error; // Now has userMessage
 *   }
 *
 *   return response.body;
 * }
 *
 * Then in your component:
 *
 * const { withErrorHandling, addError } = useApiErrorHandler();
 *
 * try {
 *   const stream = await withErrorHandling(() => sendChatMessage(...));
 * } catch (error) {
 *   // Already handled by hook
 * }
 */
