/**
 * Error Code Enumeration
 * Used for client-side error identification and categorization
 */
export enum ErrorCode {
  // Validation errors
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  MISSING_REQUIRED_FIELD = 'MISSING_REQUIRED_FIELD',
  INVALID_FORMAT = 'INVALID_FORMAT',

  // Authentication/Authorization errors
  AUTH_ERROR = 'AUTH_ERROR',
  UNAUTHORIZED = 'UNAUTHORIZED',
  FORBIDDEN = 'FORBIDDEN',
  SESSION_EXPIRED = 'SESSION_EXPIRED',

  // Not Found errors
  NOT_FOUND = 'NOT_FOUND',
  RESOURCE_NOT_FOUND = 'RESOURCE_NOT_FOUND',

  // Network errors
  NETWORK_ERROR = 'NETWORK_ERROR',
  REQUEST_TIMEOUT = 'REQUEST_TIMEOUT',
  CONNECTION_FAILED = 'CONNECTION_FAILED',

  // Feature-specific errors (Chat)
  CHAT_ERROR = 'CHAT_ERROR',
  CHAT_MESSAGE_FAILED = 'CHAT_MESSAGE_FAILED',
  CHAT_STREAM_FAILED = 'CHAT_STREAM_FAILED',

  // Feature-specific errors (Loans)
  LOAN_ERROR = 'LOAN_ERROR',
  LOAN_CREATE_FAILED = 'LOAN_CREATE_FAILED',
  LOAN_UPDATE_FAILED = 'LOAN_UPDATE_FAILED',
  LOAN_DELETE_FAILED = 'LOAN_DELETE_FAILED',

  // Feature-specific errors (Goals)
  GOAL_ERROR = 'GOAL_ERROR',
  GOAL_CREATE_FAILED = 'GOAL_CREATE_FAILED',
  GOAL_UPDATE_FAILED = 'GOAL_UPDATE_FAILED',
  GOAL_DELETE_FAILED = 'GOAL_DELETE_FAILED',

  // Storage errors
  STORAGE_ERROR = 'STORAGE_ERROR',
  STORAGE_QUOTA_EXCEEDED = 'STORAGE_QUOTA_EXCEEDED',

  // Generic errors
  INTERNAL_ERROR = 'INTERNAL_ERROR',
  UNKNOWN_ERROR = 'UNKNOWN_ERROR',
}

/**
 * Error Severity Levels
 * Used for UI/UX decisions (alert type, retry strategy, etc.)
 */
export enum ErrorSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

/**
 * Standardized API Error Response
 * Used for consistent error communication between server and client
 */
export interface ErrorResponse {
  code: ErrorCode;
  message: string;
  userMessage: string;
  severity: ErrorSeverity;
  statusCode: number;
  context?: Record<string, unknown>;
  timestamp: string;
}
