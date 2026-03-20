import React, { useCallback, useMemo, useState } from 'react';
import { Checkbox, Collapse, Modal, Radio, Tag } from 'antd';
import type { CheckboxChangeEvent } from 'antd/es/checkbox';
import { useT } from '@/hooks/useT';

/** 导出列定义 */
interface ExportColumn {
  code: string;
  label: string;
}

interface ExportModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: ExportParams) => void;
  confirmLoading?: boolean;
}

export interface ExportParams {
  selectedColumns: string[];
  format: 'csv' | 'xlsx';
}

// ---------------------------------------------------------------------------
// 告警导出列定义
// ---------------------------------------------------------------------------
const ALARM_COLUMNS: ExportColumn[] = [
  { code: 'alarmId', label: '告警码' },
  { code: 'severity', label: '告警级别' },
  { code: 'alarmIdentifier', label: '告警标识' },
  { code: 'alarmName', label: '可能原因' },
  { code: 'neType', label: '基站制式' },
  { code: 'equipInfo', label: '网元定位' },
  { code: 'eventType', label: '事件类型' },
  { code: 'dealState', label: '告警状态' },
  { code: 'alarmType', label: '告警类型' },
  { code: 'eventTime', label: '告警时间' },
  { code: 'updTime', label: '更新时间' },
  { code: 'specificProblem', label: '具体故障' },
  { code: 'alarmCount', label: '告警次数' },
  { code: 'dealMemo', label: '描述' },
];

// ---------------------------------------------------------------------------
// 默认必选字段 — 与告警列表默认显示列一致，导出时始终勾选且不可取消
// ---------------------------------------------------------------------------
const DEFAULT_LOCKED_CODES = new Set([
  'alarmId',         // 告警码
  'severity',        // 告警级别
  'alarmIdentifier', // 告警标识
  'alarmName',       // 可能原因
  'neType',          // 基站制式
  'equipInfo',       // 网元定位
  'eventTime',       // 告警时间
]);

export default function ExportModal({ open, onClose, onConfirm, confirmLoading }: ExportModalProps) {
  const t = useT();
  const [selectedCodes, setSelectedCodes] = useState<Set<string>>(new Set(DEFAULT_LOCKED_CODES));
  const [format, setFormat] = useState<'csv' | 'xlsx'>('xlsx');

  const handleCheckAll = useCallback((columns: ExportColumn[], checked: boolean) => {
    setSelectedCodes((prev) => {
      const next = new Set(prev);
      for (const col of columns) {
        if (DEFAULT_LOCKED_CODES.has(col.code)) continue;
        if (checked) next.add(col.code); else next.delete(col.code);
      }
      return next;
    });
  }, []);

  const handleToggle = useCallback((code: string, checked: boolean) => {
    if (DEFAULT_LOCKED_CODES.has(code)) return;
    setSelectedCodes((prev) => {
      const next = new Set(prev);
      if (checked) next.add(code); else next.delete(code);
      return next;
    });
  }, []);

  const handleOk = useCallback(() => {
    onConfirm({
      selectedColumns: Array.from(selectedCodes),
      format,
    });
  }, [onConfirm, selectedCodes, format]);

  const handleCancel = useCallback(() => {
    setSelectedCodes(new Set(DEFAULT_LOCKED_CODES));
    setFormat('xlsx');
    onClose();
  }, [onClose]);

  const checkedCount = ALARM_COLUMNS.filter((c) => selectedCodes.has(c.code)).length;
  const allChecked = checkedCount === ALARM_COLUMNS.length;
  const indeterminate = checkedCount > 0 && checkedCount < ALARM_COLUMNS.length;

  const collapseItems = useMemo(() => [
    {
      key: 'alarm',
      label: (
        <span onClick={(e) => e.stopPropagation()}>
          <Checkbox
            checked={allChecked}
            indeterminate={indeterminate}
            onChange={(e: CheckboxChangeEvent) => handleCheckAll(ALARM_COLUMNS, e.target.checked)}
            style={{ marginRight: 8 }}
          />
          {t('export.selectFields')}
          <Tag style={{ marginLeft: 8, fontSize: 11 }}>{checkedCount}/{ALARM_COLUMNS.length}</Tag>
        </span>
      ),
      children: (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}>
          {ALARM_COLUMNS.map((col) => (
            <div key={col.code} style={{ width: '33.3%', minWidth: 150 }}>
              <Checkbox
                checked={selectedCodes.has(col.code)}
                disabled={DEFAULT_LOCKED_CODES.has(col.code)}
                onChange={(e: CheckboxChangeEvent) => handleToggle(col.code, e.target.checked)}
              >
                {col.label}
              </Checkbox>
            </div>
          ))}
        </div>
      ),
    },
  ], [t, allChecked, indeterminate, checkedCount, selectedCodes, handleCheckAll, handleToggle]);

  return (
    <Modal
      title={t('export.title')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('export.startExport')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={600}
      destroyOnClose
    >
      {/* 1. 列表字段 */}
      <div style={{ marginBottom: 16 }}>
        <Collapse
          defaultActiveKey={['alarm']}
          items={collapseItems}
          size="small"
        />
      </div>

      {/* 2. 导出格式 */}
      <div>
        <span style={{ fontWeight: 'bold', marginRight: 12 }}>{t('export.format')}</span>
        <Radio.Group value={format} onChange={(e) => setFormat(e.target.value as 'csv' | 'xlsx')}>
          <Radio value="xlsx">XLSX</Radio>
          <Radio value="csv">CSV</Radio>
        </Radio.Group>
      </div>
    </Modal>
  );
}
