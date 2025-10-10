const { createGlobPatternsForDependencies } = require('@nx/react/tailwind');
const { join } = require('path');

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    join(__dirname, '{src,pages,components,app}/**/*!(*.stories|*.spec).{ts,tsx,html}'),
    ...createGlobPatternsForDependencies(__dirname),
  ],
  theme: {
    extend: {
      colors: {
        background: '#F5F5F5',
        foreground: '#003366',
        primary: '#1C8558',
        accent: '#FF6F61',
        border: '#E5E7EB',
        'muted-foreground': '#6B7280',
      },
    },
  },
  plugins: [],
};
