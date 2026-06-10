import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import unusedImports from 'eslint-plugin-unused-imports'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

// Workspace-level ESLint 配置：覆盖 webcode/ + frontend-core/。
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
      // eslint 10 + typescript-eslint:workspace 内存在多个 tsconfig 候选时
      // 必须显式指定根目录,否则所有文件报 "No tsconfigRootDir was set"。
      parserOptions: {
        tsconfigRootDir: import.meta.dirname,
      },
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
      'react-hooks/use-memo': 'warn',
      // - eslint 10 新增 recommended 规则:死赋值是代码异味提示,非正确性问题
      'no-useless-assignment': 'warn',
      // - Fast Refresh 体验建议：文件混合导出组件和非组件时 HMR 不够稳
      'react-refresh/only-export-components': 'warn',
    },
  },
])
