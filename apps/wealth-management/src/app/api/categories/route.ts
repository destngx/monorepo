import { NextResponse } from 'next/server';
import { getCategories } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET() {
  try {
    const categories = await getCategories();
    return NextResponse.json(categories);
  } catch (error) {
    return handleApiError(error, 'Categories');
  }
}
