import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { App, Button, Dropdown, Input, Popconfirm, Popover, Space, Tag, Tooltip, Typography } from 'antd';
import type { MenuProps } from 'antd';
import {
  CheckOutlined,
  CloseOutlined,
  EditOutlined,
  ExportOutlined,
  EyeOutlined,
  MoreOutlined,
  ReloadOutlined,
  RestOutlined,
  SwapOutlined,
  SyncOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import StatisticsPanel from '@/components/StatisticsPanel';
import StatusIndicator from '@/components/StatusIndicator';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useDeviceList, useDeleteDevices } from '@/hooks/api/useDevices';
import { useT } from '@/hooks/useT';
import type { Device } from '@/types/device';
import SyncParamsModal from './SyncParamsModal';
import type { SyncNetworkType } from './SyncParamsModal';
import MoveToGroupModal from './MoveToGroupModal';
import ExportModal from './ExportModal';
import type { ExportParams } from './ExportModal';

const { Link } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};


export default function DeviceList() {
  const t = useT();
  const navigate = useNavigate();
  const { message, modal } = App.useApp();
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [syncModalOpen, setSyncModalOpen] = useState(false);
  // 同步目标：batch=批量(使用 selectedRowKeys)，single=单设备(指定 device)
  const [syncTarget, setSyncTarget] = useState<{ mode: 'batch' } | { mode: 'single'; device: Device }>({ mode: 'batch' });
  const [moveToGroupModalOpen, setMoveToGroupModalOpen] = useState(false);
  const [exportModalOpen, setExportModalOpen] = useState(false);

  // Remark 列头自定义标签
  const [remarkLabel, setRemarkLabel] = useState(() => {
    return localStorage.getItem('omc_remark_label') || 'Remark';
  });
  const [editingRemark, setEditingRemark] = useState(false);
  const [remarkInput, setRemarkInput] = useState('');

  const handleRemarkLabelSave = useCallback(() => {
    const val = remarkInput.trim();
    if (!val) return;
    setRemarkLabel(val);
    setEditingRemark(false);
    localStorage.setItem('omc_remark_label', val);
    // TODO: 接入 POST /cell/cpeinfos/updateColumnAlias.action
    // params: { columnName: 'remark', columnAlias: val }
  }, [remarkInput]);

  const handleRemarkLabelCancel = useCallback(() => {
    setEditingRemark(false);
  }, []);

  const remarkHeaderRender = useMemo(() => {
    if (editingRemark) {
      return (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }} onClick={(e) => e.stopPropagation()}>
          <Input
            size="small"
            value={remarkInput}
            onChange={(e) => setRemarkInput(e.target.value)}
            onPressEnter={handleRemarkLabelSave}
            style={{ width: 120 }}
            maxLength={30}
            autoFocus
          />
          <CheckOutlined
            style={{ fontSize: 12, color: '#52c41a', cursor: 'pointer' }}
            onClick={handleRemarkLabelSave}
          />
          <CloseOutlined
            style={{ fontSize: 12, color: '#ff4d4f', cursor: 'pointer' }}
            onClick={handleRemarkLabelCancel}
          />
        </span>
      );
    }
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        <Tooltip title={remarkLabel}>
          <span style={{ maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {remarkLabel}
          </span>
        </Tooltip>
        <EditOutlined
          style={{ fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}
          onClick={(e) => {
            e.stopPropagation();
            setRemarkInput(remarkLabel);
            setEditingRemark(true);
          }}
        />
      </span>
    );
  }, [editingRemark, remarkInput, remarkLabel, handleRemarkLabelSave, handleRemarkLabelCancel]);

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize } as Parameters<typeof useDeviceList>[0]),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useDeviceList(queryParams);
  const deleteDevices = useDeleteDevices();

  const devices: Device[] = data?.items ?? [];
  const total = data?.total ?? 0;
  const stats = data?.stats ?? { total: 0, online: 0, offline: 0, alarmed: 0 };

  // 计算选中设备的统一制式 — 全部相同时返回该制式，否则 null
  const selectedNetworkType: SyncNetworkType | null = useMemo(() => {
    if (selectedRowKeys.length === 0) return null;
    const selectedDevices = devices.filter((d) => selectedRowKeys.includes(d.id));
    if (selectedDevices.length === 0) return null;
    const firstType = selectedDevices[0].networkType;
    const allSame = selectedDevices.every((d) => d.networkType === firstType);
    if (!allSame) return null;
    if (firstType === 'eNB' || firstType === 'gNB' || firstType === 'GSM') return firstType;
    return null;
  }, [selectedRowKeys, devices]);

  const SEVERITY_LABEL: Record<string, string> = useMemo(() => ({
    critical: t('alarm.severity.critical'),
    major: t('alarm.severity.major'),
    minor: t('alarm.severity.minor'),
    warning: t('alarm.severity.warning'),
    none: t('alarm.severity.none'),
  }), [t]);


  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    // --- 搜索项：文本搜索覆盖 SN/名称/IP/MAC/ECI/PCI ---
    {
      name: 'searchText',
      label: t('filter.searchText'),
      type: 'input',
      placeholder: 'SN / ' + t('device.hostName') + ' / IP / MAC / ECI / PCI',
    },

    // --- 筛选项：三制式公共（默认显示） ---
    {
      name: 'connStatus',
      label: t('device.connStatus'),
      type: 'multi-select',
      options: [
        { label: t('filter.conn.normal'), value: '1' },
        { label: t('filter.conn.disconnected'), value: '0' },
        { label: t('filter.conn.syncing'), value: '3' },
        { label: t('filter.conn.syncFailed'), value: '2' },
      ],
    },
    {
      name: 'opState',
      label: t('device.opState'),
      type: 'select',
      options: [
        { label: t('status.active'), value: '1' },
        { label: t('status.inactive'), value: '0' },
      ],
    },
    {
      name: 'networkType',
      label: t('device.radioMode'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'GSM', value: 'GSM' },
      ],
    },
    {
      name: 'productModel',
      label: t('device.productType'),
      type: 'multi-select',
      options: [
        // eNB 产品类型（动态，后端返回）— 此处先列举已知选项
        { label: 'PM-B4860', value: 'PM-B4860' },
        { label: 'QAFA', value: 'QAFA' },
        { label: 'QATA', value: 'QATA' },
        { label: 'QAFB', value: 'QAFB' },
        { label: 'RTD', value: 'RTD' },
        // gNB 产品类型（硬编码）
        { label: 'BaiBNX', value: 'BaiBNX' },
        { label: 'BaiBNQ', value: 'BaiBNQ' },
        // GSM 产品类型（硬编码）
        { label: 'BSC', value: 'BSC' },
        { label: 'BTS', value: 'BTS' },
      ],
    },

    // --- 筛选项：三制式公共（默认折叠） ---
    {
      name: 'modelName',
      label: t('device.model'),
      type: 'multi-select',
      options: [],  // TODO: 动态加载 /cell/cpeinfos/getModelNameList.action
    },
    {
      name: 'softwareVersion',
      label: t('device.softwareVersion'),
      type: 'multi-select',
      options: [],  // TODO: 动态加载 /cell/cpeinfos/getCellVersionList.action
    },
    {
      name: 'firmwareVersion',
      label: t('device.firmwareVersion'),
      type: 'multi-select',
      options: [],  // TODO: 动态加载 /cell/cpeinfos/getFirmwareVersionList.action
    },
    {
      name: 'groupId',
      label: t('device.groupName'),
      type: 'multi-select',
      options: [],  // TODO: 动态加载 /cell/cpeinfos/getDeviceGroupListByCell.action
    },

    // --- 筛选项：eNB + gNB ---
    {
      name: 'halobFlag',
      label: 'HaloB',
      type: 'select',
      options: [
        { label: t('common.yes'), value: '1' },
        { label: t('common.no'), value: '0' },
      ],
    },

    // --- 筛选项：仅 gNB ---
    {
      name: 'multiPlmnEnable',
      label: 'MultiPLMN',
      type: 'select',
      options: [
        { label: t('common.enable'), value: '1' },
        { label: t('common.disable'), value: '0' },
      ],
    },

    // --- 筛选项：仅 GSM ---
    {
      name: 'bscSerialnumber',
      label: t('filter.bscCode'),
      type: 'multi-select',
      options: [],  // TODO: 动态加载 /cell/cpeinfos/getBSCSnForBTSList.action
    },
  ], [t]);

  // 统计面板 — 基于筛选条件的全量统计（由后端/mock 返回，非当前页）
  const statsItems = useMemo(() => [
    { label: t('device.count.total'), value: stats.total },
    { label: t('status.online'), value: stats.online, color: '#52C41A' },
    { label: t('status.offline'), value: stats.offline, color: '#8C8C8C' },
    { label: t('common.hasAlarm'), value: stats.alarmed, color: '#FA8C16' },
  ], [stats, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setCurrentPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
  }, []);

  // 批量重启 — 确认对话框 → "命令已经下发。"
  const handleBatchReboot = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('common.confirm'),
        content: t('common.rebootConfirmMsg'),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: async () => {
          // TODO: 接入 POST /task/reboot/batchRebootCell.action，ids 传给后端
          console.log('batch reboot ids:', ids);
          void message.info(t('common.commandSent'));
          setSelectedRowKeys([]);
          void refetch();
        },
      });
    },
    [modal, message, t, refetch]
  );

  // 批量回收站 — 警告确认 + 说明文字
  const handleRecycle = useCallback(
    (ids: string[]) => {
      modal.confirm({
        title: t('common.recycleConfirmTitle'),
        content: t('common.recycleConfirmDesc'),
        okText: t('common.confirm'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: async () => {
          // TODO: 接入 POST /recycle/moveDeviceToRecycle.action
          await deleteDevices.mutateAsync(ids);
          void message.success(t('common.operationSuccess'));
          setSelectedRowKeys([]);
        },
      });
    },
    [modal, message, deleteDevices, t]
  );

  // 批量同步 — 打开参数选择对话框
  const handleBatchSync = useCallback(() => {
    setSyncTarget({ mode: 'batch' });
    setSyncModalOpen(true);
  }, []);

  // 同步确认回调 — 区分单设备和批量
  const handleSyncConfirm = useCallback(
    (selectedParams: string[], alarmSync: boolean) => {
      const params = alarmSync ? [...selectedParams, 'sync_alarm'] : selectedParams;

      if (syncTarget.mode === 'single') {
        // 单设备同步
        // TODO: 接入 POST /cell/param/refreshCellInfo.action
        // params: { smallCellCode: syncTarget.device.sn, selectedParams: params.join(','), isGnb: syncTarget.device.networkType === 'gNB' ? '1' : '0' }
        console.log('single sync:', { sn: syncTarget.device.sn, networkType: syncTarget.device.networkType, selectedParams: params.join(',') });
        void message.success(t('common.commandSent'));
      } else {
        // 批量同步
        const cellCodes = selectedRowKeys.join(',');
        // TODO: 接入 POST /cell/quicksettings/batchSyncCell.action
        // params: { smallCellCode: cellCodes, selectedParams: params.join(',') }
        console.log('batch sync:', { cellCodes, selectedParams: params.join(',') });
        setSelectedRowKeys([]);
      }

      setSyncModalOpen(false);
      void refetch();
    },
    [syncTarget, selectedRowKeys, refetch, message, t]
  );

  // 移动到设备组 — 打开设备组选择对话框
  const handleMoveToGroup = useCallback(() => {
    setMoveToGroupModalOpen(true);
  }, []);

  // 移动到设备组确认回调
  const handleMoveToGroupConfirm = useCallback(
    (groupId: string) => {
      const cellCodes = selectedRowKeys.join(',');
      // TODO: 接入 POST /system/deviceGroup/moveCellToGroup.action
      // params: { toGroupId: groupId, ids: cellCodes }
      console.log('move to group:', { groupId, cellCodes });
      void message.success(t('common.operationSuccess'));
      setMoveToGroupModalOpen(false);
      setSelectedRowKeys([]);
      void refetch();
    },
    [selectedRowKeys, message, t, refetch]
  );

  // 导出确认回调
  const handleExportConfirm = useCallback(
    (params: ExportParams) => {
      // TODO: 根据 params.format 选择 API 端点
      // CSV: POST /cell/cpeinfos/exportCellsToCsv.action
      // XLSX: POST /cell/cpeinfos/exportCellsToExcel.action
      // License: POST /cell/cpeinfos/exportEnodebLicenseInfos.action
      console.log('export:', params);
      void message.info(t('common.exportInProgress'));
      setExportModalOpen(false);
    },
    [message, t]
  );

  // 行级操作 — "执行"下拉菜单
  // ── 行级操作处理 ──
  // 原始 JSP 行为：同步→打开同步弹窗；重启→确认弹窗→下发命令→"命令已下发"；
  // 激活/射频→切换状态→"下发成功"→刷新；HaloB→确认需重启→下发；
  // 日志→直接下发→"日志正在收集"；报文→检查已有采集→打开时长弹窗→"报文正在收集"
  const handleRowAction = useCallback(
    (key: string, record: Device) => {
      const sn = record.sn;

      switch (key) {
        // ──── 同步：打开同步参数弹窗（按制式区分） ────
        case 'sync':
          setSyncTarget({ mode: 'single', device: record });
          setSyncModalOpen(true);
          break;

        // ──── 重启：确认弹窗 → 下发 → "命令已下发" ────
        case 'reboot':
          modal.confirm({
            title: t('device.action.reboot'),
            content: t('device.action.rebootConfirm'),
            okType: 'danger',
            onOk: () => {
              // TODO: 接入 cellReboot API (cell_code, isGnb)
              void message.success(t('common.commandSent'));
            },
          });
          break;

        // ──── 日志收集：直接下发 → "日志正在收集" ────
        case 'logCollect':
          // 原始 JSP: confirmImmediateCollectLogFile → POST goImmediateCollectLogFile.action
          // TODO: 接入 logCollect API (serial_number, device_code, execute_type='Immediately')
          void message.success(t('device.action.logCollecting'));
          break;

        // ──── 报文收集：打开收集时长弹窗 → "报文正在收集" ────
        case 'tr069Collect':
          // 原始 JSP: showCollectMessage → 检查是否已有采集 → 选择 5/10 分钟 → start trace
          // TODO: 接入 isExistTracingDevice + trace/start API
          void message.success(t('device.action.tr069Collecting'));
          break;

        // ──── HaloB 开/关：确认需重启 → 下发 ────
        case 'halob':
          modal.confirm({
            title: record.halobFlag ? t('device.action.halobOff') : t('device.action.halobOn'),
            content: t('device.action.halobConfirm'),
            okType: 'danger',
            onOk: () => {
              // TODO: 接入 setCellHalobSwitch API (cell_code, halob_switch)
              void message.success(t('common.commandSent'));
            },
          });
          break;

        // ──── 恢复默认配置：确认弹窗 → 下发 ────
        case 'resetConfig':
          modal.confirm({
            title: t('device.action.resetConfig'),
            content: t('device.action.resetConfigConfirm'),
            okType: 'danger',
            onOk: () => {
              // TODO: 接入 configReset API (cellCode)
              void message.success(t('common.commandSent'));
            },
          });
          break;

        default:
          // 激活/射频 per-cell 操作: activate_0, activate_1, rf_0, rf_1, ...
          if (key.startsWith('activate_') || key.startsWith('rf_')) {
            const [action] = key.split('_');
            const actionLabel = action === 'activate' ? t('device.action.activate') : t('device.action.rfOn');
            modal.confirm({
              title: actionLabel,
              content: action === 'activate'
                ? t('device.action.activateConfirm', { action: actionLabel })
                : t('device.action.rfConfirm', { action: actionLabel }),
              onOk: () => {
                // TODO: 接入 cellModifyActiveStatus / cellModifyRadioStatus API
                // params: { small_cell_code, op_state/radioStatus, cellNumber }
                void message.success(t('common.commandSent'));
              },
            });
          } else {
            void message.info(`${key}: ${sn} — ${t('common.featureInDev')}`);
          }
          break;
      }
    },
    [message, modal, t]
  );

  /** 解析多小区逗号分隔值为 cell 数组 */
  const parseCellValues = useCallback((v: string | undefined | null): string[] => {
    if (!v || v === '--') return [];
    return String(v).split(',').map((s) => s.trim()).filter(Boolean);
  }, []);

  const getActionMenuItems = useCallback(
    (record: Device): MenuProps['items'] => {
      const isOffline = record.connStatus !== 'online';
      const isGSM = record.networkType === 'GSM';

      // 解析多小区状态
      const opCells = parseCellValues(record.opState);
      const rfCells = parseCellValues(record.rfStatus);
      const isMultiCellOp = opCells.length > 1;
      const isMultiCellRf = rfCells.length > 1;

      const items: MenuProps['items'] = [
        // Group 1: 同步 & 报文收集 — 三制式共有
        {
          key: 'sync',
          label: t('device.action.sync'),
          disabled: isOffline,
        },
        {
          key: 'tr069Collect',
          label: t('device.action.tr069Collect'),
          disabled: isOffline,
        },
      ];

      // Group 2: 重启 & 恢复默认配置 — 三制式共有
      items.push(
        { type: 'divider' },
        {
          key: 'reboot',
          label: t('device.action.reboot'),
          disabled: isOffline,
        },
        {
          key: 'resetConfig',
          label: t('device.action.resetConfig'),
          disabled: isOffline,
        },
      );

      // Group 3: 激活/射频/HaloB — eNB 和 gNB
      // GSM 原始页面也支持激活（带 CA 多小区），但不支持射频和 HaloB
      if (!isGSM) {
        items.push({ type: 'divider' });

        // ── 激活/去激活 ──
        if (isMultiCellOp) {
          // 多小区: 展开为子菜单，逐 Cell 控制
          items.push({
            key: 'activate_sub',
            label: t('device.action.activate'),
            disabled: isOffline,
            children: opCells.map((cellState, idx) => {
              const isOn = ['1', 'active'].includes(cellState);
              return {
                key: `activate_${idx}`,
                label: isOn
                  ? t('device.action.deactivateCell', { n: idx + 1 })
                  : t('device.action.activateCell', { n: idx + 1 }),
              };
            }),
          });
        } else {
          const isActive = ['1', 'active'].includes(opCells[0] ?? '');
          items.push({
            key: 'activate_0',
            label: isActive ? t('device.action.deactivate') : t('device.action.activate'),
            disabled: isOffline,
          });
        }

        // ── RF 开/关 ──
        if (isMultiCellRf) {
          // 多射频: 展开为子菜单，逐 RF 控制
          items.push({
            key: 'rf_sub',
            label: t('device.action.rfOn'),
            disabled: isOffline,
            children: rfCells.map((rfState, idx) => {
              const isOn = ['on', '1'].includes(rfState);
              return {
                key: `rf_${idx}`,
                label: isOn
                  ? t('device.action.rfOffCell', { n: idx + 1 })
                  : t('device.action.rfOnCell', { n: idx + 1 }),
              };
            }),
          });
        } else {
          const rfOn = ['on', '1'].includes(rfCells[0] ?? '');
          items.push({
            key: 'rf_0',
            label: rfOn ? t('device.action.rfOff') : t('device.action.rfOn'),
            disabled: isOffline,
          });
        }

        // ── HaloB 开/关 ──
        items.push({
          key: 'halob',
          label: record.halobFlag ? t('device.action.halobOff') : t('device.action.halobOn'),
          disabled: isOffline,
        });
      } else {
        // GSM: 只有激活（也支持多小区 CA）
        items.push({ type: 'divider' });
        if (isMultiCellOp) {
          items.push({
            key: 'activate_sub',
            label: t('device.action.activate'),
            disabled: isOffline,
            children: opCells.map((cellState, idx) => {
              const isOn = ['1', 'active'].includes(cellState);
              return {
                key: `activate_${idx}`,
                label: isOn
                  ? t('device.action.deactivateCell', { n: idx + 1 })
                  : t('device.action.activateCell', { n: idx + 1 }),
              };
            }),
          });
        } else {
          const isActive = ['1', 'active'].includes(opCells[0] ?? '');
          items.push({
            key: 'activate_0',
            label: isActive ? t('device.action.deactivate') : t('device.action.activate'),
            disabled: isOffline,
          });
        }
      }

      // Group 4: 日志 — 三制式共有
      items.push(
        { type: 'divider' },
        {
          key: 'logCollect',
          label: t('device.action.logCollect'),
          disabled: isOffline,
        },
      );

      return items;
    },
    [t, parseCellValues]
  );

  // 格式化时间戳
  const fmtTime = useCallback((v: string) => (v ? new Date(v).toLocaleString('zh-CN') : '-'), []);

  // 格式化在线时长(秒)
  const fmtDuration = useCallback((seconds: number) => {
    if (!seconds) return '-';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : h > 0 ? `${h}h ${m}m` : `${m}m`;
  }, []);

  // 状态值渲染辅助
  const fmtStatus = useCallback(
    (value: string | number | boolean | undefined | null, map: Record<string, { label: string; color: string }>) => {
      const v = String(value ?? '');
      const entry = map[v];
      if (!entry) return v || '-';
      return <Tag color={entry.color}>{entry.label}</Tag>;
    },
    []
  );

  // ── 多小区/多连接状态渲染辅助 ──
  // 原始 JSP: 逗号分隔 "on,off,on" / "1,0,1" 表示多小区状态
  // 汇总显示 + 可点击 [N/M] Popover 查看逐小区明细

  /** 判断多小区汇总状态: 'all_on' | 'mixed' | 'all_off' */
  const getCellSummary = useCallback((cells: string[], onValues: string[]): 'all_on' | 'mixed' | 'all_off' => {
    const hasOn = cells.some((c) => onValues.includes(c));
    const hasOff = cells.some((c) => !onValues.includes(c));
    if (hasOn && hasOff) return 'mixed';
    if (hasOn) return 'all_on';
    return 'all_off';
  }, []);

  /** 渲染多小区状态: 汇总 Tag + [N/M] Popover */
  const renderMultiCellStatus = useCallback(
    (
      value: string | undefined | null,
      onValues: string[],
      labels: { on: string; off: string; title: string },
      colors: { on: string; off: string; mixed: string },
    ) => {
      if (!value || value === '--') return '-';
      const cells = parseCellValues(value);
      if (cells.length === 0) return '-';

      // 单小区 — 直接显示 Tag
      if (cells.length === 1) {
        const isOn = onValues.includes(cells[0]);
        return <Tag color={isOn ? colors.on : colors.off}>{isOn ? labels.on : labels.off}</Tag>;
      }

      // 多小区 — 汇总 + Popover
      const summary = getCellSummary(cells, onValues);
      const activeCount = cells.filter((c) => onValues.includes(c)).length;
      const summaryColor = summary === 'all_on' ? colors.on : summary === 'all_off' ? colors.off : colors.mixed;
      const summaryLabel = summary === 'all_off' ? labels.off : labels.on;

      const popoverContent = (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px 16px', padding: '8px 0' }}>
          {cells.map((cell, idx) => {
            const isOn = onValues.includes(cell);
            return (
              <span key={idx} style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                Cell {idx + 1}:
                <Tag color={isOn ? colors.on : colors.off} style={{ margin: 0 }}>
                  {isOn ? labels.on : labels.off}
                </Tag>
              </span>
            );
          })}
        </div>
      );

      return (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <Tag color={summaryColor}>{summaryLabel}</Tag>
          <Popover title={labels.title} content={popoverContent} trigger="click">
            <span style={{ color: '#4d84ff', cursor: 'pointer' }}>
              [{activeCount}/{cells.length}]
            </span>
          </Popover>
        </span>
      );
    },
    [parseCellValues, getCellSummary]
  );

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      // =====================================================================
      // 公共字段 (common) — 三制式共有或多制式共享
      // =====================================================================
      {
        key: 'sn',
        title: 'SN',
        dataIndex: 'sn',
        width: 180,
        fixed: 'left',
        mono: true,
        copyable: true,
        group: 'common',
        render: (_val, record) => (
          <Link
            style={{ fontFamily: 'monospace' }}
            onClick={() => void navigate(`/device/detail/${record.sn}`)}
          >
            {record.sn}
          </Link>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 100,
        fixed: 'left',
        group: 'common',
        render: (_val, record) => (
          <StatusIndicator
            status={record.connStatus === 'online' ? 'online' : 'offline'}
            text={record.connStatus === 'online' ? t('status.online') : t('status.offline')}
          />
        ),
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const color = SEVERITY_COLOR[record.alarmLevel] ?? 'default';
          const label = SEVERITY_LABEL[record.alarmLevel] ?? record.alarmLevel;
          if (record.alarmLevel && record.alarmLevel !== 'none') {
            // 点击告警跳转到设备详情告警 tab
            return (
              <Tag color={color} style={{ cursor: 'pointer' }} onClick={() => void navigate(`/device/detail/${record.sn}?tab=alarm`)}>
                {label}
              </Tag>
            );
          }
          return <Tag color={color}>{label}</Tag>;
        },
      },
      { key: 'hostName', title: t('device.hostName'), dataIndex: 'hostName', width: 150, ellipsis: true, group: 'common' },
      {
        key: 'networkType',
        title: t('device.radioMode'),
        dataIndex: 'networkType',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const colorMap: Record<string, string> = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
          return <Tag color={colorMap[record.networkType] ?? 'default'}>{record.networkType || '-'}</Tag>;
        },
      },
      {
        key: 'productType',
        title: t('device.productType'),
        dataIndex: 'productType',
        width: 120,
        group: 'common',
        // 原始 JSP: product 字段 — PM-B4860/QAFA/BaiBNX/BSC/BTS 等
        render: (_val, record) => record.productType || '-',
      },
      {
        key: 'platformType',
        title: t('device.platformType'),
        dataIndex: 'platformType',
        width: 140,
        hidden: true,
        group: 'common',
        // 原始 JSP: platformType 字段 — 影响多小区/CA/DC 行为
        render: (_val, record) => record.platformType || '-',
      },
      { key: 'deviceModel', title: t('device.model'), dataIndex: 'deviceModel', width: 120, ellipsis: true, group: 'common' },
      { key: 'softwareVersion', title: t('device.softwareVersion'), dataIndex: 'softwareVersion', width: 140, ellipsis: true, group: 'common' },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true, copyable: true, group: 'common' },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 120, group: 'common' },
      {
        key: 'ipAddress',
        title: t('device.ipAddress'),
        dataIndex: 'ipAddress',
        width: 140,
        hidden: true,
        mono: true,
        copyable: true,
        group: 'common',
        // 原始 JSP: IP 地址可点击，打开设备 Web UI
        render: (_val, record) => {
          const ip = record.ipAddress;
          if (!ip) return '-';
          return (
            <a href={`https://${ip}`} target="_blank" rel="noopener noreferrer"
              style={{ fontFamily: 'monospace', color: '#4d84ff' }}
            >
              {ip}
            </a>
          );
        },
      },
      {
        key: 'onlineTime',
        title: t('device.onlineTime'),
        dataIndex: 'onlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.onlineTime),
      },
      {
        key: 'offlineTime',
        title: t('device.offlineTime'),
        dataIndex: 'offlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.offlineTime),
      },
      {
        key: 'opState',
        title: t('device.opState'),
        dataIndex: 'opState',
        width: 140,
        hidden: true,
        group: 'common',
        // 原始 JSP: 支持多小区 "1,0,1"，汇总 + [N/M] Popover
        // 兼容 active/inactive 文本值和 1/0 数值
        render: (_val, record) => renderMultiCellStatus(
          record.opState,
          ['1', 'active'],
          { on: t('status.active'), off: t('status.inactive'), title: t('device.multiCellStatus') },
          { on: 'success', off: 'error', mixed: 'warning' },
        ),
      },
      {
        key: 'ueCount',
        title: t('device.ueCount'),
        dataIndex: 'ueCount',
        width: 80,
        group: 'common',
        // JSP 行为: eNB >0 且非 CA 站可点击(跳转 UE 详情页)；gNB/GSM 不可点击
        render: (_val, record) => {
          const v = record.ueCount;
          if (v === -1 || v === null || v === undefined) return '--';
          if (v === 0) return '0';
          // 仅 eNB 且非 CA 站支持点击跳转 UE 详情页
          const isEnb = record.networkType === 'eNB';
          const isCaSite = record.platformType?.includes('_CA');
          if (isEnb && !isCaSite) {
            return (
              <Link onClick={() => navigate(`/device/ue-detail/${record.sn}?platformType=${encodeURIComponent(record.platformType ?? '')}&name=${encodeURIComponent(record.name || record.sn)}&ueCount=${record.ueCount}`)}>
                {v}
              </Link>
            );
          }
          return String(v);
        },
      },
      {
        key: 'rfStatus',
        title: t('device.rfStatus'),
        dataIndex: 'rfStatus',
        width: 150,
        hidden: true,
        group: 'common',
        // 原始 JSP: 支持多小区 "on,off,on"，汇总 + [N/M] Popover
        render: (_val, record) => renderMultiCellStatus(
          record.rfStatus,
          ['on', '1'],
          { on: t('status.rfOn'), off: t('status.rfOff'), title: t('device.multiCellStatus') },
          { on: 'success', off: 'error', mixed: 'warning' },
        ),
      },
      {
        key: 'syncStatus',
        title: t('device.syncStatus'),
        dataIndex: 'syncStatus',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          const v = record.syncStatus;
          if (!v) return '-';
          if (v === 'not synchronized') {
            return <Tag color="error" style={{ fontWeight: 600 }}>{t('status.notSynchronized')}</Tag>;
          }
          return fmtStatus(v, {
            synchronized: { label: t('status.synchronized'), color: 'success' },
            'GPS synchronized': { label: 'GPS ' + t('status.synchronized'), color: 'success' },
            '1588 synchronized': { label: '1588 ' + t('status.synchronized'), color: 'success' },
            'REM synchronized': { label: 'REM ' + t('status.synchronized'), color: 'success' },
          });
        },
      },
      { key: 'productName', title: t('device.productName'), dataIndex: 'productName', width: 130, hidden: true, group: 'common' },
      { key: 'firmwareVersion', title: t('device.firmwareVersion'), dataIndex: 'firmwareVersion', width: 140, hidden: true, ellipsis: true, group: 'common' },
      {
        key: 'onlineDuration',
        title: t('device.onlineDuration'),
        dataIndex: 'onlineDuration',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtDuration(record.onlineDuration),
      },
      { key: 'upTime', title: t('device.upTime'), dataIndex: 'upTime', width: 120, hidden: true, group: 'common' },
      {
        key: 'firstOnlineTime',
        title: t('device.firstOnlineTime'),
        dataIndex: 'firstOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.firstOnlineTime),
      },
      {
        key: 'lastInformTime',
        title: t('device.lastInformTime'),
        dataIndex: 'lastInformTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.lastInformTime),
      },
      {
        key: 'lastOnlineTime',
        title: t('device.lastOnline'),
        dataIndex: 'lastOnlineTime',
        width: 165,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtTime(record.lastOnlineTime),
      },
      { key: 'siteName', title: t('device.siteName'), dataIndex: 'siteName', width: 130, hidden: true, ellipsis: true, group: 'common' },
      { key: 'remark', title: t('device.remark'), dataIndex: 'remark', width: 185, hidden: true, ellipsis: true, group: 'common', headerRender: remarkHeaderRender },
      {
        key: 'longitude',
        title: t('device.longitude'),
        dataIndex: 'longitude',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          const v = record.longitude;
          if (v === null || v === undefined) return '--';
          if (record.networkType !== 'eNB') return v;
          return (
            <Space size={4}>
              <Popconfirm
                title={`${t('device.longitude')}: ${record.longitude}   ${t('device.latitude')}: ${record.latitude}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
                description={t('device.gpsInconsistent')}
                onConfirm={() => void message.success(t('device.gpsSyncSuccess'))}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <WarningOutlined style={{ color: '#faad14', cursor: 'pointer' }} />
              </Popconfirm>
              {v}
            </Space>
          );
        },
      },
      {
        key: 'latitude',
        title: t('device.latitude'),
        dataIndex: 'latitude',
        width: 130,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          const v = record.latitude;
          if (v === null || v === undefined) return '--';
          if (record.networkType !== 'eNB') return v;
          return (
            <Space size={4}>
              <Popconfirm
                title={`${t('device.longitude')}: ${record.longitude}   ${t('device.latitude')}: ${record.latitude}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
                description={t('device.gpsInconsistent')}
                onConfirm={() => void message.success(t('device.gpsSyncSuccess'))}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <WarningOutlined style={{ color: '#faad14', cursor: 'pointer' }} />
              </Popconfirm>
              {v}
            </Space>
          );
        },
      },
      {
        key: 'gpsHeight',
        title: t('device.gpsHeight'),
        dataIndex: 'gpsHeight',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          const v = record.gpsHeight;
          if (v === null || v === undefined) return '--';
          if (record.networkType !== 'eNB') return v;
          return (
            <Space size={4}>
              <Popconfirm
                title={`${t('device.longitude')}: ${record.longitude}   ${t('device.latitude')}: ${record.latitude}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
                description={t('device.gpsInconsistent')}
                onConfirm={() => void message.success(t('device.gpsSyncSuccess'))}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <WarningOutlined style={{ color: '#faad14', cursor: 'pointer' }} />
              </Popconfirm>
              {v}
            </Space>
          );
        },
      },
      {
        key: 'gpsSatelliteCount',
        title: t('device.gpsSatelliteCount'),
        dataIndex: 'gpsSatelliteCount',
        width: 110,
        hidden: true,
        group: 'common',
        // 原始 JSP: 有卫星详情时可点击查看信号表（卫星号、信号强度）
        render: (_val, record) => {
          const v = record.gpsSatelliteCount;
          if (v === null || v === undefined) return '-';
          // TODO: 判断 hasSatelliteDetail 并点击打开卫星详情面板 (getSatellitesDataList.action)
          return v > 0 ? <Link onClick={() => void navigate(`/device/detail/${record.sn}?tab=gps`)}>{v}</Link> : String(v);
        },
      },
      { key: 'installAddress', title: t('device.installAddress'), dataIndex: 'installAddress', width: 180, hidden: true, ellipsis: true, group: 'common' },
      { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 80, hidden: true, group: 'common' },
      { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 80, hidden: true, group: 'common' },
      { key: 'band', title: 'Band', dataIndex: 'band', width: 100, hidden: true, group: 'common' },
      { key: 'dlEarfcn', title: t('device.dlEarfcn'), dataIndex: 'dlEarfcn', width: 110, hidden: true, group: 'common' },
      { key: 'ulEarfcn', title: t('device.ulEarfcn'), dataIndex: 'ulEarfcn', width: 110, hidden: true, group: 'common' },
      { key: 'networkModel', title: t('device.networkModel'), dataIndex: 'networkModel', width: 110, hidden: true, group: 'common' },
      { key: 'txPower', title: 'Tx Power', dataIndex: 'txPower', width: 100, hidden: true, group: 'common' },
      {
        key: 'halobFlag',
        title: 'HaloB',
        dataIndex: 'halobFlag',
        width: 90,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          if (record.halobFlag === undefined || record.halobFlag === null) return '-';
          return <Tag color={record.halobFlag ? 'success' : 'default'}>{record.halobFlag ? t('status.enabled') : t('status.disabled')}</Tag>;
        },
      },
      {
        key: 'adminState',
        title: 'Admin State',
        dataIndex: 'adminState',
        width: 120,
        hidden: true,
        group: 'common',
        // 原始 gNB JSP: 1→Locked, 2→Unlocked, 3→ShuttingDown
        render: (_val, record) => fmtStatus(record.adminState, {
          '1': { label: 'Locked', color: 'warning' },
          '2': { label: 'Unlocked', color: 'success' },
          '3': { label: 'ShuttingDown', color: 'error' },
        }),
      },
      { key: 'ipsecAddr', title: t('device.ipsecAddr'), dataIndex: 'ipsecAddr', width: 140, hidden: true, mono: true, group: 'common' },

      // =====================================================================
      // 操作列
      // =====================================================================
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 100,
        fixed: 'right',
        render: (_val, record) => (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => void navigate(`/device/detail/${record.sn}`)}
            >
              {t('common.detail')}
            </Button>
            <Dropdown
              menu={{
                items: getActionMenuItems(record),
                onClick: ({ key }) => handleRowAction(key, record),
              }}
              trigger={['click']}
            >
              <Button type="text" size="small" icon={<MoreOutlined style={{ fontSize: 16 }} />} />
            </Dropdown>
          </Space>
        ),
      },
    ],
    [navigate, t, fmtTime, fmtDuration, fmtStatus, renderMultiCellStatus, SEVERITY_LABEL, getActionMenuItems, handleRowAction, remarkHeaderRender]
  );

  const batchActions = useMemo((): BatchAction[] => {
    const actions: BatchAction[] = [
      {
        key: 'move-to-group',
        label: t('common.moveToGroup'),
        icon: <SwapOutlined />,
        onClick: () => handleMoveToGroup(),
      },
    ];

    // 批量同步仅在所有选中设备为同一制式时显示
    if (selectedNetworkType) {
      actions.push({
        key: 'batch-sync',
        label: `${t('common.batchSync')} (${selectedNetworkType})`,
        icon: <SyncOutlined />,
        onClick: () => handleBatchSync(),
      });
    }

    actions.push(
      {
        key: 'batch-reboot',
        label: t('common.batchReboot'),
        icon: <ReloadOutlined />,
        onClick: (keys) => handleBatchReboot(keys as string[]),
      },
      {
        key: 'recycle',
        label: t('common.recycleBin'),
        icon: <RestOutlined />,
        danger: true,
        onClick: (keys) => handleRecycle(keys as string[]),
      },
    );

    return actions;
  }, [handleMoveToGroup, handleBatchSync, handleBatchReboot, handleRecycle, selectedNetworkType, t]);

  return (
    <ListPageLayout
      title={t('nav.device.list')}
      extra={
        <Button
          type="primary"
          icon={<ExportOutlined />}
          onClick={() => setExportModalOpen(true)}
        >
          {t('common.export')}
        </Button>
      }
    >
      <FilterBar
        filterId="device-list"
        fields={FILTER_FIELDS}
        onSearch={handleSearch}
        onReset={handleReset}
        collapsedRows={1}
      />

      <StatisticsPanel items={statsItems} style={{ marginBottom: 8 }} />

      <DataTable<Device>
        tableId="device-list-table"
        columns={columns}
        dataSource={devices}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={total}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={(page, size) => {
          setCurrentPage(page);
          setPageSize(size);
        }}
        batchActions={batchActions}
        onRefresh={() => void refetch()}
        defaultDensity="compact"
      />

      <SyncParamsModal
        open={syncModalOpen}
        networkType={
          syncTarget.mode === 'single'
            ? (syncTarget.device.networkType as SyncNetworkType) ?? 'eNB'
            : selectedNetworkType ?? 'eNB'
        }
        onClose={() => setSyncModalOpen(false)}
        onConfirm={handleSyncConfirm}
      />

      <MoveToGroupModal
        open={moveToGroupModalOpen}
        onClose={() => setMoveToGroupModalOpen(false)}
        onConfirm={handleMoveToGroupConfirm}
        selectedCount={selectedRowKeys.length}
      />

      <ExportModal
        open={exportModalOpen}
        onClose={() => setExportModalOpen(false)}
        onConfirm={handleExportConfirm}
      />

    </ListPageLayout>
  );
}
