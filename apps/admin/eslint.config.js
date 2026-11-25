// ETHER Admin — ESLint flat config, Phase 01 skeleton.
// Versions pinned in package.json (eslint 9.28.0, typescript-eslint 8.33.0,
// @eslint/js 9.28.0); pending CI verification. No other plugins are added
// in Phase 01; no formatting rules here — formatting is Prettier's job.
// Admin linting is identical to Web: no weaker checking for admin code.
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
);
