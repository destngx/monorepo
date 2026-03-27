import { config } from 'dotenv';

// Load environment variables from .env.local file
config({ path: '.env.local' });

export const env = {
  // Redis
  redis: {
    url: process.env.UPSTASH_REDIS_REST_URL,
    token: process.env.UPSTASH_REDIS_REST_TOKEN,
  },

  // AI Models
  ai: {
    openaiApiKey: process.env.OPENAI_API_KEY,
    googleGenAiApiKey: process.env.GOOGLE_GENERATIVE_AI_API_KEY,
    anthropicApiKey: process.env.ANTHROPIC_API_KEY,
    githubToken: process.env.GITHUB_TOKEN,
  },

  // Google Sheets
  sheets: {
    id: process.env.GOOGLE_SHEETS_ID,
    clientId: process.env.GOOGLE_CLIENT_ID,
    clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    refreshToken: process.env.GOOGLE_REFRESH_TOKEN,
  },

  // Data Sources
  dataSources: {
    vnstockServerUrl: process.env.VNSTOCK_SERVER_URL || 'http://localhost:8000',
  },

  // Search APIs
  search: {
    tavilyApiKey: process.env.TAVILY_API_KEY,
    exaApiKey: process.env.EXA_API_KEY,
  },
};
console.log('Loaded environment variables:');
console.log({ env });

export const isProduction = process.env.NODE_ENV === 'production';
