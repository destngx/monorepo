import { NextResponse } from 'next/server';
import { getPendingNotifications, markNotificationDone } from '@wealth-management/services/server';
import { AppError, ValidationError, isAppError } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const notifications = await getPendingNotifications();
    return NextResponse.json(notifications);
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError({
      message: error instanceof Error ? error.message : 'Failed to fetch notifications',
    });
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}

export async function PATCH(req: Request) {
  try {
    const { rowNumbers } = (await req.json()) as { rowNumbers: number[] };
    if (!Array.isArray(rowNumbers)) {
      throw new ValidationError('rowNumbers must be an array');
    }

    await Promise.all(rowNumbers.map((n) => markNotificationDone(n)));
    return NextResponse.json({ success: true });
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError({
      message: error instanceof Error ? error.message : 'Failed to update notifications',
    });
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
