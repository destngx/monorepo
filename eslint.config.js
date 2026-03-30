import js from '@eslint/js';
import ts from 'typescript-eslint';
import sveltePlugin from 'eslint-plugin-svelte';
import prettierConfig from 'eslint-config-prettier';

export default [
  // Global ignores
  {
    ignores: [
      'node_modules/**',
      'dist/**',
      'build/**',
      '.next/**',
      '.svelte-kit/**',
      'coverage/**',
      '.nx/**',
      '.pytest_cache/**',
      '.venv/**',
      '**/__generated__/**',
      // Legacy/reference surfaces not in active scope
      'apps/cloudinary-photos-app/**',
      'apps/portfolio-landpage/**',
      'apps/wealth-management-legacy/**',
      'libs/cloudinary-photos-app/**',
      'libs/portfolio-landpage/**',
      'libs/wealth-management-legacy/**',
    ],
  },
  // JavaScript base
  js.configs.recommended,
  // TypeScript for .ts/.tsx files only
  ...ts.configs.recommendedTypeChecked.map((config) => ({
    ...config,
    files: ['**/*.ts', '**/*.tsx'],
  })),
  // TypeScript parser config
  {
    files: ['**/*.ts', '**/*.tsx'],
    languageOptions: {
      parserOptions: {
        project: './tsconfig.base.json',
        tsconfigRootDir: process.cwd(),
      },
    },
  },
  // Svelte
  ...sveltePlugin.configs['flat/prettier'],
  // Browser + Node globals
  {
    languageOptions: {
      globals: {
        // Node.js
        process: 'readonly',
        // Browser globals
        fetch: 'readonly',
        Request: 'readonly',
        Response: 'readonly',
        localStorage: 'readonly',
        sessionStorage: 'readonly',
        console: 'readonly',
        // SvelteKit
        string: 'readonly',
      },
    },
  },
  // Global rules
  {
    files: ['**/*.{js,jsx,ts,tsx,mjs,svelte}'],
    rules: {
      'max-len': ['warn', { code: 120 }],
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'spaced-comment': ['warn', 'always', { exceptions: ['/'] }],
      'object-curly-spacing': 'off',
      indent: ['warn', 2, { ignoredNodes: ['PropertyDefinition'] }],
      'new-cap': 'off',
      'require-jsdoc': 'off',
      'no-unused-vars': 'warn',
    },
  },
  // TypeScript-specific rules
  {
    files: ['**/*.ts', '**/*.tsx'],
    rules: {
      '@typescript-eslint/naming-convention': [
        'warn',
        { selector: 'variable', modifiers: ['destructured'], format: null },
        { selector: 'interface', format: ['PascalCase'], custom: { regex: '^I[A-Z]', match: false } },
        {
          selector: 'variable',
          types: ['boolean'],
          format: ['PascalCase'],
          prefix: ['is', 'should', 'has', 'can', 'did', 'will'],
        },
        { selector: 'memberLike', modifiers: ['private'], format: ['camelCase'], leadingUnderscore: 'require' },
      ],
      '@typescript-eslint/no-unsafe-argument': 'warn',
      '@typescript-eslint/no-unsafe-assignment': 'warn',
      '@typescript-eslint/no-unsafe-call': 'warn',
      '@typescript-eslint/no-unsafe-member-access': 'warn',
      '@typescript-eslint/no-unsafe-return': 'warn',
      '@typescript-eslint/no-unused-vars': 'warn',
      '@typescript-eslint/no-misused-promises': 'warn',
      '@typescript-eslint/restrict-plus-operands': 'warn',
      '@typescript-eslint/no-floating-promises': 'warn',
      '@typescript-eslint/no-empty-function': 'warn',
      '@typescript-eslint/no-redundant-type-constituents': 'warn',
      '@typescript-eslint/require-await': 'warn',
      'no-constant-condition': 'warn',
    },
  },
  // Test globals
  {
    files: ['**/*.{spec,test}.{ts,tsx,js,jsx}'],
    languageOptions: {
      globals: {
        describe: 'readonly',
        it: 'readonly',
        expect: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
        beforeAll: 'readonly',
        afterAll: 'readonly',
        vi: 'readonly',
      },
    },
  },
  // Prettier last
  prettierConfig,
];
