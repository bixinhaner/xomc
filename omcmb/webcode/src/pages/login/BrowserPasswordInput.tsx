import { useState } from 'react';
import { Input } from 'antd';
import type { InputProps } from 'antd';
import { EyeInvisibleOutlined, EyeOutlined } from '@ant-design/icons';
import styles from './Login.module.css';

interface BrowserPasswordInputProps extends InputProps {
  showPasswordLabel: string;
  hidePasswordLabel: string;
}

function supportsWebkitTextSecurity(): boolean {
  return typeof CSS !== 'undefined'
    && typeof CSS.supports === 'function'
    && CSS.supports('-webkit-text-security', 'disc');
}

/**
 * “禁止浏览器自动记录密码”开启时使用的登录输入框。
 *
 * Chromium 只要看到原生 password 字段，就可能忽略 autocomplete 并弹出保存提示。
 * 目标浏览器支持 -webkit-text-security 时改用普通文本字段做视觉掩码；密码值仍由
 * Ant Design Form 受控并直接提交，不复制到存储、dataset 或隐藏字段。
 * 不支持该 CSS 能力的浏览器回退原生密码框，避免明文显示。
 */
export function BrowserPasswordInput({
  showPasswordLabel,
  hidePasswordLabel,
  ...props
}: BrowserPasswordInputProps) {
  const [visible, setVisible] = useState(false);

  if (!supportsWebkitTextSecurity()) {
    return <Input.Password {...props} autoComplete="off" />;
  }

  return (
    <Input
      {...props}
      id="login_credential"
      type="text"
      autoComplete="off"
      spellCheck={false}
      classNames={{
        input: visible ? styles.revealedPasswordInput : styles.maskedPasswordInput,
      }}
      suffix={visible ? (
        <button
          type="button"
          className={styles.passwordVisibilityToggle}
          aria-label={hidePasswordLabel}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => setVisible(false)}
        >
          <EyeOutlined />
        </button>
      ) : (
        <button
          type="button"
          className={styles.passwordVisibilityToggle}
          aria-label={showPasswordLabel}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => setVisible(true)}
        >
          <EyeInvisibleOutlined />
        </button>
      )}
    />
  );
}
