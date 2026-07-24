import { useCallback } from 'react';
import { Button, Drawer, Form, Input, Radio, Select, Switch, Typography } from 'antd';
import type { FormInstance } from 'antd';
import type { NameFilterItem } from './types';
import { generateId } from './types';
import NameFilterEditor, { FilterConditionLabel } from './NameFilterEditor';

const { Text } = Typography;

export interface EditLevel2GroupDrawerProps {
  open: boolean;
  form: FormInstance<{ name?: string; autoAssignEnabled?: boolean; matchingMode: 'deviceName' | 'lac' | 'tac' | 'serialNumber'; tacRag: string; sourceGroupId?: string; serialNumbers?: string }>;
  /** 上级（一级）分组名称，只读展示，自动填充。 */
  parentGroupName?: string;
  autoAssignEnabled: boolean;
  matchingMode: string | undefined;
  nameFilters: NameFilterItem[];
  sourceGroupOptions: Array<{ label: string; value: string }>;
  onClose: () => void;
  onSave: () => void;
  onNameFiltersChange: React.Dispatch<React.SetStateAction<NameFilterItem[]>>;
  t: (id: string, values?: Record<string, string | number>) => string;
}

/**
 * 编辑 L2 分组 Drawer。拆分自 GroupDialogs.tsx 以满足单文件 ≤ 400 行。
 *
 * 与 AddChildGroupDrawer 不同的是 — Edit 路径上 filter 状态是直接通过
 * setState 派发器更新的，没有抽 useCallback handler。这里把内联 lambda
 * 收敛成几个 useCallback 辅助函数，便于阅读。
 */
export default function EditLevel2GroupDrawer({
  open,
  form,
  parentGroupName,
  autoAssignEnabled,
  matchingMode,
  nameFilters,
  sourceGroupOptions,
  onClose,
  onSave,
  onNameFiltersChange,
  t,
}: EditLevel2GroupDrawerProps) {
  const handleMatchingModeChange = useCallback(() => {
    onNameFiltersChange([{ id: generateId(), condition: 'contain', value: '' }]);
    form.setFieldsValue({ tacRag: '', serialNumbers: '' });
  }, [form, onNameFiltersChange]);

  const handleUpdate = useCallback(
    (id: string, field: keyof NameFilterItem, value: string) => {
      onNameFiltersChange((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
    },
    [onNameFiltersChange]
  );

  const handleRemove = useCallback(
    (id: string) => {
      onNameFiltersChange((prev) => {
        if (prev.length <= 1) return prev;
        const newFilters = prev.filter((f) => f.id !== id);
        if (newFilters.length > 0 && newFilters[0].andOr !== undefined) {
          const { andOr: _andOr, ...rest } = newFilters[0];
          void _andOr;
          newFilters[0] = rest as NameFilterItem;
        }
        return newFilters;
      });
    },
    [onNameFiltersChange]
  );

  const handleAdd = useCallback(() => {
    if (nameFilters.length >= 10) return;
    const hasOr = nameFilters.some((f, index) => index > 0 && f.andOr === 'or');
    onNameFiltersChange((prev) => [
      ...prev,
      { id: generateId(), condition: 'contain', value: '', andOr: hasOr ? 'or' : 'and' },
    ]);
  }, [nameFilters, onNameFiltersChange]);

  return (
    <Drawer
      title={t('device.editGroup')}
      open={open}
      onClose={onClose}
      size={520}
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
          <Form.Item
            name="name"
            label={t('device.groupNameLabel')}
            required
            rules={[{ required: true, message: t('device.groupNameLabel') }]}
            style={{ marginBottom: 0 }}
          >
            <Input maxLength={50} placeholder={t('device.groupNameLabel')} />
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
          <Form.Item
            name="autoAssignEnabled"
            label={t('device.rules.autoAssign')}
            valuePropName="checked"
            style={{ marginBottom: 12 }}
          >
            <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
          </Form.Item>

          {autoAssignEnabled && (
            <>
              <Form.Item
                name="sourceGroupId"
                label={t('device.rules.sourceGroup')}
                rules={[{ required: true, message: t('device.rules.sourceGroupRequired') }]}
                style={{ marginBottom: 12 }}
              >
                <Select
                  showSearch
                  optionFilterProp="label"
                  options={sourceGroupOptions}
                  placeholder={t('device.rules.sourceGroupPlaceholder')}
                />
              </Form.Item>
              <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} style={{ marginBottom: 12 }}>
                <Radio.Group onChange={handleMatchingModeChange}>
                  <Radio value="deviceName">{t('device.rules.deviceName')}</Radio>
                  <Radio value="lac">LAC</Radio>
                  <Radio value="tac">TAC</Radio>
                  <Radio value="serialNumber">{t('device.rules.serialNumber')}</Radio>
                </Radio.Group>
              </Form.Item>

              {/* Device name filter conditions */}
              {matchingMode === 'deviceName' && (
                <Form.Item label={<FilterConditionLabel t={t} />} style={{ marginBottom: 0 }}>
                  <NameFilterEditor
                    filters={nameFilters}
                    onAdd={handleAdd}
                    onRemove={handleRemove}
                    onUpdate={handleUpdate}
                    t={t}
                  />
                </Form.Item>
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
              {matchingMode === 'serialNumber' && (
                <Form.Item
                  name="serialNumbers"
                  label={t('device.rules.serialNumber')}
                  rules={[{ required: true, message: t('device.rules.serialNumberRequired') }]}
                  style={{ marginBottom: 0 }}
                  extra={<Text type="secondary" style={{ fontSize: 12 }}>{t('device.rules.serialNumberHint')}</Text>}
                >
                  <Input.TextArea rows={3} maxLength={2000} />
                </Form.Item>
              )}
            </>
          )}
        </div>
      </Form>
    </Drawer>
  );
}
