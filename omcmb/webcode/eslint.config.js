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
  // ── i18n 防腐 guard（#226）──────────────────────────────────────────────
  // 禁止在 UI 层（.tsx 页面/组件）出现中文字符串字面量 / JSX 文本 / 模板串，
  // 强制所有用户可见文案走 react-intl（t('...') / formatMessage），从结构上
  // 阻止「硬编码文案不随语言切换」这类回归（#226 的根因是缺少这道闸）。
  //
  // 阶段化推进：当前以 'warn' 起步——只对新增代码冒提示，不阻断在跑的特性流；
  // 待 Phase 2/3 把存量 ~650 处高信号文案迁完、清零告警后，本规则翻 'error'
  // （连同 CI gate），届时任何新硬编码中文都会直接失败。
  //
  // 作用域只限 .tsx（UI 层）；i18n 词典本身、测试夹具、mock 演示数据按设计豁免
  // （value 枚举常量 / 演示数组不是用户切语言时看到的可翻译文案）。
  {
    files: ['src/**/*.tsx'],
    ignores: [
      'src/**/*.test.tsx',
      'src/**/__tests__/**',
      'src/**/*.stories.tsx',
      'src/mock/**',
      'src/**/mock/**',
    ],
    rules: {
      'no-restricted-syntax': [
        'warn',
        {
          selector: 'Literal[value=/[\\u4e00-\\u9fff]/]',
          message:
            '禁止硬编码中文字符串字面量：用户可见文案请走 react-intl，t(\'module.key\') / formatMessage（i18n guard #226）。',
        },
        {
          selector: 'JSXText[value=/[\\u4e00-\\u9fff]/]',
          message:
            '禁止在 JSX 中硬编码中文文本：请改为 {t(\'module.key\')}（i18n guard #226）。',
        },
        {
          selector: 'TemplateElement[value.cooked=/[\\u4e00-\\u9fff]/]',
          message:
            '禁止在模板字符串里硬编码中文：请用带插值的 ICU message，t(\'module.key\', { ... })（i18n guard #226）。',
        },
      ],
    },
  },
])
