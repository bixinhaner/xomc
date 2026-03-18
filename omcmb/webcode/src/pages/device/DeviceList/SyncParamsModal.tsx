import React, { useCallback, useMemo, useState } from 'react';
import { Checkbox, Collapse, Modal, Tag } from 'antd';
import type { CheckboxChangeEvent } from 'antd/es/checkbox';
import { useT } from '@/hooks/useT';

/** 网络类型标识 */
type NetworkScope = 'common' | 'eNB' | 'gNB' | 'GSM' | 'eNB+gNB';

interface SyncParam {
  code: string;
  label: string;
  /** 适用的网络类型范围 */
  scope: NetworkScope;
}

interface SyncParamGroup {
  key: string;
  label: string;
  params: SyncParam[];
}

interface SyncParamsModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (selectedParams: string[], alarmSync: boolean) => void;
  confirmLoading?: boolean;
}

// ---------------------------------------------------------------------------
// 同步参数定义 — 从三制式 JSP 源码逐字段提取并合并
// scope: common = 三制式共有, eNB+gNB = LTE+5G共有, eNB/gNB/GSM = 仅该制式
// ---------------------------------------------------------------------------

const BASIC_PARAMS: SyncParam[] = [
  // --- 三制式公共 ---
  { code: 'module_type',       label: '设备型号名',          scope: 'common' },
  { code: 'software_version',  label: '软件版本',            scope: 'common' },
  { code: 'firmware_version',  label: '固件版本',            scope: 'common' },
  { code: 'MAC',               label: 'MAC地址',             scope: 'common' },
  { code: 'IP',                label: 'IP地址',              scope: 'common' },
  { code: 'ue_count',          label: 'UE数',                scope: 'common' },
  // --- eNB + gNB 共有 ---
  { code: 'cell_name',         label: '主机名/站点名',       scope: 'eNB+gNB' },
  { code: 'ECI',               label: 'ECI',                 scope: 'eNB+gNB' },
  { code: 'halob_flag',        label: 'HaloB开关',           scope: 'eNB+gNB' },
  { code: 'sync_status',       label: '同步状态',            scope: 'eNB+gNB' },
  { code: 'mme_addr',          label: 'IPSec地址',           scope: 'eNB+gNB' },
  { code: 'gps_position',      label: 'GPS位置',             scope: 'eNB+gNB' },
  // --- eNB 独有 ---
  { code: 'PCI',               label: 'PCI',                 scope: 'eNB' },
  { code: 'plmn',              label: 'PLMN',                scope: 'eNB' },
  { code: 'tac',               label: 'TAC',                 scope: 'eNB' },
  { code: 'bandwidth',         label: '带宽',                scope: 'eNB' },
  { code: 'earfcn',            label: '频点',                scope: 'eNB' },
  { code: 'duplex_mode',       label: '双工模式',            scope: 'eNB' },
  { code: 'tx_power',          label: '发射功率',            scope: 'eNB' },
  { code: 'cell_status',       label: '激活状态',            scope: 'eNB' },
  { code: 'mme_status',        label: 'MME状态',             scope: 'eNB' },
  { code: 'rf_status',         label: '射频开关状态',        scope: 'eNB' },
  { code: 'lease',             label: '锁定状态',            scope: 'eNB' },
  { code: 'root_sequence_index', label: '根序列索引',        scope: 'eNB' },
  { code: 'gps_satellites',    label: 'GPS卫星数',           scope: 'eNB' },
  { code: 'sub_frame_assignment', label: '子帧配比',         scope: 'eNB' },
  { code: 'wan_speed',         label: 'WAN状态',             scope: 'eNB' },
  { code: 'ipsec_addr',        label: 'IPSec地址(eNB)',      scope: 'eNB' },
  { code: 'electronic_downtilt', label: '电子下倾角',        scope: 'eNB' },
  // --- gNB 独有 ---
  { code: 'adminState',        label: 'Admin State',         scope: 'gNB' },
  { code: 'amf_status',        label: 'AMF Status',          scope: 'gNB' },
  { code: 'multiPlmnEnable',   label: 'MultiPLMN状态',       scope: 'gNB' },
  { code: 'cellConfig',        label: '小区参数',            scope: 'gNB' },
  // --- GSM 独有 ---
  { code: 'halob_license',     label: 'License',             scope: 'GSM' },
];

const ADVANCED_PARAMS: SyncParam[] = [
  // --- eNB + gNB 共有 ---
  { code: 'rollback_version',  label: '回退版本',            scope: 'eNB+gNB' },
  { code: 'sas_param',         label: 'SAS参数',             scope: 'eNB+gNB' },
  { code: 'eu_ru',             label: 'EU/RU数',             scope: 'eNB+gNB' },
  { code: 'halob_license',     label: 'HaloB License',       scope: 'eNB+gNB' },
  // --- eNB 独有 ---
  { code: 'band',              label: '频段',                scope: 'eNB' },
  { code: 'cell_neighbor',     label: 'SAS邻区',            scope: 'eNB' },
  { code: 'itfn_param',        label: '背向接口',            scope: 'eNB' },
  { code: 'son_pci',           label: 'SON PCI',             scope: 'eNB' },
  { code: 'rollback_enable',   label: '回退开关',            scope: 'eNB' },
  { code: 'uboot_version',     label: 'UBoot版本',           scope: 'eNB' },
  { code: 'kernel_version',    label: 'Kernel版本',          scope: 'eNB' },
  { code: 'is_https',          label: 'Https状态',           scope: 'eNB' },
  { code: 'lan_enable',        label: 'LAN状态',             scope: 'eNB' },
  { code: 'wan_ip',            label: 'WAN IP地址',          scope: 'eNB' },
  { code: 'lte_turbo_enable',  label: 'LTE Turbo',           scope: 'eNB' },
  { code: 'lock_mac_addr',     label: '锁定MAC',             scope: 'eNB' },
  { code: 'lock_mac_status',   label: '锁定MAC状态',         scope: 'eNB' },
  { code: 'ipsec_bind_interface', label: 'IPSec Bind Interface', scope: 'eNB' },
  { code: 'lgw_basic',         label: 'WCG参数',             scope: 'eNB' },
  { code: 'lgw_mode_ue_speed_statistics', label: 'UE Speed Statistics', scope: 'eNB' },
  { code: 'ipsec_auto_enroll', label: 'IPSec Auto Enroll',   scope: 'eNB' },
  { code: 'slot',              label: 'Slot',                scope: 'eNB' },
  // --- gNB 独有 ---
  { code: 'energy_saving',     label: 'Energy Saving',       scope: 'gNB' },
  { code: 'gnb_topo_cellmgr',  label: 'gNB TOPO',            scope: 'gNB' },
  { code: 'ssl_cert_validity', label: 'SSL Cert Validity',   scope: 'gNB' },
];

const BSC_PARAMS: SyncParam[] = [
  { code: 'BtsNum',            label: 'BTS数',               scope: 'GSM' },
];

const BTS_PARAMS: SyncParam[] = [
  { code: 'cell_status',       label: '激活状态',            scope: 'GSM' },
  { code: 'rf_status',         label: '射频开关状态',        scope: 'GSM' },
  { code: 'sync_status',       label: '同步状态',            scope: 'GSM' },
  { code: 'gps_satellites',    label: 'GPS卫星数',           scope: 'GSM' },
  { code: 'currentLac',        label: 'LAC',                 scope: 'GSM' },
  { code: 'currentArfcn',      label: '频点+上行频率+下行频率', scope: 'GSM' },
  { code: 'gps_position',      label: 'GPS经度+纬度+高度',   scope: 'GSM' },
  { code: 'bts_bsc_relationship', label: 'IPA Unit ID + OML Remote IP + BSC关系', scope: 'GSM' },
];

/** scope → Tag 颜色 */
const SCOPE_COLOR: Record<NetworkScope, string> = {
  common: '',
  'eNB+gNB': 'purple',
  eNB: 'blue',
  gNB: 'green',
  GSM: 'orange',
};

/** scope → Tag 文本 */
const SCOPE_TEXT: Record<NetworkScope, string> = {
  common: '',
  'eNB+gNB': 'LTE+5G',
  eNB: 'eNB',
  gNB: 'gNB',
  GSM: 'GSM',
};

function ScopeTag({ scope }: { scope: NetworkScope }) {
  if (scope === 'common') return null;
  return (
    <Tag color={SCOPE_COLOR[scope]} style={{ fontSize: 10, lineHeight: '16px', padding: '0 4px', marginLeft: 4 }}>
      {SCOPE_TEXT[scope]}
    </Tag>
  );
}

export default function SyncParamsModal({ open, onClose, onConfirm, confirmLoading }: SyncParamsModalProps) {
  const t = useT();
  const [alarmSync, setAlarmSync] = useState(true);
  const [selectedCodes, setSelectedCodes] = useState<Set<string>>(new Set());

  const groups: SyncParamGroup[] = useMemo(() => [
    { key: 'basic',    label: t('sync.basicConfig'),    params: BASIC_PARAMS },
    { key: 'advanced', label: t('sync.advancedConfig'),  params: ADVANCED_PARAMS },
    { key: 'bsc',      label: 'BSC',                    params: BSC_PARAMS },
    { key: 'bts',      label: 'BTS',                    params: BTS_PARAMS },
  ], [t]);

  const handleCheckAll = useCallback((params: SyncParam[], checked: boolean) => {
    setSelectedCodes((prev) => {
      const next = new Set(prev);
      for (const p of params) {
        if (checked) next.add(p.code); else next.delete(p.code);
      }
      return next;
    });
  }, []);

  const handleToggle = useCallback((code: string, checked: boolean) => {
    setSelectedCodes((prev) => {
      const next = new Set(prev);
      if (checked) next.add(code); else next.delete(code);
      return next;
    });
  }, []);

  const handleOk = useCallback(() => {
    onConfirm(Array.from(selectedCodes), alarmSync);
  }, [onConfirm, selectedCodes, alarmSync]);

  const handleCancel = useCallback(() => {
    setSelectedCodes(new Set());
    setAlarmSync(true);
    onClose();
  }, [onClose]);

  const collapseItems = useMemo(() =>
    groups.map((group) => {
      const checkedCount = group.params.filter((p) => selectedCodes.has(p.code)).length;
      const allChecked = checkedCount === group.params.length && group.params.length > 0;
      const indeterminate = checkedCount > 0 && checkedCount < group.params.length;

      return {
        key: group.key,
        label: (
          <span onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={allChecked}
              indeterminate={indeterminate}
              onChange={(e: CheckboxChangeEvent) => handleCheckAll(group.params, e.target.checked)}
              style={{ marginRight: 8 }}
            />
            {group.label}
            <Tag style={{ marginLeft: 8, fontSize: 11 }}>{checkedCount}/{group.params.length}</Tag>
          </span>
        ),
        children: (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 0' }}>
            {group.params.map((p) => (
              <div key={p.code} style={{ width: '50%', minWidth: 240 }}>
                <Checkbox
                  checked={selectedCodes.has(p.code)}
                  onChange={(e: CheckboxChangeEvent) => handleToggle(p.code, e.target.checked)}
                >
                  {p.label}
                </Checkbox>
                <ScopeTag scope={p.scope} />
              </div>
            ))}
          </div>
        ),
      };
    }), [groups, selectedCodes, handleCheckAll, handleToggle]);

  return (
    <Modal
      title={t('sync.title')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={680}
      destroyOnClose
    >
      {/* 告警管理 */}
      <div style={{ marginBottom: 12, paddingLeft: 4 }}>
        <Checkbox
          checked={alarmSync}
          onChange={(e: CheckboxChangeEvent) => setAlarmSync(e.target.checked)}
        >
          <strong>{t('sync.alarmManagement')}</strong>
          <span style={{ color: '#999', fontSize: 12, marginLeft: 8 }}>({t('sync.activeAlarms')})</span>
        </Checkbox>
      </div>

      {/* 检测参数 */}
      <Collapse
        defaultActiveKey={['basic', 'advanced', 'bsc', 'bts']}
        items={collapseItems}
        size="small"
      />
    </Modal>
  );
}
