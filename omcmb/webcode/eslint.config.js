import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
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
      // eslint 10 + typescript-eslint:目录下存在多个 tsconfig 候选时必须显式指定根目录。
      parserOptions: {
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      // ── 对齐 workspace 根配置(omcmb/eslint.config.js)的严重度口径 ──
      // eslint 10 改为按文件就近解析配置,根配置的降级不再覆盖本目录,
      // 故在此复刻同一套「提示/优化建议降为 warning」的规则,保持两种跑法行为一致。
      'react-refresh/only-export-components': 'warn',
      '@typescript-eslint/no-explicit-any': 'warn',
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/preserve-manual-memoization': 'warn',
      'react-hooks/immutability': 'warn',
      'react-hooks/refs': 'warn',
      'react-hooks/purity': 'warn',
      'react-hooks/set-state-in-render': 'warn',
      'react-hooks/static-components': 'warn',
      'react-hooks/component-hook-factories': 'warn',
      'react-hooks/unsupported-syntax': 'warn',
      'react-hooks/use-memo': 'warn',
      'no-useless-assignment': 'warn',
      // 采用项目既有的 `_` 前缀约定：以下划线开头的形参/变量为「有意未用」
      // （满足回调/样式函数等签名契约的占位参数），不视为噪声。
      // ignoreRestSiblings：`const { a, ...rest } = obj` 的剔除式解构，被剔除的键合法。
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
          ignoreRestSiblings: true,
        },
      ],
    },
  },
])
