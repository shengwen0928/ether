// ETHER Web — ESLint flat config, Phase 01 skeleton.
// Versions pinned in package.json (eslint 9.28.0, typescript-eslint 8.33.0,
// @eslint/js 9.28.0); pending CI verification. No other plugins are added
// in Phase 01; no formatting rules here — formatting is Prettier's job.
import js from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist', 'node_modules'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        document: 'readonly',
        navigator: 'readonly',
        window: 'readonly',
      },
    },
  },
  {
    // Phase 02: Node scripts under tests/ (runner, contrast and audit checks).
    // Scoped to that directory only; app source keeps browser-only globals.
    files: ['tests/**/*.mjs'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        Blob: 'readonly',
        URL: 'readonly',
        URLSearchParams: 'readonly',
        Response: 'readonly',
        console: 'readonly',
        process: 'readonly',
      },
    },
  },
);
