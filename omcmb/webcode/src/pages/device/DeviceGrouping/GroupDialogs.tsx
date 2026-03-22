import React, { useMemo } from 'react';
import { Button, Drawer, Form, Input, Modal, Radio, Select, Typography } from 'antd';
import { CloseCircleOutlined, PlusOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd';
import type { NameFilterItem } from './types';
import {
  generateId,
  generateOperators,
  getAndOrOptions,
  getFilterConditionOptions,
} from './types';

const { Text } = Typography;

export interface GroupDialogsProps {
  // Add Level-1 Group Modal
  addModalOpen: boolean;
  addForm: FormInstance<{ name: string; description: string }>;
  onAddModalOk: () => void;
  onAddModalCancel: () => void;

  // Edit Level-1 Group Modal
  editModalOpen: boolean;
  editForm: FormInstance<{ name: string; description: string }>;
  onEditModalOk: () => void;
  onEditModalCancel: () => void;

  // Add Child Group (Level-2) Drawer
  addChildDrawerOpen: boolean;
  addChildForm: FormInstance<{ name: string; matchingMode: 'deviceName' | 'lac' | 'tac'; tacRag: string }>;
  matchingMode: string | undefined;
  nameFilters: NameFilterItem[];
  onAddChildDrawerClose: () => void;
  onSaveChildGroup: () => void;
  onMatchingModeChange: () => void;
  onAddFilter: () => void;
  onRemoveFilter: (id: string) => void;
  onUpdateFilter: (id: string, field: keyof NameFilterItem, value: string) => void;

  // Edit Level-2 Group Drawer
  editLevel2DrawerOpen: boolean;
  editLevel2Form: FormInstance<{ name: string; matchingMode: 'deviceName' | 'lac' | 'tac'; tacRag: string }>;
  editLevel2MatchingMode: string | undefined;
  editLevel2NameFilters: NameFilterItem[];
  onEditLevel2DrawerClose: () => void;
  onSaveEditLevel2: () => void;
  onEditLevel2NameFiltersChange: React.Dispatch<React.SetStateAction<NameFilterItem[]>>;

  t: (id: string, values?: Record<string, unknown>) => string;
}

export default function GroupDialogs({
  addModalOpen,
  addForm,
  onAddModalOk,
  onAddModalCancel,
  editModalOpen,
  editForm,
  onEditModalOk,
  onEditModalCancel,
  addChildDrawerOpen,
  addChildForm,
  matchingMode,
  nameFilters,
  onAddChildDrawerClose,
  onSaveChildGroup,
  onMatchingModeChange,
  onAddFilter,
  onRemoveFilter,
  onUpdateFilter,
  editLevel2DrawerOpen,
  editLevel2Form,
  editLevel2MatchingMode,
  editLevel2NameFilters,
  onEditLevel2DrawerClose,
  onSaveEditLevel2,
  onEditLevel2NameFiltersChange,
  t,
}: GroupDialogsProps) {
  // Preview condition description for add child form
  const previewText = useMemo(() => {
    if (matchingMode === 'deviceName') {
      return generateOperators({ matchingMode: 'deviceName', nameRuleList: nameFilters }, t);
    }
    return '';
  }, [matchingMode, nameFilters, t]);

  return (
    <>
      {/* Add Group Modal */}
      <Modal
        title={t('common.add')}
        open={addModalOpen}
        onOk={onAddModalOk}
        onCancel={onAddModalCancel}
        okText={t('common.confirm')}
      >
        <Form form={addForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('table.name')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit Group Modal */}
      <Modal
        title={t('common.edit')}
        open={editModalOpen}
        onOk={onEditModalOk}
        onCancel={onEditModalCancel}
        okText={t('common.save')}
      >
        <Form form={editForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label={t('table.name')}
            rules={[{ required: true, message: t('common.placeholder') }]}
          >
            <Input placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('table.description')}>
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Child Group Drawer */}
      <Drawer
        title={t('device.addChildGroup')}
        open={addChildDrawerOpen}
        onClose={onAddChildDrawerClose}
        width={520}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={onAddChildDrawerClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={onSaveChildGroup}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={addChildForm} layout="vertical">
          {/* Basic info */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8,
            marginBottom: 16
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('device.basicInfo')}
            </div>
            <Form.Item
              name="name"
              label={t('device.childGroupName')}
              rules={[{ required: true, message: t('common.placeholder') }]}
              style={{ marginBottom: 0 }}
            >
              <Input placeholder={t('common.placeholder')} maxLength={50} />
            </Form.Item>
          </div>

          {/* Match rule */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8
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
                <Form.Item
                  label={
                    <span>
                      {t('device.rules.filterCondition')}
                      <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
                        {t('device.rules.conditionLimit', { max: 10 })}
                      </Text>
                    </span>
                  }
                  style={{ marginBottom: 0 }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    {nameFilters.map((filter, index) => (
                      <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                        {index === 0 ? (
                          <>
                            <Select
                              value={filter.condition}
                              style={{ width: 120 }}
                              options={getFilterConditionOptions(t)}
                              onChange={(v) => onUpdateFilter(filter.id, 'condition', v)}
                            />
                            <Input
                              value={filter.value}
                              style={{ flex: 1 }}
                              maxLength={64}
                              placeholder={t('common.placeholder')}
                              onChange={(e) => onUpdateFilter(filter.id, 'value', e.target.value)}
                            />
                          </>
                        ) : (
                          <>
                            <Select
                              value={filter.andOr || 'and'}
                              style={{ width: 70 }}
                              options={getAndOrOptions(t)}
                              onChange={(v) => onUpdateFilter(filter.id, 'andOr', v)}
                            />
                            <Select
                              value={filter.condition}
                              style={{ width: 120 }}
                              options={getFilterConditionOptions(t)}
                              onChange={(v) => onUpdateFilter(filter.id, 'condition', v)}
                            />
                            <Input
                              value={filter.value}
                              style={{ flex: 1 }}
                              maxLength={64}
                              placeholder={t('common.placeholder')}
                              onChange={(e) => onUpdateFilter(filter.id, 'value', e.target.value)}
                            />
                            <Button
                              type="text"
                              size="small"
                              icon={<CloseCircleOutlined />}
                              onClick={() => onRemoveFilter(filter.id)}
                              style={{ color: 'var(--color-text-quaternary)' }}
                            />
                          </>
                        )}
                      </div>
                    ))}
                  </div>
                  {nameFilters.length < 10 && (
                    <Button type="dashed" icon={<PlusOutlined />} onClick={onAddFilter} style={{ marginTop: 8 }}>
                      {t('device.rules.addCondition')}
                    </Button>
                  )}
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

      {/* Edit Level 2 Group Drawer */}
      <Drawer
        title={t('device.editGroup')}
        open={editLevel2DrawerOpen}
        onClose={onEditLevel2DrawerClose}
        width={520}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={onEditLevel2DrawerClose}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={onSaveEditLevel2}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form form={editLevel2Form} layout="vertical">
          {/* Basic info */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8,
            marginBottom: 16
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('device.basicInfo')}
            </div>
            <Form.Item
              name="name"
              label={t('device.groupName')}
              rules={[{ required: true, message: t('common.placeholder') }]}
              style={{ marginBottom: 0 }}
            >
              <Input placeholder={t('common.placeholder')} maxLength={50} />
            </Form.Item>
          </div>

          {/* Match rule */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8
          }}>
            <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('device.matchRule')}
            </div>
            <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
              {t('device.matchRuleDesc')}
            </Text>
            <Form.Item name="matchingMode" label={t('device.rules.matchingMode')} style={{ marginBottom: 12 }}>
              <Radio.Group onChange={() => {
                onEditLevel2NameFiltersChange([{ id: generateId(), condition: 'contain', value: '' }]);
                editLevel2Form.setFieldsValue({ tacRag: '' });
              }}>
                <Radio value="deviceName">{t('device.rules.deviceName')}</Radio>
                <Radio value="lac">LAC</Radio>
                <Radio value="tac">TAC</Radio>
              </Radio.Group>
            </Form.Item>

            {/* Device name filter conditions */}
            {editLevel2MatchingMode === 'deviceName' && (
              <>
                <Form.Item
                  label={
                    <span>
                      {t('device.rules.filterCondition')}
                      <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
                        {t('device.rules.conditionLimit', { max: 10 })}
                      </Text>
                    </span>
                  }
                  style={{ marginBottom: 0 }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    {editLevel2NameFilters.map((filter, index) => (
                      <div key={filter.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                        {index === 0 ? (
                          <>
                            <Select
                              value={filter.condition}
                              style={{ width: 120 }}
                              options={getFilterConditionOptions(t)}
                              onChange={(v) => {
                                onEditLevel2NameFiltersChange(prev => prev.map(f => f.id === filter.id ? { ...f, condition: v } : f));
                              }}
                            />
                            <Input
                              value={filter.value}
                              style={{ flex: 1 }}
                              maxLength={64}
                              placeholder={t('common.placeholder')}
                              onChange={(e) => {
                                onEditLevel2NameFiltersChange(prev => prev.map(f => f.id === filter.id ? { ...f, value: e.target.value } : f));
                              }}
                            />
                          </>
                        ) : (
                          <>
                            <Select
                              value={filter.andOr || 'and'}
                              style={{ width: 70 }}
                              options={getAndOrOptions(t)}
                              onChange={(v) => {
                                onEditLevel2NameFiltersChange(prev => prev.map(f => f.id === filter.id ? { ...f, andOr: v } : f));
                              }}
                            />
                            <Select
                              value={filter.condition}
                              style={{ width: 120 }}
                              options={getFilterConditionOptions(t)}
                              onChange={(v) => {
                                onEditLevel2NameFiltersChange(prev => prev.map(f => f.id === filter.id ? { ...f, condition: v } : f));
                              }}
                            />
                            <Input
                              value={filter.value}
                              style={{ flex: 1 }}
                              maxLength={64}
                              placeholder={t('common.placeholder')}
                              onChange={(e) => {
                                onEditLevel2NameFiltersChange(prev => prev.map(f => f.id === filter.id ? { ...f, value: e.target.value } : f));
                              }}
                            />
                            <Button
                              type="text"
                              size="small"
                              icon={<CloseCircleOutlined />}
                              onClick={() => {
                                onEditLevel2NameFiltersChange(prev => {
                                  if (prev.length <= 1) return prev;
                                  const newFilters = prev.filter(f => f.id !== filter.id);
                                  if (newFilters.length > 0 && newFilters[0].andOr !== undefined) {
                                    const { andOr: _, ...rest } = newFilters[0];
                                    newFilters[0] = rest as NameFilterItem;
                                  }
                                  return newFilters;
                                });
                              }}
                              style={{ color: 'var(--color-text-quaternary)' }}
                            />
                          </>
                        )}
                      </div>
                    ))}
                  </div>
                  {editLevel2NameFilters.length < 10 && (
                    <Button
                      type="dashed"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        if (editLevel2NameFilters.length >= 10) return;
                        const hasOr = editLevel2NameFilters.some((f, index) => index > 0 && f.andOr === 'or');
                        onEditLevel2NameFiltersChange(prev => [
                          ...prev,
                          { id: generateId(), condition: 'contain', value: '', andOr: hasOr ? 'or' : 'and' },
                        ]);
                      }}
                      style={{ marginTop: 8 }}
                    >
                      {t('device.rules.addCondition')}
                    </Button>
                  )}
                </Form.Item>
              </>
            )}

            {/* TAC/LAC input */}
            {(editLevel2MatchingMode === 'tac' || editLevel2MatchingMode === 'lac') && (
              <Form.Item
                name="tacRag"
                label={editLevel2MatchingMode === 'tac' ? 'TAC' : 'LAC'}
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
    </>
  );
}
