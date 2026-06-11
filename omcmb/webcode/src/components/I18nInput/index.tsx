import { Form, Input } from 'antd';
import type { Rule } from 'antd/es/form';
import { AddonInput } from '@/components/common/InputAddon';
import { useT } from '@/hooks/useT';

/**
 * I18nInput — 在 antd Form 里同时录入中文 + 英文 文案。
 *
 * 后端字段为 JSONB `{ 'zh-CN': '...', 'en-US': '...' }`,前端表单 namePath
 * 用 `[name, 'zh-CN']` / `[name, 'en-US']` 双子段,Form 自动序列化为嵌套对象。
 *
 * 中文为必填(主标识),英文 optional;英文留空显示时自动 fallback 到中文。
 *
 * 用法:
 *   <Form.Item label={t('groupManagement.groupName')} required>
 *     <I18nInput name="name_i18n" required maxLength={128} />
 *   </Form.Item>
 *
 * 也可直接当 Form.Item 用(自动渲染外层 label):
 *   <I18nInput name="name_i18n" label="名称" required maxLength={128} />
 */
interface I18nInputProps {
  /** Form item 字段名前缀,会展开为 [name, 'zh-CN'] / [name, 'en-US'] */
  name: string;
  /** 外层 label 文案;省略时不渲染外层 Form.Item */
  label?: string;
  /** 中文输入是否必填 (默认 true) */
  required?: boolean;
  /** Input maxLength */
  maxLength?: number;
  /** 是否 TextArea 模式 */
  textarea?: boolean;
  /** TextArea 行数 */
  rows?: number;
  /** 自定义占位符 */
  placeholder?: { zh?: string; en?: string };
  /** 额外的中文 / 英文校验规则 */
  rules?: { zh?: Rule[]; en?: Rule[] };
}

export function I18nInput({
  name,
  label,
  required = true,
  maxLength,
  textarea = false,
  rows = 2,
  placeholder,
  rules,
}: I18nInputProps): JSX.Element {
  const t = useT();
  const InputComp = textarea ? Input.TextArea : Input;
  const inputCommonProps = textarea ? { rows, maxLength } : { maxLength };

  const zhRules: Rule[] = [
    ...(required ? [{ required: true, message: t('common.required') }] : []),
    ...(rules?.zh ?? []),
  ];
  const enRules = rules?.en ?? [];

  const body = (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Form.Item name={[name, 'zh-CN']} noStyle rules={zhRules}>
        {textarea ? (
          <InputComp
            {...inputCommonProps}
            placeholder={placeholder?.zh ?? t('i18nInput.placeholderZh')}
            prefix="中"
          />
        ) : (
          <AddonInput
            addonBefore="中"
            maxLength={maxLength}
            placeholder={placeholder?.zh ?? t('i18nInput.placeholderZh')}
          />
        )}
      </Form.Item>
      <Form.Item name={[name, 'en-US']} noStyle rules={enRules}>
        {textarea ? (
          <InputComp
            {...inputCommonProps}
            placeholder={placeholder?.en ?? t('i18nInput.placeholderEn')}
            prefix="EN"
          />
        ) : (
          <AddonInput
            addonBefore="EN"
            maxLength={maxLength}
            placeholder={placeholder?.en ?? t('i18nInput.placeholderEn')}
          />
        )}
      </Form.Item>
    </div>
  );

  if (label) {
    return (
      <Form.Item label={label} required={required}>
        {body}
      </Form.Item>
    );
  }
  return body;
}

export default I18nInput;
