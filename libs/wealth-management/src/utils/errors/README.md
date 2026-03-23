# Error Handling Library

Comprehensive, type-safe error handling system for wealth-management app.

## Usage

### Basic Error Throwing

```typescript
import { NetworkError, ValidationError, AppError } from '@wealth-management/utils/errors';

// Throw specific error types
throw new NetworkError('Failed to fetch data', {
  context: { endpoint: '/api/data', statusCode: 500 },
});

throw new ValidationError('Email is invalid', {
  userMessage: 'Please enter a valid email address',
});

// Throw generic error
throw new AppError('Something went wrong');
```

### Error Catching & Handling

```typescript
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

try {
  // operation
} catch (error) {
  if (isAppError(error)) {
    console.log(error.userMessage); // Safe user-facing message
    console.log(error.toResponse()); // API response format
  } else {
    console.log(getErrorMessage(error));
  }
}
```

### In React Hooks

```typescript
const [error, setError] = useState<AppError | null>(null);

fetch('/api/data').catch((err) => {
  const appError = err instanceof AppError ? err : new AppError(err.message);
  setError(appError);
});
```

### Error Severity & Categorization

- **LOW**: Validation, not found (user error, recoverable)
- **MEDIUM**: Network, storage (environmental, retry recommended)
- **HIGH**: Auth, feature-specific (security or data integrity)
- **CRITICAL**: Reserved for severe failures

## Error Types

- `ValidationError` — Invalid input (400)
- `AuthError` — Authentication/authorization (401)
- `NotFoundError` — Resource not found (404)
- `NetworkError` — Network/connectivity (503)
- `ChatError` — Chat feature (500)
- `LoanError` — Loan feature (500)
- `GoalError` — Goal feature (500)
- `StorageError` — Storage operation (500)
- `AppError` — Generic fallback (500)
