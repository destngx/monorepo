import { NextResponse } from 'next/server';
import { getPendingNotifications, markNotificationDone } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET() {
  try {
    const notifications = await getPendingNotifications();
    return NextResponse.json(notifications);
  } catch (error: any) {
    return handleApiError(error, 'Notifications');
  }
}

export async function PATCH(req: Request) {
  try {
    const { rowNumbers } = await req.json();
    if (!Array.isArray(rowNumbers)) {
      return NextResponse.json({ error: 'rowNumbers must be an array' }, { status: 400 });
    }

    await Promise.all(rowNumbers.map(n => markNotificationDone(n)));
    return NextResponse.json({ success: true });
  } catch (error: any) {
    return handleApiError(error, 'Notifications');
  }
}
