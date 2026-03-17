import { NextResponse } from 'next/server';
import { getGoals } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const forceFresh = searchParams.get('force') === 'true';

    const goals = await getGoals(forceFresh);
    return NextResponse.json(goals);
  } catch (error) {
    return handleApiError(error, 'Goals');
  }
}
