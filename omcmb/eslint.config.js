import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import unusedImports from 'eslint-plugin-unused-imports'
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
    plugins: {
      'unused-imports': unusedImports,
    },
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // 关闭默认的 no-unused-vars，换成 unused-imports plugin 的可自动修复版本
      '@typescript-eslint/no-unused-vars': 'off',
      'unused-imports/no-unused-imports': 'error',
      'unused-imports/no-unused-vars': [
        'error',
        {
          args: 'all',
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
          destructuredArrayIgnorePattern: '^_',
          ignoreRestSiblings: true,
        },
      ],

      // 以下规则是"提示/优化建议"而非代码正确性问题，降级为 warning 以不阻塞 CI gate：
      // - any 类型需要业务知识替换，逐步治理
      '@typescript-eslint/no-explicit-any': 'warn',
      // - React Compiler 静态分析规则（React 19 + react-hooks 7 新引入），
      //   代码虽不完美但无正确性问题；等 React Compiler 默认启用后再系统性治理
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/preserve-manual-memoization': 'warn',
      'react-hooks/immutability': 'warn',
      'react-hooks/refs': 'warn',
      'react-hooks/purity': 'warn',
      'react-hooks/set-state-in-render': 'warn',
      'react-hooks/static-components': 'warn',
      'react-hooks/component-hook-factories': 'warn',
      'react-hooks/unsupported-syntax': 'warn',
      // - Fast Refresh 体验建议：文件混合导出组件和非组件时 HMR 不够稳
      'react-refresh/only-export-components': 'warn',
    },
  },
])
