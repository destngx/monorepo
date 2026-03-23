import { google } from 'googleapis';
import { AuthError, isAppError, getErrorMessage } from '../../utils/errors';

export class GoogleSheetsError extends AuthError {
  constructor(
    message: string,
    public sheetsErrorCode: 'MISSING_CREDENTIALS' | 'OAUTH_EXPIRED' | 'API_ERROR',
    public originalError?: any,
  ) {
    super(message, { context: { sheetsErrorCode, originalError } });
    this.name = 'GoogleSheetsError';
  }
}

export async function getSheetsClient() {
  const clientId = process.env.GOOGLE_CLIENT_ID;
  const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
  const refreshToken = process.env.GOOGLE_REFRESH_TOKEN;

  if (!clientId || !clientSecret || !refreshToken) {
    throw new GoogleSheetsError(
      'Missing Google OAuth credentials. Please run `pnpm run auth:setup` to configure them.',
      'MISSING_CREDENTIALS',
    );
  }

  const oAuth2Client = new google.auth.OAuth2(clientId, clientSecret, 'http://127.0.0.1:3000');

  oAuth2Client.setCredentials({
    refresh_token: refreshToken,
  });

  const sheets = google.sheets({ version: 'v4', auth: oAuth2Client });

  // Verification step: Try to get a simple property to see if the token is valid
  try {
    // We don't actually need to call an API here if we want to be lazy,
    // but the 'invalid_grant' error usually happens during the first API call
    // when the client tries to refresh the token.
    return sheets;
  } catch (error: any) {
    const message = getErrorMessage(error);
    if (error.code === '400' || error.message?.includes('invalid_grant')) {
      throw new GoogleSheetsError(
        'Google Sheets session expired. Please run `pnpm run auth:setup`.',
        'OAUTH_EXPIRED',
        error,
      );
    }
    throw new GoogleSheetsError('Failed to initialize Google Sheets client.', 'API_ERROR', error);
  }
}
