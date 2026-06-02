import { useMemo } from 'react';
import { Button, Drawer, Form, Input, Radio, Typography } from 'antd';
import type { FormInstance } from 'antd';
import type { NameFilterItem } from './types';
import { generateOperators } from './types';
import NameFilterEditor, { FilterConditionLabel } from './NameFilterEditor';
import I18nInput from '@/components/I18nInput';

const { Text } = Typography;

export interface AddChildGroupDrawerProps {
  open: boolean;
  form: FormInstance<{ name_i18n?: Record<string, string>; matchingMode: 'deviceName' | 'lac' | 'tac'; tacRag: string }>;
  /** 上级（一级）分组名称，只读展示，自动填充。 */
  parentGroupName?: string;
  matchingMode: string | undefined;
  nameFilters: NameFilterItem[];
  onClose: () => void;
  onSave: () => void;
  onMatchingModeChange: () => void;
  onAddFilter: () => void;
  onRemoveFilter: (id: string) => void;
  onUpdateFilter: (id: string, field: keyof NameFilterItem, value: string) => void;
  t: (id: string, values?: Record<string, string | number>) => string;
}

/**
 * 新增 L2（子）分组 Drawer。拆分自 GroupDialogs.tsx 以满足单文件 ≤ 400 行。
 */
export default function AddChildGroupDrawer({
  open,
  form,
  parentGroupName,
  matchingMode,
  nameFilters,
  onClose,
  onSave,
  onMatchingModeChange,
  onAddFilter,
  onRemoveFilter,
  onUpdateFilter,
  t,
}: AddChildGroupDrawerProps) {
  // Preview condition description for add child form
  const previewText = useMemo(() => {
    if (matchingMode === 'deviceName') {
      return generateOperators({ matchingMode: 'deviceName', nameRuleList: nameFilters }, t);
    }
    return '';
  }, [matchingMode, nameFilters, t]);

  return (
    <Drawer
      title={t('device.addChildGroup')}
      open={open}
      onClose={onClose}
      width={520}
      destroyOnHidden
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={onSave}>
            {t('common.confirm')}
          </Button>
        </div>
      }
    >
      <Form form={form} layout="vertical">
        {/* Basic info */}
        <div style={{
          padding: '16px',
          background: 'var(--color-fill-quaternary)',
          borderRadius: 8,
          marginBottom: 16,
        }}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('device.basicInfo')}
          </div>
          <Form.Item label={t('device.superiorGroup')} style={{ marginBottom: 12 }}>
            <Input value={parentGroupName ?? ''} disabled />
          </Form.Item>
          <Form.Item label={t('device.groupNameLabel')} required style={{ marginBottom: 0 }}>
            <I18nInput name="name_i18n" required maxLength={50} />
          </Form.Item>
        </div>

        {/* Match rule */}
        <div style={{
          padding: '16px',
          background: 'var(--color-fill-quaternary)',
          borderRadius: 8,
        }}>
          <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('device.matchRule')}
          </div>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
            {t('device.matchRuleDesc')}
          </Text>
          <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} style={{ marginBottom: 12 }}>
            <Radio.Group onChange={onMatchingModeChange}>
              <Radio value="deviceName">{t('device.rules.deviceName')}</Radio>
              <Radio value="lac">LAC</Radio>
              <Radio value="tac">TAC</Radio>
            </Radio.Group>
          </Form.Item>

          {/* Device name filter conditions */}
          {matchingMode === 'deviceName' && (
            <>
              <Form.Item label={<FilterConditionLabel t={t} />} style={{ marginBottom: 0 }}>
                <NameFilterEditor
                  filters={nameFilters}
                  onAdd={onAddFilter}
                  onRemove={onRemoveFilter}
                  onUpdate={onUpdateFilter}
                  t={t}
                />
              </Form.Item>

              {/* Preview condition description */}
              {previewText && (
                <div
                  style={{
                    marginTop: 12,
                    color: 'var(--color-text-tertiary)',
                    fontSize: 12,
                    padding: '8px 12px',
                    background: 'var(--color-bg-container)',
                    borderRadius: 4,
                    wordBreak: 'break-all',
                    border: '1px solid var(--color-border-secondary)',
                  }}
                >
                  {previewText}
                </div>
              )}
            </>
          )}

          {/* TAC/LAC input */}
          {(matchingMode === 'tac' || matchingMode === 'lac') && (
            <Form.Item
              name="tacRag"
              label={matchingMode === 'tac' ? 'TAC' : 'LAC'}
              style={{ marginBottom: 0 }}
              extra={
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('device.rules.formatRange', { range: '0-65535' })}
                </Text>
              }
            >
              <Input placeholder="eg: 1,2,3,1-3" maxLength={50} />
            </Form.Item>
          )}
        </div>
      </Form>
    </Drawer>
  );
}
