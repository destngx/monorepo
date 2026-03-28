# Error Notification System

Comprehensive error handling with user-friendly notifications for the wealth-management app.

## Features

✅ **Global Error Display** - Toast-style notifications that appear in top-right corner  
✅ **Auto-dismiss** - Errors automatically close after 5 seconds  
✅ **Manual Dismiss** - Users can close errors with an X button  
✅ **Severity Levels** - Error (red), Warning (yellow), Info (blue) styling  
✅ **Dark Mode Support** - Adapts to light/dark theme  
✅ **Stacked Notifications** - Multiple errors display together  
✅ **Type-Safe** - Full TypeScript support

## Architecture

### Components & Hooks

```
libs/wealth-management/src/
├── hooks/
│   ├── use-error-notifications.ts     # Context hook (library)
│   └── use-api-error-handler.ts       # Component hook (library)
├── utils/
│   ├── api-error-handler.ts           # Error parsing utility
│   └── api-wrapper.ts                 # Fetch wrapper (optional)
└── errors/
    ├── AppError.ts                    # Base error class with userMessage
    └── types.ts                       # Error codes & severity enums

apps/wealth-management/src/
├── components/
│   ├── error-notifications-provider.tsx    # Context provider (app-level)
│   └── error-notification-display.tsx      # Toast UI component
└── app/
    └── layout.tsx                          # Root layout integration
```

## How It Works

### 1. Setup (Already Done)

Root layout wraps the app with the provider:

```tsx
// apps/wealth-management/src/app/layout.tsx
<ErrorNotificationsProvider>
  {/* App content */}
  <ErrorNotificationDisplay />
</ErrorNotificationsProvider>
```

### 2. Throwing Errors in API Calls

Mutations should throw `AppError` with `userMessage`:

```tsx
// libs/wealth-management/src/features/chat/model/mutations.ts
export async function sendChatMessage(...): Promise<ReadableStream> {
  const response = await fetch('/api/chat', { ... });

  if (!response.ok) {
    const error = await handleApiError(response);  // Parses response
    throw error;  // Has userMessage property
  }

  return response.body;
}
```

### 3. Using in Components

Components catch errors and display them:

```tsx
'use client';

import { useApiErrorHandler } from '@wealth-management/hooks';
import { sendChatMessage } from '@wealth-management/features/chat';

export function ChatInterface() {
  const { withErrorHandling } = useApiErrorHandler();

  const handleSendMessage = async () => {
    const result = await withErrorHandling(
      () => sendChatMessage(messages),
      'Failed to send message', // Fallback message
    );

    if (result) {
      // Process stream...
    }
  };

  return <button onClick={handleSendMessage}>Send</button>;
}
```

## API Reference

### useErrorNotifications()

Hook to access the error notification context.

**Returns:**

```typescript
{
  notifications: ErrorNotification[];
  addError: (message: string, severity?: 'error' | 'warning' | 'info') => void;
  removeError: (id: string) => void;
  clearAll: () => void;
}
```

**Example:**

```tsx
const { notifications, addError, removeError } = useErrorNotifications();

addError('Something went wrong', 'error');
removeError(notifications[0].id);
```

### useApiErrorHandler()

Hook for catching and displaying errors in components.

**Returns:**

```typescript
{
  handleError: (error: unknown, fallbackMessage?: string) => void;
  withErrorHandling: <T>(asyncFn: () => Promise<T>, fallbackMessage?: string) => Promise<T | null>;
  addError: (message: string, severity?: 'error' | 'warning' | 'info') => void;
}
```

**Example:**

```tsx
const { withErrorHandling, handleError } = useApiErrorHandler();

// Pattern 1: withErrorHandling wrapper
const result = await withErrorHandling(() => fetchData(), 'Failed to fetch data');

// Pattern 2: Try-catch with handleError
try {
  await apiCall();
} catch (error) {
  handleError(error, 'Custom message');
}
```

### handleApiError(response: Response)

Converts a failed fetch response to AppError with userMessage.

**Returns:** `Promise<AppError>`

**Example:**

```tsx
const response = await fetch('/api/data');
if (!response.ok) {
  const error = await handleApiError(response);
  throw error; // Now has userMessage
}
```

### ErrorNotification

```typescript
interface ErrorNotification {
  id: string;
  message: string;
  severity: 'error' | 'warning' | 'info';
  timestamp: number;
  dismissible?: boolean;
}
```

## Styling

Notifications automatically adapt to dark mode and support Tailwind classes.

**Error (Red):**

```
Light: bg-red-50 border-red-200 text-red-900
Dark: bg-red-900/20 border-red-800 text-red-100
```

**Warning (Yellow):**

```
Light: bg-yellow-50 border-yellow-200 text-yellow-900
Dark: bg-yellow-900/20 border-yellow-800 text-yellow-100
```

**Info (Blue):**

```
Light: bg-blue-50 border-blue-200 text-blue-900
Dark: bg-blue-900/20 border-blue-800 text-blue-100
```

## Integration Patterns

### Pattern 1: Catch and Display in Component

```tsx
export function MyComponent() {
  const { withErrorHandling } = useApiErrorHandler();
  const [data, setData] = useState(null);

  const loadData = async () => {
    const result = await withErrorHandling(() => fetchMyData(), 'Failed to load data');
    if (result) setData(result);
  };

  return (
    <>
      <button onClick={loadData}>Load</button>
      {data && <div>{JSON.stringify(data)}</div>}
    </>
  );
}
```

### Pattern 2: Error Boundary with Fallback

```tsx
export function ProtectedComponent({ children }) {
  const { handleError, addError } = useApiErrorHandler();

  useEffect(() => {
    const errorHandler = (event: ErrorEvent) => {
      handleError(event.error);
    };
    window.addEventListener('error', errorHandler);
    return () => window.removeEventListener('error', errorHandler);
  }, [handleError]);

  return children;
}
```

### Pattern 3: Mutation with Built-in Error Handling

```tsx
export async function createLoanSafe(loan: Loan) {
  const response = await fetch('/api/loans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(loan),
  });

  if (!response.ok) {
    const error = await handleApiError(response);
    throw error; // Throws with userMessage
  }

  return response.json();
}

// In component:
const result = await withErrorHandling(() => createLoanSafe(data));
```

## Backend Error Response Format

The system expects backend errors to include `userMessage`:

```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid email format provided",
  "userMessage": "Please enter a valid email address",
  "statusCode": 400
}
```

If `userMessage` is missing, a default message based on status code is used:

- 400: "Invalid request. Please check your input and try again."
- 401: "Authentication failed. Please log in again."
- 403: "You do not have permission to access this resource."
- 404: "The requested resource was not found."
- 500: "Server error occurred. Please try again later."

## Testing

### Manual Testing

1. Open the app in browser
2. Trigger an API error (e.g., network failure)
3. Verify toast notification appears top-right
4. Verify it auto-dismisses after 5 seconds
5. Verify the X button manually closes it

### Testing Patterns

```tsx
// In your component test
const { addError } = useErrorNotifications();

await act(() => {
  addError('Test error', 'error');
});

expect(screen.getByText('Test error')).toBeInTheDocument();
```

## Common Issues & Solutions

### Issue: Notification not showing

**Solution:** Ensure `ErrorNotificationsProvider` wraps your component tree and `ErrorNotificationDisplay` is rendered.

### Issue: Error message is generic

**Solution:** Update backend to include `userMessage` in error response.

### Issue: Multiple notifications not stacking

**Solution:** This is expected - check `ErrorNotificationDisplay` for z-index conflicts in your CSS.

## Files Changed

### New Files

- `libs/wealth-management/src/hooks/use-error-notifications.ts`
- `libs/wealth-management/src/hooks/use-api-error-handler.ts`
- `libs/wealth-management/src/utils/api-error-handler.ts`
- `libs/wealth-management/src/utils/api-wrapper.ts` (optional)
- `apps/wealth-management/src/components/error-notifications-provider.tsx`
- `apps/wealth-management/src/components/error-notification-display.tsx`
- `libs/wealth-management/src/features/loans/model/mutations-with-error-handling.example.ts`

### Modified Files

- `libs/wealth-management/src/hooks/index.ts` - Added exports
- `apps/wealth-management/src/app/layout.tsx` - Added provider & display

## Next Steps

1. **Update existing mutations** to use `handleApiError()` for proper user messages
2. **Add error handling to components** using `useApiErrorHandler()`
3. **Test with real API failures** (network, 500, auth failures)
4. **Customize styling** if needed in `error-notification-display.tsx`
5. **Add more error types** to backend as needed

## Build Status

✅ Build passes: `bun run build wealth-management`
