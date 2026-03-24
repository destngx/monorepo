import { AppError } from './errors';

export interface ApiErrorResponse {
  code?: string;
  message?: string;
  userMessage?: string;
  statusCode?: number;
}

export async function handleApiError(response: Response): Promise<AppError> {
  let errorData: ApiErrorResponse = {};

  try {
    errorData = await response.json();
  } catch {
    errorData = {};
  }

  const message = errorData.message || response.statusText || 'An error occurred';
  const userMessage = errorData.userMessage || getDefaultUserMessage(response.status);
  const statusCode = response.status;

  const error = new AppError(message, undefined, statusCode, undefined, {
    context: { endpoint: new URL(response.url).pathname, statusCode },
    userMessage,
  });

  return error;
}

function getDefaultUserMessage(statusCode: number): string {
  switch (statusCode) {
    case 400:
      return 'Invalid request. Please check your input and try again.';
    case 401:
      return 'Authentication failed. Please log in again.';
    case 403:
      return 'You do not have permission to access this resource.';
    case 404:
      return 'The requested resource was not found.';
    case 500:
      return 'Server error occurred. Please try again later.';
    case 503:
      return 'Service is temporarily unavailable. Please try again later.';
    default:
      return 'An error occurred. Please try again.';
  }
}

export function extractUserMessage(error: unknown): string {
  if (error instanceof AppError) {
    return error.userMessage;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'An unexpected error occurred';
}
