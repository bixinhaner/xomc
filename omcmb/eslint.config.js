import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

// Workspace-level ESLint 配置：覆盖 webcode/ + frontend-core/（以及未来 webcode-vN/）。
// webcode/eslint.config.js 仍然保留，供在 webcode/ 子目录下单独跑 lint。
export default defineConfig([
  globalIgnores([
    '**/dist',
    '**/node_modules',
    '**/.tmp',
    '**/.vite',
  ]),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
  },
])
