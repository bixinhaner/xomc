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
  /** 直出字面量（纯英文/代码缩写）；与 labelKey 二选一 */
  label?: string;
  /** i18n message id；优先于 label */
  labelKey?: string;
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
// Sync parameter definitions — extracted field-by-field from the three-RAT JSP sources.
// scope: common = shared by all three RATs, eNB+gNB = LTE+5G shared, eNB/gNB/GSM = that RAT only
// ---------------------------------------------------------------------------

const BASIC_PARAMS: SyncParam[] = [
  // --- shared by all three RATs ---
  { code: 'module_type',       labelKey: 'sync.param.module_type',      scope: 'common' },
  { code: 'software_version',  labelKey: 'sync.param.software_version', scope: 'common' },
  { code: 'firmware_version',  labelKey: 'sync.param.firmware_version', scope: 'eNB+gNB' },
  { code: 'MAC',               labelKey: 'sync.param.mac',              scope: 'common' },
  { code: 'IP',                labelKey: 'sync.param.ip',               scope: 'common' },
  { code: 'ue_count',          labelKey: 'sync.param.ue_count',         scope: 'common' },
  // --- shared by eNB + gNB ---
  { code: 'cell_name',         labelKey: 'sync.param.cell_name',        scope: 'eNB+gNB' },
  { code: 'ECI',               label: 'ECI',                            scope: 'eNB+gNB' },
  { code: 'halob_flag',        label: 'HaloX',                          scope: 'eNB+gNB' },
  { code: 'sync_status',       labelKey: 'sync.param.sync_status',      scope: 'eNB+gNB' },
  { code: 'mme_addr',          labelKey: 'sync.param.mme_addr',         scope: 'eNB+gNB' },
  { code: 'gps_position',      labelKey: 'sync.param.gps_position',     scope: 'eNB+gNB' },
  // --- eNB only ---
  { code: 'PCI',               label: 'PCI',                            scope: 'eNB' },
  { code: 'plmn',              label: 'PLMN',                           scope: 'eNB' },
  { code: 'tac',               label: 'TAC',                            scope: 'eNB' },
  { code: 'bandwidth',         labelKey: 'sync.param.bandwidth',        scope: 'eNB' },
  { code: 'earfcn',            labelKey: 'sync.param.earfcn',           scope: 'eNB' },
  { code: 'duplex_mode',       labelKey: 'sync.param.duplex_mode',      scope: 'eNB' },
  { code: 'tx_power',          labelKey: 'sync.param.tx_power',         scope: 'eNB' },
  { code: 'cell_status',       labelKey: 'sync.param.cell_status',      scope: 'eNB' },
  { code: 'mme_status',        labelKey: 'sync.param.mme_status',       scope: 'eNB' },
  { code: 'rf_status',         labelKey: 'sync.param.rf_status',        scope: 'eNB' },
  { code: 'lease',             labelKey: 'sync.param.lease',            scope: 'eNB' },
  { code: 'root_sequence_index', labelKey: 'sync.param.root_sequence_index', scope: 'eNB' },
  { code: 'gps_satellites',    labelKey: 'sync.param.gps_satellites',   scope: 'eNB' },
  { code: 'sub_frame_assignment', labelKey: 'sync.param.sub_frame_assignment', scope: 'eNB' },
  { code: 'wan_speed',         labelKey: 'sync.param.wan_speed',        scope: 'eNB' },
  { code: 'ipsec_addr',        labelKey: 'sync.param.ipsec_addr',       scope: 'eNB' },
  { code: 'electronic_downtilt', labelKey: 'sync.param.electronic_downtilt', scope: 'eNB' },
  // --- gNB only ---
  { code: 'adminState',        label: 'Admin State',                    scope: 'gNB' },
  { code: 'amf_status',        label: 'AMF Status',                     scope: 'gNB' },
  { code: 'multiPlmnEnable',   labelKey: 'sync.param.multiPlmnEnable',  scope: 'gNB' },
  { code: 'cellConfig',        labelKey: 'sync.param.cellConfig',       scope: 'gNB' },
  { code: 'sub_station_name', labelKey: 'sync.param.sub_station_name',  scope: 'gNB' },
  // --- GSM only ---
  { code: 'halob_license',     label: 'License',                        scope: 'GSM' },
];

const ADVANCED_PARAMS: SyncParam[] = [
  // --- shared by eNB + gNB ---
  { code: 'rollback_version',  labelKey: 'sync.param.rollback_version', scope: 'eNB+gNB' },
  { code: 'sas_param',         labelKey: 'sync.param.sas_param',        scope: 'eNB+gNB' },
  { code: 'eu_ru',             labelKey: 'sync.param.eu_ru',            scope: 'eNB+gNB' },
  { code: 'halob_license',     label: 'HaloB License',                  scope: 'gNB' },
  // --- eNB only ---
  { code: 'band',              labelKey: 'sync.param.band',             scope: 'eNB' },
  { code: 'cell_neighbor',     labelKey: 'sync.param.cell_neighbor',    scope: 'eNB' },
  { code: 'itfn_param',        labelKey: 'sync.param.itfn_param',       scope: 'eNB' },
  { code: 'son_pci',           label: 'SON PCI',                        scope: 'eNB' },
  { code: 'rollback_enable',   labelKey: 'sync.param.rollback_enable',  scope: 'eNB' },
  { code: 'uboot_version',     label: 'UBoot',                          scope: 'eNB' },
  { code: 'kernel_version',    label: 'Kernel',                         scope: 'eNB' },
  { code: 'is_https',          labelKey: 'sync.param.is_https',         scope: 'eNB' },
  { code: 'lan_enable',        labelKey: 'sync.param.lan_enable',       scope: 'eNB' },
  { code: 'wan_ip',            labelKey: 'sync.param.wan_ip',           scope: 'eNB' },
  { code: 'lte_turbo_enable',  label: 'LTE Turbo',                      scope: 'eNB' },
  { code: 'lock_mac_addr',     labelKey: 'sync.param.lock_mac_addr',    scope: 'eNB' },
  { code: 'lock_mac_status',   labelKey: 'sync.param.lock_mac_status',  scope: 'eNB' },
  { code: 'ipsec_bind_interface', label: 'IPSec Bind Interface', scope: 'eNB' },
  { code: 'lgw_basic',         labelKey: 'sync.param.lgw_basic',        scope: 'eNB' },
  { code: 'lgw_mode_ue_speed_statistics', label: 'UE Speed Statistics', scope: 'eNB' },
  { code: 'ipsec_auto_enroll', label: 'IPSec Auto Enroll',   scope: 'eNB' },
  { code: 'slot',              label: 'Slot',                scope: 'eNB' },
  // --- gNB only ---
  { code: 'energy_saving',     label: 'Energy Saving',       scope: 'gNB' },
  { code: 'gnb_topo_cellmgr',  label: 'gNB TOPO',            scope: 'gNB' },
  { code: 'ssl_cert_validity', label: 'SSL Cert Validity',   scope: 'gNB' },
];

const BSC_PARAMS: SyncParam[] = [
  { code: 'BtsNum',            labelKey: 'sync.param.bts_num',          scope: 'GSM' },
];

const BTS_PARAMS: SyncParam[] = [
  { code: 'cell_status',       labelKey: 'sync.param.cell_status',      scope: 'GSM' },
  { code: 'rf_status',         labelKey: 'sync.param.rf_status',        scope: 'GSM' },
  { code: 'sync_status',       labelKey: 'sync.param.sync_status',      scope: 'GSM' },
  { code: 'gps_satellites',    labelKey: 'sync.param.gps_satellites',   scope: 'GSM' },
  { code: 'currentLac',        labelKey: 'sync.param.current_lac',      scope: 'GSM' },
  { code: 'currentArfcn',      labelKey: 'sync.param.current_arfcn',    scope: 'GSM' },
  { code: 'gps_position',      labelKey: 'sync.param.gps_position_full', scope: 'GSM' },
  { code: 'bts_bsc_relationship', labelKey: 'sync.param.bts_bsc_relationship', scope: 'GSM' },
];

/** 判断 scope 是否匹配指定的网络制式 */
function matchesNetworkType(scope: NetworkScope, networkType: SyncNetworkType): boolean {
  if (scope === 'common') return true;
  if (scope === networkType) return true;
  if (scope === 'eNB+gNB' && (networkType === 'eNB' || networkType === 'gNB')) return true;
  return false;
}

// ---------------------------------------------------------------------------
// Default-checked fields per RAT — extracted from the original JSP sync dialogs.
// eNB: original page dynamically maps from monitor-list visible columns (monitorCols);
//      here we take the static equivalent set of default-visible columns after monitorCols mapping
// gNB: hardcoded initial values from form.device in gnodeb_monitor.jsp
// GSM: form.basic/bsc/bts are all empty arrays in gsm_syncParams.jsp
// ---------------------------------------------------------------------------
const DEFAULT_CHECKED_CODES: Record<SyncNetworkType, string[]> = {
  eNB: [
    'module_type',        // device model <- deviceModel (default visible)
    'software_version',   // software version <- softwareVersion (default visible)
    'MAC',                // MAC address <- macAddress (default visible)
    'cell_name',          // hostname <- hostName (default visible, via monitorCols)
    'IP',                 // IP address <- ipAddress (default visible, via monitorCols)
    'cell_status',        // operational state <- opState (default visible, via monitorCols)
    'ue_count',           // UE count <- ueCount (default visible)
  ],
  gNB: [
    'cell_name',          // 5G site name
    'IP',                 // IP address
    'module_type',        // device model
    'software_version',   // software version
    'halob_flag',         // HaloB switch
    'ue_count',           // UE count
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
                  {p.labelKey ? t(p.labelKey) : p.label}
                </Checkbox>
              </div>
            ))}
          </div>
        ),
      };
    }), [groups, selectedCodes, handleCheckAll, handleToggle, t]);

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
