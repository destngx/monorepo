// scripts/help.ts

console.log('\n─────────────────────────────────────────────────────────────');
console.log('🤖 WealthOS - Available Commands');
console.log('─────────────────────────────────────────────────────────────\n');

console.log('📦 Core Development');
console.log('  pnpm run dev      Start the local development server (Next.js & Turbopack)');
console.log('  pnpm run build    Build the application for production deployment');
console.log('  pnpm run start    Start the production server after building');
console.log('  pnpm run lint     Run the linter to catch code style issues');
console.log('  pnpm run test     Run unit tests via Vitest');
console.log('');

console.log('🔐 Authentication & Setup');
console.log('  pnpm run auth:setup   Set up Google Sheets OAuth (Database access)');
console.log('                        - Used for fetching Account/Budget/Transaction data.');
console.log('                        - Requires Google Cloud Console app credentials.');
console.log('');
console.log('  pnpm run auth:ai      Set up AI Provider Keys or GitHub Copilot OAuth');
console.log('                        - Used exclusively for the AI Chat Assistant.');
console.log('                        - Supports GitHub Models, OpenAI, Gemini, Claude.');
console.log('');

console.log('🛠️ Utilities');
console.log('  pnpm run tx           Add a new transaction via the Terminal CLI');
console.log('                        - Allows you to quickly record spending without the UI.');
console.log('');
console.log('  pnpm run help         Show this help message');
console.log('\n─────────────────────────────────────────────────────────────\n');
