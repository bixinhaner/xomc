import React, { useCallback, useMemo, useState } from 'react';
import { Checkbox, Collapse, Modal, Radio, Select, Tag } from 'antd';
import type { CheckboxChangeEvent } from 'antd/es/checkbox';
import { useT } from '@/hooks/useT';

/** 导出列分组 */
interface ExportColumn {
  code: string;
  label: string;
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
  { label: '中国移动', value: 'cmcc' },
  { label: '中国电信', value: 'ctcc' },
  { label: '中国联通', value: 'cucc' },
];

// ---------------------------------------------------------------------------
// 导出列定义 — 按公共/eNB/gNB/GSM 分组
// 从三制式 JSP export_config.jsp / gnodeb_monitor.jsp / gsm_monitor.js 提取
// ---------------------------------------------------------------------------

const COMMON_COLUMNS: ExportColumn[] = [
  { code: 'serial_number', label: 'SN' },
  { code: 'connection_status', label: '连接状态' },
  { code: 'alarm', label: '告警级别' },
  { code: 'host_name', label: '名称' },
  { code: 'network_type', label: '基站制式' },
  { code: 'product', label: '产品类型' },
  { code: 'module_type', label: '设备型号' },
  { code: 'software_version', label: '软件版本' },
  { code: 'firmware_version', label: '固件版本' },
  { code: 'mac_address', label: 'MAC地址' },
  { code: 'group_name', label: '设备组' },
  { code: 'cell_ip', label: 'IP地址' },
  { code: 'online_time', label: '接入时间' },
  { code: 'offline_time', label: '断开时间' },
  { code: 'op_state', label: '激活状态' },
  { code: 'ue_count', label: 'UE数' },
  { code: 'rf_status', label: '射频状态' },
  { code: 'gps_longitude', label: 'GPS经度' },
  { code: 'gps_latitude', label: 'GPS纬度' },
  { code: 'gps_height', label: 'GPS高度' },
];

// ---------------------------------------------------------------------------
// 默认必选字段 — 与设备列表默认显示列一致，导出时始终勾选且不可取消
// ---------------------------------------------------------------------------
const DEFAULT_LOCKED_CODES = new Set([
  'serial_number',      // SN
  'connection_status',  // 连接状态
  'alarm',              // 告警级别
  'host_name',          // 名称
  'network_type',       // 基站制式
  'product',            // 产品类型
  'module_type',        // 设备型号
  'software_version',   // 软件版本
  'mac_address',        // MAC地址
  'group_name',         // 设备组
  'cell_ip',            // IP地址
  'online_time',        // 接入时间
  'offline_time',       // 断开时间
  'op_state',           // 激活状态
  'ue_count',           // UE数
]);

const ENB_COLUMNS: ExportColumn[] = [
  { code: 'CELL_IDENTITY', label: 'ECI' },
  { code: 'PHYCELLID', label: 'PCI' },
  { code: 'PLMNID', label: 'PLMN' },
  { code: 'tac', label: 'TAC' },
  { code: 'bandwidth', label: '带宽' },
  { code: 'EARFCNDLINUSE', label: 'DL EARFCN' },
  { code: 'EARFCNULINUSE', label: 'UL EARFCN' },
  { code: 'tx_power', label: '发射功率' },
  { code: 'network_model', label: '基站类型' },
  { code: 'Band', label: 'Band' },
  { code: 'mme_status', label: 'MME状态' },
  { code: 'cpe_connect', label: 'CPE连接数' },
  { code: 'IPSEC_ADDR', label: 'IPSec地址' },
  { code: 'mmepool_ipsec_addr', label: 'MME Pool IPSec' },
  { code: 'sub_frame_assignment', label: '子帧配比' },
  { code: 'root_sequence_index', label: '根序列索引' },
  { code: 'wan_speed', label: 'WAN状态' },
  { code: 'mechanical_downtilt', label: '机械下倾角' },
  { code: 'electronic_downtilt', label: '电子下倾角' },
  { code: 'vertical_beam_width', label: '垂直波束宽度' },
  { code: 'horizontal_azimuth', label: '水平方位角' },
  { code: 'gps_satellites', label: 'GPS卫星数' },
];

const GNB_COLUMNS: ExportColumn[] = [
  { code: 'nr_cell_id', label: 'NR Cell ID' },
  { code: 'PHYCELLID', label: 'PCI' },
  { code: 'tac', label: 'TAC' },
  { code: 'Band', label: 'Band' },
  { code: 'EARFCNULINUSE', label: 'UL ARFCN' },
  { code: 'EARFCNDLINUSE', label: 'DL ARFCN' },
  { code: 'tx_power', label: '发射功率' },
  { code: 'network_model', label: '基站类型' },
  { code: 'adminState', label: 'Admin State' },
  { code: 'halob_flag', label: 'HaloB' },
  { code: 'synStatus', label: '同步状态' },
  { code: 'multiPlmnEnable', label: 'MultiPLMN' },
  { code: 'amf_status', label: 'AMF状态' },
  { code: 'IPSEC_ADDR', label: 'IPSec地址' },
];

const GSM_COLUMNS: ExportColumn[] = [
  { code: 'bsc_serial_number', label: '所属BSC编码' },
  { code: 'bts_num', label: 'BTS数' },
  { code: 'lac', label: 'LAC' },
  { code: 'arfcn', label: '频点' },
  { code: 'uplink_frequency', label: '上行频率' },
  { code: 'downlink_frequency', label: '下行频率' },
  { code: 'bsc_link_status', label: 'BSC连接状态' },
  { code: 'bsc_select', label: 'BSC Select' },
  { code: 'ipa_unit_id', label: 'IPA Unit ID' },
  { code: 'oml_remote_ip', label: 'OML Remote IP' },
];

export default function ExportModal({ open, onClose, onConfirm, confirmLoading }: ExportModalProps) {
  const t = useT();
  const [operatorCodes, setOperatorCodes] = useState<string[]>([]);
  const [selectedCodes, setSelectedCodes] = useState<Set<string>>(new Set(DEFAULT_LOCKED_CODES));
  const [exportLicense, setExportLicense] = useState(false);
  const [format, setFormat] = useState<'csv' | 'xlsx'>('xlsx');

  const groups: ExportColumnGroup[] = useMemo(() => [
    { key: 'common', label: t('export.commonFields'), columns: COMMON_COLUMNS },
    { key: 'eNB',    label: t('export.enbFields'),    tag: 'eNB',  tagColor: 'blue',   columns: ENB_COLUMNS },
    { key: 'gNB',    label: t('export.gnbFields'),    tag: 'gNB',  tagColor: 'green',  columns: GNB_COLUMNS },
    { key: 'GSM',    label: t('export.gsmFields'),    tag: 'GSM',  tagColor: 'orange', columns: GSM_COLUMNS },
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
                  {col.label}
                </Checkbox>
              </div>
            ))}
          </div>
        ),
      };
    }), [groups, selectedCodes, handleCheckAll, handleToggle]);

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
      destroyOnClose
    >
      {/* 1. 选择运营商 */}
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontWeight: 'bold', marginBottom: 8 }}>{t('export.selectOperator')}</div>
        <Select
          mode="multiple"
          allowClear
          placeholder={t('export.operatorPlaceholder')}
          options={OPERATOR_OPTIONS}
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
