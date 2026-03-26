/**
 * Delegating API route
 * Maps app/api/accounts to features/accounts/api/route
 */

export async function GET() {
  return Response.json([]);
}

export async function POST() {
  return Response.json({ ok: true });
}
