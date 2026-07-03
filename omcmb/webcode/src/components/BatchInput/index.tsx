import React, { useState } from 'react';
import { Alert, Modal, Space, Tag, Typography } from 'antd';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

export interface BatchInputProps {
  visible: boolean;
  onOk: (sns: string[]) => void;
  onCancel: () => void;
  title?: string;
  validateFn?: (sn: string) => boolean;
  validationMessage?: string;
  maxCount?: number;
}

const BatchInput: React.FC<BatchInputProps> = ({
  visible,
  onOk,
  onCancel,
  title,
  validateFn,
  validationMessage,
  maxCount,
}) => {
  const t = useT();
  const [inputText, setInputText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const token = useThemeToken();

  const resolvedTitle = title ?? t('batchInput.title');
  const resolvedValidationMessage = validationMessage ?? t('batchInput.invalidFormat');

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
      const samples = `${invalid.slice(0, 5).join(', ')}${invalid.length > 5 ? '...' : ''}`;
      setError(
        t('batchInput.invalidList', {
          count: invalid.length,
          message: resolvedValidationMessage,
          samples,
        }),
      );
      return;
    }
    if (maxCount && unique.length > maxCount) {
      setError(t('batchInput.exceedMax', { max: maxCount, current: unique.length }));
      return;
    }
    if (unique.length === 0) {
      setError(t('batchInput.emptyInput'));
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
      title={resolvedTitle}
      open={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      destroyOnHidden
      width={520}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Typography.Text type="secondary" style={{ fontSize: 13 }}>
          {t('batchInput.hint')}
          {maxCount && t('batchInput.hintMax', { max: maxCount })}
        </Typography.Text>

        <textarea
          value={inputText}
          onChange={(e) => {
            setInputText(e.target.value);
            setError(null);
          }}
          placeholder={t('batchInput.placeholder')}
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
          <Alert type="error" title={error} showIcon style={{ fontSize: 13 }} />
        )}

        {/* Preview */}
        {unique.length > 0 && (
          <div>
            <Space wrap>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('batchInput.parseResult')}
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
                <Tag style={{ fontSize: 12 }}>{t('batchInput.moreCount', { count: unique.length - 10 })}</Tag>
              )}
            </Space>
            <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
              {t('batchInput.totalCount', { count: unique.length })}
              {parsed.length !== unique.length ? t('batchInput.dedupSuffix', { origin: parsed.length }) : ''}
            </Typography.Text>
          </div>
        )}
      </div>
    </Modal>
  );
};

export default BatchInput;
