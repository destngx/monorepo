import { google } from 'googleapis';
import * as readline from 'readline';
import * as fs from 'fs';
import * as path from 'path';
import * as dotenv from 'dotenv';
import { URL } from 'url';

// Load existing .env or .env.local
const envPath = path.resolve(process.cwd(), 'apps/wealth-management/.env.local');
dotenv.config({ path: envPath });

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

const question = (query: string): Promise<string> => new Promise((resolve) => rl.question(query, resolve));

async function main() {
  /* eslint-disable-next-line no-console */
  console.log('--- Google OAuth Setup CLI ---');
  /* eslint-disable-next-line no-console */
  console.log('\n> [!] IMPORTANT: If your Google Cloud project is in "Testing" mode,');
  /* eslint-disable-next-line no-console */
  console.log('> tokens expire every 7 days. To get a permanent token, go to:');
  /* eslint-disable-next-line no-console */
  console.log('> https://console.cloud.google.com/apis/credentials/consent');
  /* eslint-disable-next-line no-console */
  console.log('> and set the publishing status to "In Production".\n');

  let clientId = process.env.GOOGLE_CLIENT_ID;
  let clientSecret = process.env.GOOGLE_CLIENT_SECRET;

  if (!clientId || !clientSecret) {
    /* eslint-disable-next-line no-console */
    console.log('Missing GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET in .env.local.');
    clientId = await question('Enter your Google Client ID: ');
    clientSecret = await question('Enter your Google Client Secret: ');
  }

  // Use localhost as the redirect URI
  const redirectUri = 'http://127.0.0.1:3000';

  const oAuth2Client = new google.auth.OAuth2(clientId, clientSecret, redirectUri);

  const authUrl = oAuth2Client.generateAuthUrl({
    access_type: 'offline', // IMPORTANT to get a refresh token
    prompt: 'consent', // Force consent so refresh token is always returned
    scope: ['https://www.googleapis.com/auth/spreadsheets'],
  });

  /* eslint-disable-next-line no-console */
  console.log('\n1. Please visit the following URL to authorize this application:');
  /* eslint-disable-next-line no-console */
  console.log('\n', authUrl, '\n');
  /* eslint-disable-next-line no-console */
  console.log(
    `2. After authorizing, you will be redirected to a URL that starts with ${redirectUri} and looks like it's broken.`,
  );
  /* eslint-disable-next-line no-console */
  console.log('3. Copy that entire redirected URL and paste it below:\n');

  const redirectedUrl = await question('Redirected URL: ');

  try {
    const url = new URL(redirectedUrl.trim());
    const code = url.searchParams.get('code');

    if (!code) {
      throw new Error('Could not find the "code" parameter in the URL provided.');
    }

    /* eslint-disable-next-line no-console */
    console.log('\nExchanging code for tokens...');
    const { tokens } = await oAuth2Client.getToken(code);

    if (!tokens.refresh_token) {
      /* eslint-disable-next-line no-console */
      console.warn(
        '⚠️ WARNING: No refresh token returned. You might need to revoke access connected to this client and try again.',
      );
    }

    // Prepare .env contents
    let envContents = '';
    if (fs.existsSync(envPath)) {
      envContents = fs.readFileSync(envPath, 'utf8');
    }

    // Helper to replace or append to .env
    const updateEnvVar = (key: string, value: string) => {
      const regex = new RegExp(`^${key}=.*$`, 'm');
      if (regex.test(envContents)) {
        envContents = envContents.replace(regex, `${key}=${value}`);
      } else {
        envContents += `\n${key}=${value}`;
      }
    };

    updateEnvVar('GOOGLE_CLIENT_ID', clientId);
    updateEnvVar('GOOGLE_CLIENT_SECRET', clientSecret);

    if (tokens.refresh_token) {
      updateEnvVar('GOOGLE_REFRESH_TOKEN', tokens.refresh_token);
    }

    // Write back
    fs.writeFileSync(envPath, envContents.trim() + '\n');
    /* eslint-disable-next-line no-console */
    console.log('\n✅ Successfully obtained OAuth tokens!');
    /* eslint-disable-next-line no-console */
    console.log('✅ Updated .env.local with GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REFRESH_TOKEN.');

    /* eslint-disable-next-line no-console */
    console.log('\nNote: You also need an existing Google Sheet ID.');
    if (!process.env.GOOGLE_SHEETS_ID) {
      /* eslint-disable-next-line no-console */
      console.log('Please set GOOGLE_SHEETS_ID in your .env.local as well.');
    } else {
      /* eslint-disable-next-line no-console */
      console.log(`GOOGLE_SHEETS_ID is currently set to: ${process.env.GOOGLE_SHEETS_ID}`);
    }
  } catch (err: any) {
    /* eslint-disable-next-line no-console */
    console.error('\n❌ Error authenticating:', err.message);
  } finally {
    rl.close();
  }
}

/* eslint-disable-next-line no-console */
void main();
