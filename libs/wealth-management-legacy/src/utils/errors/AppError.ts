import { ErrorCode, ErrorResponse, ErrorSeverity } from './types';

/**
 * Base error class for application errors
 * Provides structured error information with context preservation
 */
export class AppError extends Error {
  readonly code: ErrorCode;
  readonly severity: ErrorSeverity;
  readonly statusCode: number;
  readonly context: Record<string, unknown>;
  readonly userMessage: string;

  constructor(
    message: string,
    code: ErrorCode = ErrorCode.INTERNAL_ERROR,
    statusCode = 500,
    severity: ErrorSeverity = ErrorSeverity.MEDIUM,
    options?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.severity = severity;
    this.statusCode = statusCode;

    const { context, userMessage, ...contextProps } = options || {};
    this.context = (context as Record<string, unknown>) || contextProps || {};
    this.userMessage = (userMessage as string) || this.getDefaultUserMessage(code);

    // Maintain proper prototype chain
    Object.setPrototypeOf(this, AppError.prototype);
  }

  private getDefaultUserMessage(code: ErrorCode): string {
    switch (code) {
      case ErrorCode.VALIDATION_ERROR:
        return 'Invalid input provided. Please check your data and try again.';
      case ErrorCode.AUTH_ERROR:
      case ErrorCode.UNAUTHORIZED:
        return 'Authentication failed. Please log in again.';
      case ErrorCode.FORBIDDEN:
        return 'You do not have permission to perform this action.';
      case ErrorCode.NOT_FOUND:
      case ErrorCode.RESOURCE_NOT_FOUND:
        return 'The requested resource was not found.';
      case ErrorCode.NETWORK_ERROR:
      case ErrorCode.CONNECTION_FAILED:
        return 'Connection failed. Please check your internet and try again.';
      case ErrorCode.REQUEST_TIMEOUT:
        return 'Request timed out. Please try again.';
      case ErrorCode.STORAGE_ERROR:
        return 'Storage operation failed. Please try again.';
      default:
        return 'An unexpected error occurred. Please try again later.';
    }
  }

  toResponse(): ErrorResponse {
    return {
      code: this.code,
      message: this.message,
      userMessage: this.userMessage,
      severity: this.severity,
      statusCode: this.statusCode,
      context: this.context,
      timestamp: new Date().toISOString(),
    };
  }
}

export class ValidationError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.VALIDATION_ERROR, 400, ErrorSeverity.LOW, options);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }
}

export class AuthError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.AUTH_ERROR, 401, ErrorSeverity.HIGH, options);
    this.name = 'AuthError';
    Object.setPrototypeOf(this, AuthError.prototype);
  }
}

export class NotFoundError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.NOT_FOUND, 404, ErrorSeverity.LOW, options);
    this.name = 'NotFoundError';
    Object.setPrototypeOf(this, NotFoundError.prototype);
  }
}

export class NetworkError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.NETWORK_ERROR, 503, ErrorSeverity.MEDIUM, options);
    this.name = 'NetworkError';
    Object.setPrototypeOf(this, NetworkError.prototype);
  }
}

export class ChatError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.CHAT_ERROR, 500, ErrorSeverity.HIGH, options);
    this.name = 'ChatError';
    Object.setPrototypeOf(this, ChatError.prototype);
  }
}

export class LoanError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.LOAN_ERROR, 500, ErrorSeverity.HIGH, options);
    this.name = 'LoanError';
    Object.setPrototypeOf(this, LoanError.prototype);
  }
}

export class GoalError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.GOAL_ERROR, 500, ErrorSeverity.HIGH, options);
    this.name = 'GoalError';
    Object.setPrototypeOf(this, GoalError.prototype);
  }
}

export class StorageError extends AppError {
  constructor(message: string, options?: Record<string, unknown>) {
    super(message, ErrorCode.STORAGE_ERROR, 500, ErrorSeverity.MEDIUM, options);
    this.name = 'StorageError';
    Object.setPrototypeOf(this, StorageError.prototype);
  }
}
