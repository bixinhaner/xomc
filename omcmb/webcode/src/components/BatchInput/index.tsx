import React, { useState } from 'react';
import { Alert, Modal, Space, Tag, Typography } from 'antd';
import { useThemeToken } from '@/hooks/useThemeToken';

export interface BatchInputProps {
  visible: boolean;
  onOk: (sns: string[]) => void;
  onCancel: () => void;
  title?: string;
  validateFn?: (sn: string) => boolean;
  validationMessage?: string;
  maxCount?: number;
}

const DEFAULT_PLACEHOLDER = '输入SN，每行一个或用分号/空格分隔\n例如：\nSN000001\nSN000002;SN000003\nSN000004 SN000005';

const BatchInput: React.FC<BatchInputProps> = ({
  visible,
  onOk,
  onCancel,
  title = '批量输入设备SN',
  validateFn,
  validationMessage = 'SN格式不正确',
  maxCount,
}) => {
  const [inputText, setInputText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const token = useThemeToken();

  const parseInput = (text: string): string[] => {
    return text
      .split(/[\n;,\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
  };

  const parsed = parseInput(inputText);
  const unique = [...new Set(parsed)];
  const invalid = validateFn ? unique.filter((sn) => !validateFn(sn)) : [];

  const handleOk = () => {
    if (invalid.length > 0) {
      setError(`以下${invalid.length}个SN${validationMessage}：${invalid.slice(0, 5).join(', ')}${invalid.length > 5 ? '...' : ''}`);
      return;
    }
    if (maxCount && unique.length > maxCount) {
      setError(`最多支持 ${maxCount} 个SN，当前输入 ${unique.length} 个`);
      return;
    }
    if (unique.length === 0) {
      setError('请至少输入一个SN');
      return;
    }
    setError(null);
    onOk(unique);
    setInputText('');
  };

  const handleCancel = () => {
    setInputText('');
    setError(null);
    onCancel();
  };

  return (
    <Modal
      title={title}
      open={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="确认"
      cancelText="取消"
      destroyOnClose
      width={520}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Typography.Text type="secondary" style={{ fontSize: 13 }}>
          每行一个SN，或使用分号、空格分隔。重复项将自动去重。
          {maxCount && ` 最多 ${maxCount} 个。`}
        </Typography.Text>

        <textarea
          value={inputText}
          onChange={(e) => {
            setInputText(e.target.value);
            setError(null);
          }}
          placeholder={DEFAULT_PLACEHOLDER}
          rows={8}
          style={{
            width: '100%',
            resize: 'vertical',
            fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace",
            fontSize: 13,
            padding: '8px 12px',
            border: error ? '1px solid #ff4d4f' : `1px solid ${token.colorBorder}`,
            borderRadius: 6,
            outline: 'none',
            lineHeight: 1.6,
            color: token.colorTextHeading,
            background: token.colorBgContainer,
            boxSizing: 'border-box',
          }}
          onFocus={(e) => {
            (e.target as HTMLTextAreaElement).style.borderColor = token.colorPrimary;
            (e.target as HTMLTextAreaElement).style.boxShadow = `0 0 0 2px ${token.colorPrimaryBg}`;
          }}
          onBlur={(e) => {
            (e.target as HTMLTextAreaElement).style.borderColor = error ? '#ff4d4f' : '#d9d9d9';
            (e.target as HTMLTextAreaElement).style.boxShadow = 'none';
          }}
        />

        {error && (
          <Alert type="error" message={error} showIcon style={{ fontSize: 13 }} />
        )}

        {/* Preview */}
        {unique.length > 0 && (
          <div>
            <Space wrap>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                解析结果：
              </Typography.Text>
              {unique.slice(0, 10).map((sn) => (
                <Tag
                  key={sn}
                  color={invalid.includes(sn) ? 'error' : 'blue'}
                  style={{ fontSize: 12, fontFamily: 'monospace' }}
                >
                  {sn}
                </Tag>
              ))}
              {unique.length > 10 && (
                <Tag style={{ fontSize: 12 }}>+{unique.length - 10} 个</Tag>
              )}
            </Space>
            <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
              共 {unique.length} 个{parsed.length !== unique.length ? `（去重后，原始 ${parsed.length} 个）` : ''}
            </Typography.Text>
          </div>
        )}
      </div>
    </Modal>
  );
};

export default BatchInput;
