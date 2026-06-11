/**
 * InputAddon 套件 —— 替代 antd6 已废弃的 `Input`/`InputNumber` 的
 * `addonBefore` / `addonAfter`。
 *
 * - InputAddon：复刻 `.ant-input-group-addon` 的静态前/后缀灰盒（灰底
 *   colorFillTertiary + 边框 + 居中文案 + paddingSM），放进 Space.Compact。
 * - AddonInput / AddonInputNumber：受控薄包装。Form.Item 通过 cloneElement 把
 *   value/onChange 注入「直接子节点」，若直接套 Space.Compact 会注入到 div 上导致
 *   失绑。这两个包装接住注入的受控属性并转发给内层 Input/InputNumber，外层用
 *   Space.Compact + InputAddon 拼前/后缀，绑定与外观都不变。
 *
 * 用法：
 *   // 非受控（自己管 value/onChange）——直接拼即可：
 *   <Space.Compact><InputAddon>实例号</InputAddon><InputNumber ... /></Space.Compact>
 *   // 受控于 Form.Item：
 *   <Form.Item name="x"><AddonInputNumber addon="天" addonPos="after" ... /></Form.Item>
 */
import type { ReactNode } from 'react';
import { Input, InputNumber, Space, theme } from 'antd';

interface InputAddonProps {
  children: ReactNode;
}

export function InputAddon({ children }: InputAddonProps) {
  const { token } = theme.useToken();
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: `0 ${token.paddingSM}px`,
        color: token.colorText,
        fontWeight: 'normal',
        textAlign: 'center',
        whiteSpace: 'nowrap',
        background: token.colorFillTertiary,
        border: `${token.lineWidth}px ${token.lineType} ${token.colorBorder}`,
      }}
    >
      {children}
    </span>
  );
}

interface AddonWrapExtra {
  /** 前缀盒子内容（对应原 addonBefore） */
  addonBefore?: ReactNode;
  /** 后缀盒子内容（对应原 addonAfter） */
  addonAfter?: ReactNode;
  /** 外层 Space.Compact 样式（默认占满宽度） */
  compactStyle?: React.CSSProperties;
}

/** 受控 Input + 前/后缀盒子（供 Form.Item 注入 value/onChange）。 */
export function AddonInput({
  addonBefore,
  addonAfter,
  compactStyle,
  ...rest
}: AddonWrapExtra & React.ComponentProps<typeof Input>) {
  return (
    <Space.Compact style={{ width: '100%', ...compactStyle }}>
      {addonBefore != null && <InputAddon>{addonBefore}</InputAddon>}
      <Input {...rest} />
      {addonAfter != null && <InputAddon>{addonAfter}</InputAddon>}
    </Space.Compact>
  );
}

/** 受控 InputNumber + 前/后缀盒子（供 Form.Item 注入 value/onChange）。 */
export function AddonInputNumber({
  addonBefore,
  addonAfter,
  compactStyle,
  ...rest
}: AddonWrapExtra & React.ComponentProps<typeof InputNumber>) {
  return (
    <Space.Compact style={{ width: '100%', ...compactStyle }}>
      {addonBefore != null && <InputAddon>{addonBefore}</InputAddon>}
      <InputNumber {...rest} />
      {addonAfter != null && <InputAddon>{addonAfter}</InputAddon>}
    </Space.Compact>
  );
}

export default InputAddon;
