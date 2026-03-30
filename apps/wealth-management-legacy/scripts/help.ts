// scripts/help.ts

/* eslint-disable-next-line no-console */
console.log('\n─────────────────────────────────────────────────────────────');
/* eslint-disable-next-line no-console */
console.log('🤖 WealthOS - Available Commands');
/* eslint-disable-next-line no-console */
console.log('─────────────────────────────────────────────────────────────\n');

/* eslint-disable-next-line no-console */
console.log('📦 Core Development');
/* eslint-disable-next-line no-console */
console.log('  pnpm run dev      Start the local development server (Next.js & Turbopack)');
/* eslint-disable-next-line no-console */
console.log('  pnpm run build    Build the application for production deployment');
/* eslint-disable-next-line no-console */
console.log('  pnpm run start    Start the production server after building');
/* eslint-disable-next-line no-console */
console.log('  pnpm run lint     Run the linter to catch code style issues');
/* eslint-disable-next-line no-console */
console.log('  pnpm run test     Run unit tests via Vitest');
/* eslint-disable-next-line no-console */
console.log('');

/* eslint-disable-next-line no-console */
console.log('🔐 Authentication & Setup');
/* eslint-disable-next-line no-console */
console.log('  pnpm run auth:setup   Set up Google Sheets OAuth (Database access)');
/* eslint-disable-next-line no-console */
console.log('                        - Used for fetching Account/Budget/Transaction data.');
/* eslint-disable-next-line no-console */
console.log('                        - Requires Google Cloud Console app credentials.');
/* eslint-disable-next-line no-console */
console.log('');
/* eslint-disable-next-line no-console */
console.log('  pnpm run auth:ai      Set up AI Provider Keys or GitHub Copilot OAuth');
/* eslint-disable-next-line no-console */
console.log('                        - Used exclusively for the AI Chat Assistant.');
/* eslint-disable-next-line no-console */
console.log('                        - Supports GitHub Models, OpenAI, Gemini, Claude.');
/* eslint-disable-next-line no-console */
console.log('');

/* eslint-disable-next-line no-console */
console.log('🛠️ Utilities');
/* eslint-disable-next-line no-console */
console.log('  pnpm run tx           Add a new transaction via the Terminal CLI');
/* eslint-disable-next-line no-console */
console.log('                        - Allows you to quickly record spending without the UI.');
/* eslint-disable-next-line no-console */
console.log('');
/* eslint-disable-next-line no-console */
console.log('  pnpm run help         Show this help message');
/* eslint-disable-next-line no-console */
console.log('\n─────────────────────────────────────────────────────────────\n');
