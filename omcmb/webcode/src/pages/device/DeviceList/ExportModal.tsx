import { useCallback, useMemo, useState } from 'react';
import { Checkbox, Collapse, Modal, Radio, Select, Tag } from 'antd';
import type { CheckboxChangeEvent } from 'antd/es/checkbox';
import { useT } from '@/hooks/useT';

/** 导出列分组 */
interface ExportColumn {
  code: string;
  /** 直出字面量（纯英文/代码缩写，如 SN）；与 labelKey 二选一 */
  label?: string;
  /** i18n message id；优先于 label */
  labelKey?: string;
}

interface ExportColumnGroup {
  key: string;
  label: string;
  tag?: string;
  tagColor?: string;
  columns: ExportColumn[];
}

interface ExportModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: ExportParams) => void;
  confirmLoading?: boolean;
}

export interface ExportParams {
  operatorCodes: string[];
  selectedColumns: string[];
  exportLicense: boolean;
  format: 'csv' | 'xlsx';
}

// ---------------------------------------------------------------------------
// 运营商选项（后续从 API 动态获取）
// ---------------------------------------------------------------------------
const OPERATOR_OPTIONS = [
  { labelKey: 'export.operator.cmcc', value: 'cmcc' },
  { labelKey: 'export.operator.ctcc', value: 'ctcc' },
  { labelKey: 'export.operator.cucc', value: 'cucc' },
];

// ---------------------------------------------------------------------------
// 导出列定义 — 按公共/eNB/gNB/GSM 分组
// 从三制式 JSP export_config.jsp / gnodeb_monitor.jsp / gsm_monitor.js 提取
// ---------------------------------------------------------------------------

const COMMON_COLUMNS: ExportColumn[] = [
  { code: 'serial_number', label: 'SN' },
  { code: 'connection_status', labelKey: 'export.col.connStatus' },
  { code: 'alarm', labelKey: 'export.col.alarmLevel' },
  { code: 'host_name', labelKey: 'export.col.name' },
  { code: 'network_type', labelKey: 'export.col.radioMode' },
  { code: 'product', labelKey: 'export.col.product' },
  { code: 'module_type', labelKey: 'export.col.deviceModel' },
  { code: 'software_version', labelKey: 'export.col.softwareVersion' },
  { code: 'firmware_version', labelKey: 'export.col.firmwareVersion' },
  { code: 'mac_address', labelKey: 'export.col.macAddress' },
  { code: 'group_name', labelKey: 'export.col.deviceGroup' },
  { code: 'cell_ip', labelKey: 'export.col.ipAddress' },
  { code: 'online_time', labelKey: 'export.col.onlineTime' },
  { code: 'offline_time', labelKey: 'export.col.offlineTime' },
  { code: 'op_state', labelKey: 'export.col.opState' },
  { code: 'ue_count', labelKey: 'export.col.ueCount' },
  { code: 'rf_status', labelKey: 'export.col.rfStatus' },
  { code: 'gps_longitude', labelKey: 'export.col.gpsLongitude' },
  { code: 'gps_latitude', labelKey: 'export.col.gpsLatitude' },
  { code: 'gps_height', labelKey: 'export.col.gpsHeight' },
];

// ---------------------------------------------------------------------------
// 默认必选字段 — 与设备列表默认显示列一致，导出时始终勾选且不可取消
// ---------------------------------------------------------------------------
const DEFAULT_LOCKED_CODES = new Set([
  'serial_number',      // SN
  'connection_status',  // connection status
  'alarm',              // alarm level
  'host_name',          // name
  'network_type',       // radio mode
  'product',            // product type
  'module_type',        // device model
  'software_version',   // software version
  'mac_address',        // MAC address
  'group_name',         // device group
  'cell_ip',            // IP address
  'online_time',        // online time
  'offline_time',       // offline time
  'op_state',           // operational state
  'ue_count',           // UE count
]);

export default function ExportModal({ open, onClose, onConfirm, confirmLoading }: ExportModalProps) {
  const t = useT();
  const [operatorCodes, setOperatorCodes] = useState<string[]>([]);
  const [selectedCodes, setSelectedCodes] = useState<Set<string>>(new Set(DEFAULT_LOCKED_CODES));
  const [exportLicense, setExportLicense] = useState(false);
  const [format, setFormat] = useState<'csv' | 'xlsx'>('xlsx');

  const groups: ExportColumnGroup[] = useMemo(() => [
    { key: 'common', label: t('export.commonFields'), columns: COMMON_COLUMNS },
  ], [t]);

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
      operatorCodes,
      selectedColumns: Array.from(selectedCodes),
      exportLicense,
      format,
    });
  }, [onConfirm, operatorCodes, selectedCodes, exportLicense, format]);

  const handleCancel = useCallback(() => {
    setOperatorCodes([]);
    setSelectedCodes(new Set(DEFAULT_LOCKED_CODES));
    setExportLicense(false);
    setFormat('xlsx');
    onClose();
  }, [onClose]);

  const collapseItems = useMemo(() =>
    groups.map((group) => {
      const checkedCount = group.columns.filter((c) => selectedCodes.has(c.code)).length;
      const allChecked = checkedCount === group.columns.length && group.columns.length > 0;
      const indeterminate = checkedCount > 0 && checkedCount < group.columns.length;

      return {
        key: group.key,
        label: (
          <span onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={allChecked}
              indeterminate={indeterminate}
              onChange={(e: CheckboxChangeEvent) => handleCheckAll(group.columns, e.target.checked)}
              style={{ marginRight: 8 }}
            />
            {group.tag && <Tag color={group.tagColor} style={{ marginRight: 4 }}>{group.tag}</Tag>}
            {group.label}
            <Tag style={{ marginLeft: 8, fontSize: 11 }}>{checkedCount}/{group.columns.length}</Tag>
          </span>
        ),
        children: (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}>
            {group.columns.map((col) => (
              <div key={col.code} style={{ width: '33.3%', minWidth: 180 }}>
                <Checkbox
                  checked={selectedCodes.has(col.code)}
                  disabled={DEFAULT_LOCKED_CODES.has(col.code)}
                  onChange={(e: CheckboxChangeEvent) => handleToggle(col.code, e.target.checked)}
                >
                  {col.labelKey ? t(col.labelKey) : col.label}
                </Checkbox>
              </div>
            ))}
          </div>
        ),
      };
    }), [groups, selectedCodes, handleCheckAll, handleToggle, t]);

  return (
    <Modal
      title={t('export.title')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('export.startExport')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={780}
      destroyOnHidden
    >
      {/* 1. 选择运营商 */}
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontWeight: 'bold', marginBottom: 8 }}>{t('export.selectOperator')}</div>
        <Select
          mode="multiple"
          allowClear
          placeholder={t('export.operatorPlaceholder')}
          options={OPERATOR_OPTIONS.map((o) => ({ label: t(o.labelKey), value: o.value }))}
          value={operatorCodes}
          onChange={setOperatorCodes}
          style={{ width: '100%' }}
          maxTagCount="responsive"
        />
      </div>

      {/* 2. 列表字段 */}
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontWeight: 'bold', marginBottom: 8 }}>{t('export.selectFields')}</div>
        <Collapse
          defaultActiveKey={['common']}
          items={collapseItems}
          size="small"
        />
      </div>

      {/* 3. 是否导出 License */}
      <div style={{ marginBottom: 16 }}>
        <Checkbox
          checked={exportLicense}
          onChange={(e: CheckboxChangeEvent) => setExportLicense(e.target.checked)}
        >
          <strong>{t('export.licenseInfo')}</strong>
          <span style={{ color: '#999', fontSize: 12, marginLeft: 8 }}>({t('export.licenseDesc')})</span>
        </Checkbox>
      </div>

      {/* 4. 导出格式 */}
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
