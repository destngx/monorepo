import { google } from 'googleapis';
import * as dotenv from 'dotenv';
import * as path from 'path';

// Load .env.local
dotenv.config({ path: path.resolve(process.cwd(), '.env.local') });

async function verifyAuth() {
  console.log('--- Google Sheets Auth Verification ---');
  
  const clientId = process.env.GOOGLE_CLIENT_ID;
  const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
  const refreshToken = process.env.GOOGLE_REFRESH_TOKEN;
  const sheetId = process.env.GOOGLE_SHEETS_ID;

  if (!clientId || !clientSecret || !refreshToken) {
    console.error('❌ Missing credentials in .env.local');
    console.log('Please run `pnpm run auth:setup` to configure them.');
    process.exit(1);
  }

  const oAuth2Client = new google.auth.OAuth2(
    clientId,
    clientSecret,
    'http://127.0.0.1:3000'
  );

  oAuth2Client.setCredentials({
    refresh_token: refreshToken,
  });

  try {
    console.log('🔄 Attempting to refresh access token...');
    const { token } = await oAuth2Client.getAccessToken();
    
    if (token) {
      console.log('✅ Access token obtained successfully!');
      
      const sheets = google.sheets({ version: 'v4', auth: oAuth2Client });
      
      if (sheetId) {
        console.log(`🔄 Verifying access to spreadsheet: ${sheetId}...`);
        await sheets.spreadsheets.get({ spreadsheetId: sheetId });
        console.log('✅ Successfully connected to Google Sheets!');
      } else {
        console.warn('⚠️ GOOGLE_SHEETS_ID not found in .env.local. Skipping sheet verification.');
      }
      
      console.log('\n✨ Your Google Sheets integration is healthy!');
    } else {
      throw new Error('No token returned');
    }
  } catch (error: any) {
    console.error('\n❌ Authentication failed:', error.message);
    if (error.message?.includes('invalid_grant')) {
      console.error('\n💡 REASON: Your refresh token has expired or been revoked.');
      console.log('   - If your GCP project is in "Testing" mode, this happens every 7 days.');
      console.log('   - Run `pnpm run auth:setup` to get a new token.');
    }
    process.exit(1);
  }
}

verifyAuth();
