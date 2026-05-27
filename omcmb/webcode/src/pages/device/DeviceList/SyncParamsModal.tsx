import { useCallback, useMemo, useState } from 'react';
import { Checkbox, Collapse, Divider, Modal, Tag, Typography } from 'antd';
import type { CheckboxChangeEvent } from 'antd/es/checkbox';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

/** 网络类型标识 */
type NetworkScope = 'common' | 'eNB' | 'gNB' | 'GSM' | 'eNB+gNB';

/** 同步参数所属的网络制式 */
export type SyncNetworkType = 'eNB' | 'gNB' | 'GSM';

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
  /** 当前选中设备的统一制式 */
  networkType: SyncNetworkType;
  onClose: () => void;
  onConfirm: (selectedParams: string[], alarmSync: boolean) => void;
  confirmLoading?: boolean;
}

// ---------------------------------------------------------------------------
// 同步参数定义 — 从三制式 JSP 源码逐字段提取并合并
// scope: common = 三制式共��, eNB+gNB = LTE+5G共有, eNB/gNB/GSM = 仅该制式
// ---------------------------------------------------------------------------

const BASIC_PARAMS: SyncParam[] = [
  // --- 三制式公共 ---
  { code: 'module_type',       label: '设备型号名',          scope: 'common' },
  { code: 'software_version',  label: '软件版本',            scope: 'common' },
  { code: 'firmware_version',  label: '固件版本',            scope: 'eNB+gNB' },
  { code: 'MAC',               label: 'MAC地址',             scope: 'common' },
  { code: 'IP',                label: 'IP地址',              scope: 'common' },
  { code: 'ue_count',          label: 'UE数',                scope: 'common' },
  // --- eNB + gNB 共有 ---
  { code: 'cell_name',         label: '主机名/站点名',       scope: 'eNB+gNB' },
  { code: 'ECI',               label: 'ECI',                 scope: 'eNB+gNB' },
  { code: 'halob_flag',        label: 'HaloX',               scope: 'eNB+gNB' },
  { code: 'sync_status',       label: '同步状态',            scope: 'eNB+gNB' },
  { code: 'mme_addr',          label: 'MME Pool IPSEC地址',   scope: 'eNB+gNB' },
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
  { code: 'ipsec_addr',        label: 'IPSEC地址',           scope: 'eNB' },
  { code: 'electronic_downtilt', label: '电子下倾角',        scope: 'eNB' },
  // --- gNB 独有 ---
  { code: 'adminState',        label: 'Admin State',         scope: 'gNB' },
  { code: 'amf_status',        label: 'AMF Status',          scope: 'gNB' },
  { code: 'multiPlmnEnable',   label: 'MultiPLMN状态',       scope: 'gNB' },
  { code: 'cellConfig',        label: '小区参数',            scope: 'gNB' },
  { code: 'sub_station_name', label: '站址名称',            scope: 'gNB' },
  // --- GSM 独有 ---
  { code: 'halob_license',     label: 'License',             scope: 'GSM' },
];

const ADVANCED_PARAMS: SyncParam[] = [
  // --- eNB + gNB 共有 ---
  { code: 'rollback_version',  label: '回退版本',            scope: 'eNB+gNB' },
  { code: 'sas_param',         label: 'SAS参数',             scope: 'eNB+gNB' },
  { code: 'eu_ru',             label: 'EU/RU数',             scope: 'eNB+gNB' },
  { code: 'halob_license',     label: 'HaloB License',       scope: 'gNB' },
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

/** 判断 scope 是否匹配指定的网络制式 */
function matchesNetworkType(scope: NetworkScope, networkType: SyncNetworkType): boolean {
  if (scope === 'common') return true;
  if (scope === networkType) return true;
  if (scope === 'eNB+gNB' && (networkType === 'eNB' || networkType === 'gNB')) return true;
  return false;
}

// ---------------------------------------------------------------------------
// 各制式默认勾选字段 — 从原始 JSP 同步弹窗提取
// eNB: 原始页面通过 init() 从监控列表可见列动态映射 (monitorCols)，
//      此处取设备列表默认可见列经 monitorCols 映射后的静态等价集
// gNB: gnodeb_monitor.jsp 中 form.device 硬编码初始值
// GSM: gsm_syncParams.jsp 中 form.basic/bsc/bts 均为空数组
// ---------------------------------------------------------------------------
const DEFAULT_CHECKED_CODES: Record<SyncNetworkType, string[]> = {
  eNB: [
    'module_type',        // 设备型号名 ← deviceModel (default visible)
    'software_version',   // 软件版本 ← softwareVersion (default visible)
    'MAC',                // MAC地址 ← macAddress (default visible)
    'cell_name',          // 主机名 ← hostName (default visible, via monitorCols)
    'IP',                 // IP地址 ← ipAddress (default visible, via monitorCols)
    'cell_status',        // 激活状态 ← opState (default visible, via monitorCols)
    'ue_count',           // UE数 ← ueCount (default visible)
  ],
  gNB: [
    'cell_name',          // 5G站点名称
    'IP',                 // IP地址
    'module_type',        // 设备型号名
    'software_version',   // 软件版本
    'halob_flag',         // HaloB开关
    'ue_count',           // UE数
  ],
  GSM: [],
};

/** 制式标签配置 */
const NETWORK_TYPE_TAG: Record<SyncNetworkType, { color: string; label: string }> = {
  eNB: { color: 'blue',   label: 'LTE eNodeB' },
  gNB: { color: 'green',  label: '5G gNodeB' },
  GSM: { color: 'orange', label: 'GSM' },
};

export default function SyncParamsModal({ open, networkType, onClose, onConfirm, confirmLoading }: SyncParamsModalProps) {
  const t = useT();
  const [alarmSync, setAlarmSync] = useState(true);
  // destroyOnHidden 确保每次打开重新挂载，useState 初始值基于当前 networkType
  const [selectedCodes, setSelectedCodes] = useState<Set<string>>(
    () => new Set(DEFAULT_CHECKED_CODES[networkType])
  );

  // 根据 networkType 过滤各组参数，隐藏空组
  const groups: SyncParamGroup[] = useMemo(() => {
    const allGroups: SyncParamGroup[] = [
      { key: 'basic',    label: t('sync.basicConfig'),    params: BASIC_PARAMS },
      { key: 'advanced', label: t('sync.advancedConfig'),  params: ADVANCED_PARAMS },
      { key: 'bsc',      label: 'BSC',                    params: BSC_PARAMS },
      { key: 'bts',      label: 'BTS',                    params: BTS_PARAMS },
    ];
    return allGroups
      .map((g) => ({
        ...g,
        params: g.params.filter((p) => matchesNetworkType(p.scope, networkType)),
      }))
      .filter((g) => g.params.length > 0);
  }, [t, networkType]);

  const totalSelected = selectedCodes.size + (alarmSync ? 1 : 0);

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
    setSelectedCodes(new Set(DEFAULT_CHECKED_CODES[networkType]));
    setAlarmSync(true);
    onClose();
  }, [onClose, networkType]);

  const collapseItems = useMemo(() =>
    groups.map((group) => {
      const checkedCount = group.params.filter((p) => selectedCodes.has(p.code)).length;
      const allChecked = checkedCount === group.params.length && group.params.length > 0;
      const indeterminate = checkedCount > 0 && checkedCount < group.params.length;

      return {
        key: group.key,
        label: (
          <span onClick={(e) => e.stopPropagation()} style={{ display: 'inline-flex', alignItems: 'center' }}>
            <Checkbox
              checked={allChecked}
              indeterminate={indeterminate}
              onChange={(e: CheckboxChangeEvent) => handleCheckAll(group.params, e.target.checked)}
              style={{ marginRight: 8 }}
            />
            <span style={{ fontWeight: 500 }}>{group.label}</span>
            <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
              {checkedCount}/{group.params.length}
            </Text>
          </span>
        ),
        children: (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 0', padding: '4px 0' }}>
            {group.params.map((p) => (
              <div key={p.code} style={{ width: '50%', minWidth: 220 }}>
                <Checkbox
                  checked={selectedCodes.has(p.code)}
                  onChange={(e: CheckboxChangeEvent) => handleToggle(p.code, e.target.checked)}
                >
                  {p.label}
                </Checkbox>
              </div>
            ))}
          </div>
        ),
      };
    }), [groups, selectedCodes, handleCheckAll, handleToggle]);

  const defaultActiveKeys = useMemo(() => groups.map((g) => g.key), [groups]);

  const tagConfig = NETWORK_TYPE_TAG[networkType];

  return (
    <Modal
      title={
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
          {t('sync.title')}
          <Tag color={tagConfig.color} style={{ marginLeft: 4, fontWeight: 400 }}>
            {tagConfig.label}
          </Tag>
        </span>
      }
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={680}
      destroyOnHidden
    >
      {/* 活动告警 */}
      <div
        style={{
          padding: '10px 12px',
          background: '#fafafa',
          borderRadius: 6,
          border: '1px solid #f0f0f0',
          marginBottom: 16,
        }}
      >
        <Checkbox
          checked={alarmSync}
          onChange={(e: CheckboxChangeEvent) => setAlarmSync(e.target.checked)}
        >
          <span style={{ fontWeight: 500 }}>{t('sync.activeAlarms')}</span>
        </Checkbox>
      </div>

      {/* 检测参数 */}
      <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>{t('sync.syncParamsLabel')}</Text>
      </div>

      <Collapse
        defaultActiveKey={defaultActiveKeys}
        items={collapseItems}
        size="small"
        style={{ borderRadius: 6 }}
      />

      <Divider style={{ margin: '12px 0 8px' }} />

      {/* 已选统计 */}
      <div style={{ textAlign: 'right' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t('sync.selectedCount', { count: totalSelected })}
        </Text>
      </div>
    </Modal>
  );
}
