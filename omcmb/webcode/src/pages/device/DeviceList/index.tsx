import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { App, Button, Card, Drawer, Input, Modal, Popconfirm, Popover, Progress, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AlertOutlined,
  CheckOutlined,
  CloseOutlined,
  EditOutlined,
  ExportOutlined,
  EyeOutlined,
  FileTextOutlined,
  ReloadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import StatisticsPanel from '@/components/StatisticsPanel';
import StatusIndicator from '@/components/StatusIndicator';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { prefetchDeviceDetailContext, useDeviceList, useBatchRebootDevices, useDeviceGroups } from '@core/hooks/api/useDevices';
import { useProductList } from '@core/hooks/api/useProducts';
import { useDictionaryBatch } from '@core/hooks/api/useSystem';
import { resolveNetworkTypeLabel } from '@core/utils/networkType';
import { activationStatusOf } from '@core/utils/activationStatus';
import { useTriggerAlarmSync } from '@core/hooks/api/useAlarms';
import { useCreateUnifiedFileTransferTask } from '@core/hooks/api/useUnifiedFileTransfer';
import { useDownloadStationLog } from '@core/hooks/api/useStationLog';
import { stationLogApi } from '@core/services/api/stationLogApi';
import { deviceApi } from '@core/services/api/deviceApi';
import { createApiSwitch } from '@core/services/apiSwitch';
import { deviceService } from '@core/mock/services/deviceService';
import {
  resolveVisibleExportColumns,
  buildCsvContent,
  triggerCsvDownload,
  triggerXlsxDownload,
  exportTimestamp,
  fetchAllPaged,
} from './deviceExport';
import { useT } from '@/hooks/useT';
import { useUserStore } from '@core/store/userStore';
import { useAppStore } from '@core/store/appStore';
import { buildDefaultUfteTaskName } from '@/pages/transfer/shared';
import dayjs from 'dayjs';
import { buildBatchTaskTypeMap, batchActionHasDetail } from './deviceBatchTask';
import type { Device } from '@core/types/device';
import { formatSystemTime } from '@core/utils/systemTime';

const { Link } = Typography;

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'red',
  major: 'orange',
  minor: 'gold',
  warning: 'blue',
  none: 'default',
};

// 列表导出复用 useDeviceList 背后同一套 mock/real 切换,保证导出与展示口径一致。
const exportDeviceApi = createApiSwitch(deviceService as unknown as typeof deviceApi, deviceApi);
// DataTable tableId,导出时据此读取"列设置"localStorage(须与 <DataTable tableId> 一致)。
const DEVICE_LIST_TABLE_ID = 'device-list-table';

// 筛选下拉框 name → 表格列 key 映射:列设置隐藏该列时,对应筛选下拉一并隐藏
// (用户决策 2026-06-09)。searchText 无对应列、不入表 → 始终显示。
const FILTER_COLUMN_MAP: Record<string, string> = {
  isOnline: 'connStatus',
  opState: 'opState',
  networkType: 'networkType',
  productId: 'deviceModel',
  productModel: 'productClass',
  modelName: 'deviceModel',
  softwareVersion: 'softwareVersion',
  groupId: 'groupName',
};

// i18n 翻译函数签名（与 useT 返回值一致）：t(id, values?) → 已格式化字符串。
// 离线时长格式化抽离为纯文本核心 offlineDurationText，render 版仅在外面套 <Tag>，
// 避免文案口径在两处漂移（#226：原先硬编码 `${years}年` 等不随语言切换）。
type TFn = (id: string, values?: Record<string, string | number>) => string;

/**
 * 离线时长的纯文本版(导出 + 渲染共用)——文案全部走 i18n，切语言即时生效。
 * @param t 翻译函数（来自 useT）
 * @param days 离线天数
 * @param hours 剩余小时数 (0-23)
 * @param minutes 剩余分钟数 (0-59)
 */
function offlineDurationText(t: TFn, days?: number, hours?: number, minutes?: number): string {
  if (days === undefined || days === null) return '-';
  if (days >= 365) {
    const years = Math.floor(days / 365);
    const remainDays = days % 365;
    return remainDays > 0
      ? t('device.duration.yearsDays', { years, days: remainDays })
      : t('device.duration.years', { years });
  }
  if (days >= 30) {
    const months = Math.floor(days / 30);
    const remainDays = days % 30;
    return remainDays > 0
      ? t('device.duration.monthsDays', { months, days: remainDays })
      : t('device.duration.months', { months });
  }
  if (days > 0) {
    return hours && hours > 0
      ? t('device.duration.daysHours', { days, hours })
      : t('device.duration.days', { days });
  }
  if (hours && hours > 0) {
    return minutes && minutes > 0
      ? t('device.duration.hoursMinutes', { hours, minutes })
      : t('device.duration.hours', { hours });
  }
  if (minutes && minutes > 0) return t('device.duration.minutes', { minutes });
  return t('device.duration.lessThanMinute');
}

/**
 * 离线时长渲染版：在纯文本基础上套配色 <Tag>。颜色阈值与文本口径解耦。
 */
function formatOfflineDuration(t: TFn, days?: number, hours?: number, minutes?: number): React.ReactNode {
  if (days === undefined || days === null) return '-';
  const text = offlineDurationText(t, days, hours, minutes);
  let color = 'default';
  if (days >= 365) color = 'red';
  else if (days >= 30) color = 'orange';
  else if (days > 0) color = days >= 7 ? 'orange' : 'gold';
  else if (hours && hours > 0) color = 'gold';
  return <Tag color={color}>{text}</Tag>;
}

// 哪些 filter 字段在 URL 里以 CSV 形式编码、需要解析回数组（与 FILTER_FIELDS
// 中 type='multi-select' 的项一一对应）。不在这个集合里的字段（典型如
// searchText 自由文本，用户可能用逗号分隔多关键字）保持字符串原样 ——
// 之前用 "value.includes(',')" 一刀切会把 searchText 也 split 成数组，导致
// axios 把 search 序列化成 search[]=a&search[]=b，后端 c.Query("search") 读
// 不到，于是同一次"搜索"先后发两个 API、第二个还把筛选丢了。
const URL_ARRAY_FIELDS = new Set<string>([
  'productModel',
  'modelName',
  'softwareVersion',
  'groupId',
]);

function parseUrlValue(key: string, value: string): unknown {
  return URL_ARRAY_FIELDS.has(key) ? value.split(',').filter(Boolean) : value;
}

export default function DeviceList() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { message, modal } = App.useApp();

  const prefetchDeviceDetailEntry = useCallback((device: Device) => {
    void import('@/pages/device/DeviceDetail');
    void prefetchDeviceDetailContext(queryClient, device);
  }, [queryClient]);

  const openDeviceDetail = useCallback((device: Device, tab?: string) => {
    prefetchDeviceDetailEntry(device);
    const suffix = tab ? `?tab=${tab}` : '';
    void navigate(`/device/detail/${device.sn}${suffix}`);
  }, [navigate, prefetchDeviceDetailEntry]);

  // 从 URL 恢复搜索条件和分页
  const [currentPage, setCurrentPage] = useState(() => {
    const page = searchParams.get('page');
    return page ? parseInt(page, 10) : 1;
  });
  const [pageSize, setPageSize] = useState(() => {
    const size = searchParams.get('pageSize');
    // 性能优化：默认 20 条而非 100 条，减少首屏 DOM 节点数量 80%
    return size ? parseInt(size, 10) : 20;
  });
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>(() => {
    const params: Record<string, unknown> = {};
    searchParams.forEach((value, key) => {
      if (key !== 'page' && key !== 'pageSize') {
        params[key] = parseUrlValue(key, value);
      }
    });
    return params;
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  // 导出确认弹窗（替代原手写下拉菜单——后者在亮色主题下白底白字不可见）。
  const [exportModalOpen, setExportModalOpen] = useState(false);

  // 同步 URL 参数到 filterParams（解决返回时 state 未恢复的问题）
  // 性能优化：使用浅比较替代 JSON.stringify 深度比较，避免循环依赖
  useEffect(() => {
    const params: Record<string, unknown> = {};
    searchParams.forEach((value, key) => {
      if (key !== 'page' && key !== 'pageSize') {
        params[key] = parseUrlValue(key, value);
      }
    });
    // 浅比较：先比较 key 数量，再逐个比较 value
    const currentKeys = Object.keys(filterParams);
    const newKeys = Object.keys(params);
    if (currentKeys.length !== newKeys.length) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setFilterParams(params);
      return;
    }
    let changed = false;
    for (const key of newKeys) {
      if (params[key] !== filterParams[key]) {
        changed = true;
        break;
      }
    }
    if (changed) {
      setFilterParams(params);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);  // 移除 filterParams 依赖，避免循环触发

  // 不再在挂载时自动把 sessionStorage 灌回 URL —— 旧实现会让"上次会话留下的过时
  // 筛选值"（比如 T-0162 已废弃的 connStatus、或新数据里不存在的 softwareVersion）
  // 在用户首次进入页面时就被应用，后端返回 0 条 → 表象就是"首次进入列表没数据，
  // 点搜索才有"。
  //
  // 现在的恢复策略：
  //   - 表单视觉值：由 FilterBar 自己挂载时从 sessionStorage 回填到 form fields
  //     （只 setFieldsValue，不触发 onSearch）—— 用户能看到上次的筛选条件
  //   - 实际查询：首次进入页面用空 filter 拿全量数据；用户主动点"搜索"才把表单
  //     当前值变成 filterParams 并写 URL
  //   - 返回导航：URL 里有 ?key=value 时由上面 useEffect 同步回 filterParams，
  //     不依赖 sessionStorage

  // 本地任务面板状态
  type TaskStatus = 'pending' | 'running' | 'success' | 'failed';
  interface LocalTask {
    id: string;
    sn: string;
    deviceName: string;
    type: string;
    status: TaskStatus;
    progress: number;
    message?: string;
    logContent?: string;
    hasDetail?: boolean; // 是否有详情可查看（只有收集操作才有）
  }

  // 实时刷新状态：受 DataTable 工具栏「开启实时刷新」按钮控制。
  // 历史 bug：只解构出 [autoRefresh] 没拿 setter，导致按钮翻不动这个值，
  // refetchInterval 永远是 undefined → 实时刷新等于摆设。
  const [autoRefresh, setAutoRefresh] = useState(false);

  // 收集任务抽屉状态
  const [collectDrawerOpen, setCollectDrawerOpen] = useState(false);
  const [collectTasks, setCollectTasks] = useState<LocalTask[]>([]);
  const [collectDrawerTitle, setCollectDrawerTitle] = useState(''); // 抽屉标题

  // 日志详情弹窗状态
  const [logModalOpen, setLogModalOpen] = useState(false);
  const [currentLogTask, setCurrentLogTask] = useState<LocalTask | null>(null);

  // 打开日志详情
  const handleViewLog = useCallback((task: LocalTask) => {
    setCurrentLogTask(task);
    setLogModalOpen(true);
  }, []);

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

  // remarkHeaderRender 保持 useMemo，因为 headerRender 需要 ReactNode 而非函数
  // 编辑状态变化不频繁，性能开销可接受
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
  // remark 列定义当前注释隐藏（见下方 columns 定义处说明）；remarkHeaderRender 有意保留以便恢复列时直接接回。
  // 此处显式引用消除 TS6133「声明未使用」，恢复 remark 列时删除本行即可。
  void remarkHeaderRender;

  const queryParams = useMemo(
    () => ({ ...filterParams, page: currentPage, pageSize } as Parameters<typeof useDeviceList>[0]),
    [filterParams, currentPage, pageSize]
  );

  const { data, isLoading, refetch } = useDeviceList(queryParams, {
    refetchInterval: autoRefresh ? 5000 : undefined,
  });
  const batchReboot = useBatchRebootDevices();
  const triggerAlarmSync = useTriggerAlarmSync();
  const createUfteTask = useCreateUnifiedFileTransferTask();
  const downloadStationLog = useDownloadStationLog();
  const currentUser = useUserStore((s) => s.currentUser);
  const appLocale = useAppStore((s) => s.locale);
  const taskNameUser = currentUser?.username || currentUser?.displayName || 'user';
  // 性能优化：使用 useMemo 避免每次渲染创建新引用，防止下游 callback/useMemo 依赖变化
  const devices = useMemo(() => data?.items ?? [], [data?.items]);
  const total = data?.total ?? 0;
  const stats = useMemo(() => data?.stats ?? { total: 0, online: 0, offline: 0, alarmed: 0, online_count: 0, offline_count: 0 }, [data?.stats]);

  // R6b: 设备分组下拉接入 device/group API（device-list-and-group-improvements-20260520.md R6b）
  const { data: groupsResp } = useDeviceGroups();
  const groupOptions = useMemo(() => {
    const groups = groupsResp?.groups ?? [];
    // 仅 L2 子分组可作为设备过滤目标（L1 是容器）；用 parentName / name 双层展示便于辨识
    const byId = new Map(groups.map((g) => [g.id, g]));
    return groups
      .filter((g) => g.parentId !== null) // 排除 L1 根分组
      .map((g) => {
        const parent = g.parentId ? byId.get(g.parentId) : null;
        return {
          label: parent ? `${parent.name} / ${g.name}` : g.name,
          value: g.id,
        };
      });
  }, [groupsResp]);

  // R6c: 字典驱动 — 在线状态 / 激活状态 / 网络制式 / 产品类型
  // 字典 code 与种子数据在 migrations/000136 / 000137 维护。
  //
  // T-0162: conn_status 字典已废弃，替换为：
  //   - lifecycle_state（生命周期 6 状态）
  //   - is_online（在线 2 状态：true/false）
  // 新增 3 个字典（device_model / software_version / firmware_version）由
  // seed/000137 初始化为现有 devices/device_parameters 表的 distinct 值。
  // 性能优化：使用批量查询一次获取所有字典，减少 HTTP 请求
  const { data: batchDicts } = useDictionaryBatch([
    'is_online',
    'op_state',
    'network_type',
    'product_class',
    'device_model',
    'software_version',
    'firmware_version',
  ]);

  const isOnlineDict = batchDicts?.['is_online'];
  const opStateDict = batchDicts?.['op_state'];
  const networkTypeDict = batchDicts?.['network_type'];
  const productClassDict = batchDicts?.['product_class'];
  const deviceModelDict = batchDicts?.['device_model'];
  const softwareVersionDict = batchDicts?.['software_version'];

  const dictToOptions = useCallback(
    (dict: { sysDictionaryDetails?: { label: string; value: string }[] } | undefined) =>
      (dict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value })),
    [],
  );

  // 产品名称下拉：选项来自 /products（label=产品名称，value=产品 UUID → devices.product_id）。
  // 与「产品类型」(product_class 字典) 不同，此处按产品装配件主键过滤。
  const { data: productListResp } = useProductList();
  const productOptions = useMemo(
    () => (productListResp?.items ?? []).map((p) => ({ label: p.name, value: p.id })),
    [productListResp],
  );

  // 性能优化：SEVERITY_LABEL 改为函数调用，移除 useMemo
  // 仅 5 个字符串映射，计算开销可忽略，避免依赖 t 函数导致频繁重建
  const getSeverityLabel = useCallback((severity: string): string => {
    const labels: Record<string, string> = {
      critical: t('alarm.severity.critical'),
      major: t('alarm.severity.major'),
      minor: t('alarm.severity.minor'),
      warning: t('alarm.severity.warning'),
      none: t('alarm.severity.none'),
    };
    return labels[severity] || severity;
  }, [t]);

  // 状态值映射：兼容多种可能的数据格式，修复乱码问题
  const mapConnStatus = useCallback((status: string | undefined | null): 'online' | 'offline' => {
    if (!status) return 'offline';

    // 标准化处理：转小写、去空格
    const normalized = String(status).toLowerCase().trim();

    // 兼容多种可能的状态值
    const onlineValues = ['online', '1', 'true', 'yes', 'connected', 'active'];
    const offlineValues = ['offline', '0', 'false', 'no', 'disconnected', 'inactive'];

    if (onlineValues.includes(normalized)) return 'online';
    if (offlineValues.includes(normalized)) return 'offline';

    // 兜底：无法识别时返回 offline
    return 'offline';
  }, []);


  const FILTER_FIELDS: FilterField[] = useMemo(() => [
    // --- 搜索项：文本搜索覆盖 SN/名称/IP/MAC/ECI/PCI ---
    // 后端 BuildSearchOR 支持英文逗号分隔多关键字（最多 50 个），
    // placeholder 提示用户可粘多 SN 一次搜。
    {
      name: 'searchText',
      label: t('filter.searchText'),
      type: 'input',
      placeholder: t('filter.searchText.multiHint'),
      width: 400,
    },

    // --- 筛选项：三制式公共（默认显示） ---
    // T-0162: 「生命周期」筛选已下线（用户反馈业务场景里实际只关心实时
    // 在线/离线，6 状态生命周期对前端筛选过细）。字典 lifecycle_state 在
    // 后端仍保留供详情页与统计 by_lifecycle 使用。
    {
      name: 'isOnline',
      label: t('device.connStatus'),
      type: 'select',
      width: 160,
      options: dictToOptions(isOnlineDict),
    },
    {
      name: 'opState',
      label: t('device.opState'),
      type: 'select',
      width: 160,
      options: dictToOptions(opStateDict),
    },
    {
      name: 'networkType',
      label: t('device.radioMode'),
      type: 'select',
      width: 160,
      options: dictToOptions(networkTypeDict),
    },
    // 产品名称（按 devices.product_id 过滤，下拉来自 /products，value=产品 UUID）——置于产品类型之前。
    {
      name: 'productId',
      label: t('device.productName'),
      type: 'select',
      width: 160,
      options: productOptions,
    },
    {
      name: 'productModel',
      label: t('device.productClass'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(productClassDict),
    },

    // --- 筛选项：三制式公共（默认折叠） ---
    // T-0162: 3 个原本写死 [] 的下拉接入字典：device_model / software_version /
    // firmware_version。字典初始化数据由 seed/000137 从现有 devices /
    // device_parameters 表 distinct 灌入，保证用户任意选一项后端 filter 一定
    // 命中至少一条设备。后续新版本进来时管理员手动补字典词条。
    {
      name: 'modelName',
      label: t('device.model'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(deviceModelDict),
    },
    {
      name: 'softwareVersion',
      label: t('device.softwareVersion'),
      type: 'multi-select',
      width: 160,
      options: dictToOptions(softwareVersionDict),
    },
    // R4 + R6b: groupId 接入 useDeviceGroups
    // width:400 与第一行 searchText 输入框对齐——分组名称长度普遍超过 160（含层级
    // "Region A / Subgroup B" 形式），160 时多选 chip 被截断成 "...""，体验差。
    {
      name: 'groupId',
      label: t('device.groupName'),
      type: 'multi-select',
      width: 400,
      options: groupOptions,
    },
  ], [
    t,
    productOptions,
    isOnlineDict,
    opStateDict,
    networkTypeDict,
    productClassDict,
    deviceModelDict,
    softwareVersionDict,
    groupOptions,
    dictToOptions,
  ]);

  // 2026-06-03 用户决策:下拉(select/multi-select)提示统一在前面加「请选择」(无显式 placeholder 时回退 label,此处前置请选择)。
  const filterFields = useMemo(
    () =>
      FILTER_FIELDS.map((f) =>
        (f.type === 'select' || f.type === 'multi-select') && !f.placeholder
          ? { ...f, placeholder: `${t('common.pleaseSelect')}${f.label ?? ''}` }
          : f
      ),
    [FILTER_FIELDS, t]
  );

  // 列设置隐藏某列 → 对应搜索下拉框一并隐藏(FILTER_COLUMN_MAP 映射;searchText 无映射,始终显示)。
  // hiddenColumnKeys 由 <DataTable onHiddenColumnsChange> 在列设置变化时抬上来。
  const [hiddenColumnKeys, setHiddenColumnKeys] = useState<string[]>([]);
  const visibleFilterFields = useMemo(
    () =>
      filterFields.filter((f) => {
        const colKey = FILTER_COLUMN_MAP[f.name];
        return !colKey || !hiddenColumnKeys.includes(colKey);
      }),
    [filterFields, hiddenColumnKeys]
  );

  // 统计面板 — 基于筛选条件的全量统计（由后端 stats 字段返回，非当前页）
  // T-0162: 优先用 online_count / offline_count（与 backend DeviceListStats 1:1）；
  // 老 stats.online / stats.offline 字段在新前端不再使用（仅 mapListResponse 内部
  // 当 fallback 保留），新 UI 直读 stats.online_count。
  const statsItems = useMemo(() => [
    { label: t('device.count.total'), value: stats.total },
    { label: t('status.online'), value: stats.online_count ?? stats.online ?? 0, color: '#52C41A' },
    { label: t('status.offline'), value: stats.offline_count ?? stats.offline ?? 0, color: '#8C8C8C' },
    { label: t('common.hasAlarm'), value: stats.alarmed, color: '#FA8C16' },
  ], [stats, t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    // 关键字上限校验：后端 BuildSearchOR 限定 ≤50 keyword × 6 fields = 300 ILIKE
    // 子句，超过会被静默截断。前端这里做截断 + 用户提示，让感知明确。
    let effective: Record<string, unknown> = values;
    const raw = values.searchText;
    if (typeof raw === 'string' && raw.includes(',')) {
      const keywords = raw
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      if (keywords.length > 50) {
        void message.warning(t('filter.searchText.tooManyKeywords'));
        effective = { ...values, searchText: keywords.slice(0, 50).join(',') };
      }
    }

    setFilterParams(effective);
    setCurrentPage(1);
    // 同步到 URL
    const newParams = new URLSearchParams();
    Object.entries(effective).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        if (Array.isArray(value)) {
          if (value.length > 0) {
            newParams.set(key, value.join(','));
          }
        } else {
          newParams.set(key, String(value));
        }
      }
    });
    setSearchParams(newParams);
  }, [setSearchParams, message, t]);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setCurrentPage(1);
    // 清空 URL 参数
    setSearchParams(new URLSearchParams());
  }, [setSearchParams]);

  // 批量操作通用确认弹窗
  const handleBatchAction = useCallback(
    (actionLabel: string, ids: React.Key[], actionKey?: string) => {
      modal.confirm({
        title: t('common.confirm'),
        content: t('device.batch.actionConfirm', { action: actionLabel, count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        onOk: async () => {
          // 获取选中设备的详细信息
          const selectedDevices = devices.filter((d) => ids.includes(d.id));

          // 任务类型映射（抽到 deviceBatchTask.ts 便于单测，不含已移除的 tr069-collect）
          const taskTypeMap = buildBatchTaskTypeMap(t);

          const newTasks: LocalTask[] = selectedDevices.map((device, index) => ({
            id: `${actionKey}-${device.sn}-${Date.now()}-${index}`,
            sn: device.sn,
            deviceName: device.name || device.hostName || device.sn,
            type: taskTypeMap[actionKey ?? ''] || actionLabel,
            status: 'pending' as TaskStatus,
            progress: 0,
            hasDetail: batchActionHasDetail(actionKey), // 只有日志采集才有详情
          }));

          // 日志采集：UFTE 创建 RUNTIME_LOG_COLLECT 任务后自动跳转到「文件传输 →
          // 任务管理」的「运行日志采集」tab，让用户立刻看到刚创建的任务进度。
          if (actionKey === 'batch-log-collect') {
            try {
              // 任务名遵循 UFTE 全局统一规则：<i18n prefix>_<user>_<YYYY-MM-DD HH:mm:ss>
              // 中文环境 "运行日志_admin_..."；英文 "RuntimeLog_admin_..."
              const taskName = buildDefaultUfteTaskName(
                'RUNTIME_LOG_COLLECT',
                taskNameUser,
                appLocale,
                dayjs().format('YYYY-MM-DD HH:mm:ss'),
              );
              await createUfteTask.mutateAsync({
                taskName,
                typeCode: 'RUNTIME_LOG_COLLECT',
                deviceIds: selectedDevices.map((d) => d.id),
                deviceCount: selectedDevices.length,
                executionMode: 'immediate',
              });
              void message.success(t('ufte.taskCreatedAndNavigate'));
              // category=station_log + typeCode=RUNTIME_LOG_COLLECT —— FileTransferCenter
              // 初始化时按 URL 还原 selectedCategory + selectedTypeCode，定位到具体 tab。
              navigate('/transfer/center?category=station_log&typeCode=RUNTIME_LOG_COLLECT');
            } catch {
              void message.error(t('common.operationFailed'));
            }
            setSelectedRowKeys([]);
            return;
          }

          {
            // 同步/重启操作：使用右侧抽屉（TR069 抓包入口已移除，#179）
            setCollectDrawerTitle(t('task.taskProgress')); // 任务进度
            setCollectTasks(newTasks);
            setCollectDrawerOpen(true);

            // 模拟任务进度
            newTasks.forEach((task, index) => {
              setTimeout(() => {
                setCollectTasks((prev) => prev.map((item) =>
                  item.id === task.id ? { ...item, status: 'running', progress: 10 } : item
                ));

                const progressInterval = setInterval(() => {
                  setCollectTasks((prev) => prev.map((item) => {
                    if (item.id !== task.id) return item;
                    if (item.progress >= 100) {
                      clearInterval(progressInterval);
                      return item;
                    }
                    const randomProgress = Math.random() * 15 + 10;
                    return { ...item, progress: Math.min(item.progress + randomProgress, 90) };
                  }));
                }, 200);

                const completeTime = 1000 + Math.random() * 1000;

                setTimeout(() => {
                  clearInterval(progressInterval);
                  const success = Math.random() > 0.1;
                  // 生成操作日志
                  const timestamp = new Date().toISOString();
                  const logContent = success
                    ? `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] INFO: ${t('task.log.success')}`
                    : `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${task.sn}\n[${timestamp}] ERROR: ${t('task.log.failed')}`;
                  setCollectTasks((prev) => prev.map((item) =>
                    item.id === task.id ? {
                      ...item,
                      status: success ? 'success' : 'failed',
                      progress: 100,
                      message: success ? t('task.status.completed') : t('common.failed'),
                      logContent,
                    } : item
                  ));
                }, completeTime);
              }, index * 200);
            });
          }

          if (actionKey === 'batch-reboot') {
            try {
              await batchReboot.mutateAsync(ids.map(String));
              void message.success(t('common.commandSent'));
            } catch {
              void message.error(t('common.operationFailed'));
            }
          } else if (actionKey === 'batch-alarm-sync') {
            const sns = selectedDevices.map((d) => d.sn);
            for (const sn of sns) {
              triggerAlarmSync.mutate(sn, {
                onSuccess: () => {
                  const timestamp = new Date().toISOString();
                  const taskId = newTasks.find((t) => t.sn === sn)?.id;
                  if (taskId) {
                    setCollectTasks((prev) => prev.map((item) =>
                      item.id === taskId ? {
                        ...item,
                        status: 'success',
                        progress: 100,
                        message: t('task.status.completed'),
                        logContent: `[${timestamp}] INFO: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${sn}\n[${timestamp}] INFO: alarm sync triggered\n[${timestamp}] INFO: ${t('task.log.success')}`,
                      } : item
                    ));
                  }
                },
                onError: () => {
                  const timestamp = new Date().toISOString();
                  const taskId = newTasks.find((t) => t.sn === sn)?.id;
                  if (taskId) {
                    setCollectTasks((prev) => prev.map((item) =>
                      item.id === taskId ? {
                        ...item,
                        status: 'failed',
                        progress: 100,
                        message: t('common.failed'),
                        logContent: `[${timestamp}] ERROR: ${t('task.log.start')}\n[${timestamp}] INFO: ${t('task.log.connect')} ${sn}\n[${timestamp}] ERROR: ${t('task.log.failed')}`,
                      } : item
                    ));
                  }
                },
              });
            }
            void message.success(t('common.commandSent'));
          } else {
            void message.success(t('common.commandSent'));
          }
          setSelectedRowKeys([]);
        },
      });
    },
    [modal, message, t, batchReboot, triggerAlarmSync, createUfteTask, navigate, devices]
  );

  // 导出实现见 columns 定义之后的 handleExport(依赖 columns,需在其后声明)。

  /** 解析多小区逗号分隔值为 cell 数组 */
  const parseCellValues = useCallback((v: string | undefined | null): string[] => {
    if (!v || v === '--') return [];
    return String(v).split(',').map((s) => s.trim()).filter(Boolean);
  }, []);

  // 格式化时间戳
  const fmtTime = useCallback((v: string) => (v ? formatSystemTime(v) : '-'), []);

  // 格式化在线时长(秒)
  const fmtDuration = useCallback((seconds: number | null | undefined) => {
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

  // 激活状态 = 设备是否曾首次上线（后端 op_state = DeriveOpStateActivated(first_online_time)：
  // '1'=激活 / '0'=未激活）。与"在线(实时连接 is_online)""小区状态(cell_status)"是三个不同维度——
  // 此前误用 cell_status==='normal' 当激活，导致在线设备因小区 inactive 显示未激活。
  //
  // ™ 判定口径由 frontend-core/utils/activationStatus.ts 统一控管——
  // 三皮肤(webcode / webcode-v2 / webcode-v3) + 列表/详情头/v2 KV/v3 KV 全调同一函数，
  // 在上层各自渲染 Tag/文本。修改判定请只改 utility。
  const renderActivationStatus = useCallback((opState: string | undefined | null) => {
    const status = activationStatusOf(opState);
    if (status == null) return '-';
    const isActive = status === 'active';
    return <Tag color={isActive ? 'success' : 'error'}>{isActive ? t('status.active') : t('status.inactive')}</Tag>;
  }, [t]);

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
            onMouseEnter={() => prefetchDeviceDetailEntry(record)}
            onFocus={() => prefetchDeviceDetailEntry(record)}
            onClick={() => openDeviceDetail(record)}
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
        render: (_val, record) => {
          const mappedStatus = mapConnStatus(record.connStatus);
          return (
            <StatusIndicator
              status={mappedStatus}
              text={mappedStatus === 'online' ? t('status.online') : t('status.offline')}
            />
          );
        },
      },
      {
        key: 'alarmLevel',
        title: t('device.alarmLevel'),
        dataIndex: 'alarmLevel',
        width: 100,
        // 2026-06-04 用户决策:前三列(SN/连接状态/告警级别)固定左侧,横向滚动时
        // 保持不透明(固定列背景由 DataTable.module.css 的 .ant-table-cell-fix-left 统一处理)。
        fixed: 'left',
        group: 'common',
        render: (_val, record) => {
          const color = SEVERITY_COLOR[record.alarmLevel] ?? 'default';
          const label = getSeverityLabel(record.alarmLevel);
          if (record.alarmLevel && record.alarmLevel !== 'none') {
            // #361: 告警级别 Tag 旁拼接活动告警数（如「重要 · 3」）。
            const count = record.activeAlarmCount ?? 0;
            const display = count > 0 ? `${label} · ${count}` : label;
            // 点击告警跳转到设备详情告警 tab
            return (
              <Tag
                color={color}
                style={{ cursor: 'pointer' }}
                onMouseEnter={() => prefetchDeviceDetailEntry(record)}
                onClick={() => openDeviceDetail(record, 'alarm')}
              >
                {display}
              </Tag>
            );
          }
          return <Tag color={color}>{label}</Tag>;
        },
      },
      // "名称" 列绑定 device_name（设备名称），而非 host_name。
      { key: 'hostName', title: t('device.hostName'), dataIndex: 'deviceName', width: 150, ellipsis: true, group: 'common' },
      {
        key: 'networkType',
        title: t('device.radioMode'),
        dataIndex: 'networkType',
        width: 100,
        group: 'common',
        render: (_val, record) => {
          const colorMap: Record<string, string> = { eNB: 'blue', gNB: 'green', GSM: 'orange' };
          // issue #223: 列文案与「基站制式」筛选下拉同源——走 network_type 字典
          // value→label 映射（与回收站统一），不再直接显示原始 eNB/gNB。
          const label = resolveNetworkTypeLabel(record.networkType, networkTypeDict?.sysDictionaryDetails);
          return <Tag color={colorMap[record.networkType] ?? 'default'}>{label}</Tag>;
        },
      },
      // 产品名称（= device.model_name，inform 命中产品后回填 product.Name）显示在产品类型前面。
      { key: 'deviceModel', title: t('device.productName'), dataIndex: 'deviceModel', width: 120, ellipsis: true, group: 'common' },
      {
        key: 'productClass',
        title: t('device.productClass'),
        dataIndex: 'productClass',
        width: 120,
        group: 'common',
        // 原始 JSP: product 字段 — PM-B4860/QAFA/BaiBNX/BSC/BTS 等
        render: (_val, record) => record.productClass || '-',
      },
      { key: 'softwareVersion', title: t('device.softwareVersion'), dataIndex: 'softwareVersion', width: 140, ellipsis: true, group: 'common' },
      {
        key: 'ipAddress',
        title: t('device.ipAddress'),
        dataIndex: 'ipAddress',
        width: 140,
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
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true, copyable: true, group: 'common' },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 120, group: 'common' },
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
        group: 'common',
        // 激活状态 = 设备是否曾首次上线（op_state），与在线/小区状态正交。
        render: (_val, record) => renderActivationStatus(record.opState),
      },
      {
        key: 'offlineDuration',
        title: t('device.offlineDuration'),
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => {
          // 仅离线设备显示
          if (record.connStatus !== 'offline') return '-';
          return formatOfflineDuration(
            t,
            record.offlineDays,
            record.offlineHours,
            record.offlineMinutes
          );
        },
      },
      {
        key: 'ueCount',
        title: t('device.ueCount'),
        dataIndex: 'ueCount',
        width: 80,
        group: 'common',
        // JSP 行为: eNB >0 可点击(跳转 UE 详情页)；gNB/GSM 不可点击
        render: (_val, record) => {
          const v = record.ueCount;
          if (v === -1 || v === null || v === undefined) return '--';
          if (v === 0) return '0';
          // 仅 eNB 支持点击跳转 UE 详情页
          const isEnb = record.networkType === 'eNB';
          if (isEnb) {
            return (
              <Link onClick={() => navigate(`/device/ue-detail/${record.sn}?name=${encodeURIComponent(record.name || record.sn)}&ueCount=${record.ueCount}`)}>
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
      {
        key: 'onlineDuration',
        title: t('device.onlineDuration'),
        dataIndex: 'onlineDuration',
        width: 120,
        hidden: true,
        group: 'common',
        render: (_val, record) => fmtDuration(record.onlineDuration),
      },
      { key: 'upTime', title: t('device.upTime'), dataIndex: 'upTime', width: 120, hidden: true, group: 'common', render: (_val, record) => fmtDuration(record.upTime) },
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
      {
        key: 'siteName',
        title: t('device.cellName'),
        dataIndex: 'cellId',
        width: 130,
        hidden: true,
        ellipsis: true,
        group: 'common',
        // cellId 为空/null/空串 → 显占位符 '--'(绝不回退显 SN/设备编码,#184)。
        render: (val) => {
          const v = val as string | null | undefined;
          return v == null || v === '' ? '--' : v;
        },
      },
      // Remark 列暂时隐藏（用户反馈：含义不明 + 表头自定义编辑能力暂未对接后端持久化）。
      // 恢复方式：取消下行注释并保留 remarkHeaderRender / handleRemarkLabel* state（已保留）。
      // { key: 'remark', title: t('device.remark'), dataIndex: 'remark', width: 185, hidden: true, ellipsis: true, group: 'common', headerRender: remarkHeaderRender },
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
                title={`${t('device.longitude')}: ${record.longitude ?? '--'}   ${t('device.latitude')}: ${record.latitude ?? '--'}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
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
                title={`${t('device.longitude')}: ${record.longitude ?? '--'}   ${t('device.latitude')}: ${record.latitude ?? '--'}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
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
                title={`${t('device.longitude')}: ${record.longitude ?? '--'}   ${t('device.latitude')}: ${record.latitude ?? '--'}   ${t('device.gpsHeight')}(m): ${record.gpsHeight ?? '--'}`}
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
          return v > 0
            ? (
              <Link
                onMouseEnter={() => prefetchDeviceDetailEntry(record)}
                onFocus={() => prefetchDeviceDetailEntry(record)}
                onClick={() => openDeviceDetail(record, 'gps')}
              >
                {v}
              </Link>
            )
            : String(v);
        },
      },
      { key: 'installAddress', title: t('device.installAddress'), dataIndex: 'installAddress', width: 180, hidden: true, ellipsis: true, group: 'common' },
      { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 80, hidden: true, group: 'common' },
      { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 80, hidden: true, group: 'common' },
      { key: 'band', title: 'Band', dataIndex: 'band', width: 100, hidden: true, group: 'common' },
      { key: 'dlEarfcn', title: t('device.dlEarfcn'), dataIndex: 'dlEarfcn', width: 110, hidden: true, group: 'common' },
      { key: 'ulEarfcn', title: t('device.ulEarfcn'), dataIndex: 'ulEarfcn', width: 110, hidden: true, group: 'common' },
      // 基站类型(networkModel)暂时隐藏：当前 LTE/NR/双模 推导口径未与产品对齐;恢复时取消下行注释。
      // { key: 'networkModel', title: t('device.networkModel'), dataIndex: 'networkModel', width: 110, hidden: true, group: 'common' },
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
      {
        key: 'latestLog',
        title: t('device.latestLog'),
        dataIndex: 'id',
        width: 100,
        hidden: true,
        group: 'common',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            loading={downloadStationLog.isPending}
            onClick={async () => {
              const res = await stationLogApi.list({ deviceId: record.id, logType: 'running', page: 1, pageSize: 1 });
              const log = res.items[0];
              if (!log) {
                void message.info(t('device.noLogFile'));
                return;
              }
              downloadStationLog.mutate(log.id);
            }}
          >
            {t('common.download')}
          </Button>
        ),
      },

    ],
    // remarkHeaderRender 暂从 dep 列表移除：remark 列定义已注释，恢复时同步加回。
    [navigate, t, fmtTime, fmtDuration, fmtStatus, renderMultiCellStatus, renderActivationStatus, message, downloadStationLog, mapConnStatus, getSeverityLabel, networkTypeDict?.sysDictionaryDetails]
  );

  // ─── 列表导出(用户决策 2026-06-02) ──────────────────────────────────────
  // 1) 仅导出当前筛选条件命中的全部数据(跨分页拉全量,受 EXPORT_ROW_CAP 上限保护);
  // 2) 导出列与列顺序严格以"列设置"(ColumnVisibility)为准 —— 即与表格当前展示一致。

  // 单元格取值：与列 render 的可读文案对齐;特殊列(状态/告警/时间/时长/枚举)单独
  // 格式化,其余列回退到 record[dataIndex] 原值(数组以逗号拼接,空值 → 空串)。
  const formatExportCell = useCallback(
    (record: Device, key: string, dataIndex?: string): string => {
      switch (key) {
        case 'connStatus':
          return mapConnStatus(record.connStatus) === 'online' ? t('status.online') : t('status.offline');
        case 'alarmLevel':
          return getSeverityLabel(record.alarmLevel);
        case 'onlineTime':
          return fmtTime(record.onlineTime);
        case 'offlineTime':
          return fmtTime(record.offlineTime);
        case 'firstOnlineTime':
          return fmtTime(record.firstOnlineTime);
        case 'lastInformTime':
          return fmtTime(record.lastInformTime);
        case 'lastOnlineTime':
          return fmtTime(record.lastOnlineTime);
        case 'onlineDuration':
          return fmtDuration(record.onlineDuration);
        case 'offlineDuration':
          return record.connStatus === 'offline'
            ? offlineDurationText(t, record.offlineDays, record.offlineHours, record.offlineMinutes)
            : '-';
        case 'halobFlag':
          return record.halobFlag == null
            ? '-'
            : record.halobFlag
              ? t('status.enabled')
              : t('status.disabled');
        case 'adminState': {
          const m: Record<string, string> = { '1': 'Locked', '2': 'Unlocked', '3': 'ShuttingDown' };
          return record.adminState != null ? (m[String(record.adminState)] ?? String(record.adminState)) : '-';
        }
        case 'ueCount': {
          const v = record.ueCount;
          return v === -1 || v == null ? '--' : String(v);
        }
        case 'opState': {
          // 激活状态 = 曾上线(op_state '1'/'0')，映射为"激活/未激活"文本（与列表列同口径）。
          const os = record.opState;
          if (os == null || os === '' || os === 'unknown') return '-';
          return os === '1' ? t('status.active') : t('status.inactive');
        }
        default: {
          const v = dataIndex ? (record as unknown as Record<string, unknown>)[dataIndex] : undefined;
          if (Array.isArray(v)) return v.join(', ');
          return v == null ? '' : String(v);
        }
      }
    },
    [mapConnStatus, getSeverityLabel, fmtTime, fmtDuration, t]
  );

  // 按当前筛选条件并发分页拉取全部命中数据(不受列表当前页/页大小限制)。
  // 全表无筛选时可达 3W+ 行,串行逐页会慢到像"点了没反应",故用并发池 + 进度回调。
  const fetchAllFilteredDevices = useCallback(
    (onProgress?: (loaded: number, total: number) => void) =>
      fetchAllPaged<Device>(
        async (page, pageSize) => {
          const resp = await exportDeviceApi.getList({
            ...filterParams,
            page,
            pageSize,
          } as Parameters<typeof useDeviceList>[0]);
          return { items: resp.items, total: resp.total ?? resp.items.length };
        },
        { onProgress },
      ),
    [filterParams]
  );

  // 导出 — 选择格式(xlsx/csv)后触发：筛选生效 + 列以"列设置"为准。
  const handleExport = useCallback(
    async (format: 'xlsx' | 'csv') => {
      setExportModalOpen(false);
      const exportCols = resolveVisibleExportColumns(columns, DEVICE_LIST_TABLE_ID);
      if (exportCols.length === 0) {
        void message.warning(t('common.noColumnsToExport'));
        return;
      }
      const msgKey = 'device-list-export';
      message.open({ key: msgKey, type: 'loading', content: t('common.exportInProgress'), duration: 0 });
      try {
        // 进度更新做节流(全表 34 页会在 ~1s 内回调 34 次):同一 key 高频 message.open
        // 在部分机器上会出现"内容还没渲染就被下一次替换"的空白闪烁。限制为最多每 300ms
        // 刷新一次,并始终渲染最后一次(loaded===total),既给反馈又不抖动。
        let lastProgressAt = 0;
        const { items: rowsData, capped } = await fetchAllFilteredDevices((loaded, total) => {
          const now = Date.now();
          if (loaded < total && now - lastProgressAt < 300) return;
          lastProgressAt = now;
          const content = t('common.exportProgress', { loaded, total }) || `${loaded}/${total}`;
          message.open({ key: msgKey, type: 'loading', content, duration: 0 });
        });
        if (rowsData.length === 0) {
          message.open({ key: msgKey, type: 'warning', content: t('common.noDataToExport') });
          return;
        }
        const headers = exportCols.map((c) => c.title);
        const matrix = rowsData.map((dev) => exportCols.map((c) => formatExportCell(dev, c.key, c.dataIndex)));
        const filename = `device-list-${exportTimestamp()}.${format}`;
        if (format === 'csv') {
          triggerCsvDownload(buildCsvContent(headers, matrix), filename);
        } else {
          triggerXlsxDownload(headers, matrix, filename, 'devices');
        }
        // 命中上限时显式告知用户被截断,避免"以为导全了实际只导了一部分"。
        message.open({
          key: msgKey,
          type: capped ? 'warning' : 'success',
          content: capped
            ? t('common.exportCapped', { count: rowsData.length })
            : t('common.exportSuccess', { count: rowsData.length }),
        });
      } catch (e) {
        message.open({
          key: msgKey,
          type: 'error',
          content: t('common.exportFailed', { reason: e instanceof Error ? e.message : String(e) }),
        });
      }
    },
    [columns, message, t, fetchAllFilteredDevices, formatExportCell]
  );

  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'batch-reboot',
      label: t('common.batchReboot'),
      icon: <ReloadOutlined />,
      onClick: (keys) => handleBatchAction(t('common.batchReboot'), keys, 'batch-reboot'),
    },
    // batch-tr069-collect 批量按钮已移除（#179）：按需抓包统一以「运维管理 - TR069
    // 报文跟踪」菜单为唯一入口；设备页该入口冗余且对 NAT 后设备主动呼叫超时易误触失败。
    {
      key: 'batch-log-collect',
      label: t('device.action.logCollect'),
      icon: <FileTextOutlined />,
      onClick: (keys) => handleBatchAction(t('device.action.logCollect'), keys, 'batch-log-collect'),
    },
    {
      key: 'batch-alarm-sync',
      label: t('device.action.alarmSync'),
      icon: <AlertOutlined />,
      onClick: (keys) => handleBatchAction(t('device.action.alarmSync'), keys, 'batch-alarm-sync'),
    },
    // 恢复默认配置已隐藏
    // {
    //   key: 'batch-reset-config',
    //   label: t('device.action.resetConfig'),
    //   icon: <ExclamationCircleOutlined />,
    //   danger: true,
    //   onClick: (keys) => handleBatchAction(t('device.action.resetConfig'), keys, 'batch-reset-config'),
    // },
  ], [handleBatchAction, t]);

  // 任务面板表格列定义
  const taskColumns: ColumnsType<LocalTask> = useMemo(() => [
    {
      title: t('table.action'),
      key: 'action',
      width: 70,
      fixed: 'left',
      render: (_: unknown, record: LocalTask) => {
        // 只有收集操作（hasDetail=true）且完成或失败状态才显示查看按钮
        if (!record.hasDetail) return null;
        if (record.status !== 'success' && record.status !== 'failed') return null;
        return (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewLog(record)}
            style={{ padding: 0, fontSize: 12 }}
          >
            {t('common.view')}
          </Button>
        );
      },
    },
    {
      title: 'SN',
      dataIndex: 'sn',
      key: 'sn',
      width: 140,
      ellipsis: true,
      render: (sn: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 11 }}>{sn}</span>
      ),
    },
    {
      title: t('alarm.deviceName'),
      dataIndex: 'deviceName',
      key: 'deviceName',
      ellipsis: true,
      width: 120,
    },
    {
      title: t('table.type'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      ellipsis: true,
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: TaskStatus) => {
        const statusConfig: Record<TaskStatus, { color: string; text: string }> = {
          pending: { color: 'default', text: t('status.pending') },
          running: { color: 'processing', text: t('task.status.running') },
          success: { color: 'success', text: t('task.status.completed') },
          failed: { color: 'error', text: t('task.status.failed') },
        };
        const cfg = statusConfig[status];
        return (
          <Tag color={cfg.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
            {cfg.text}
          </Tag>
        );
      },
    },
    {
      title: t('task.progress'),
      dataIndex: 'progress',
      key: 'progress',
      width: 120,
      render: (progress: number, record) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <Progress
            percent={Math.round(progress)}
            size="small"
            status={record.status === 'failed' ? 'exception' : record.status === 'success' ? 'success' : 'active'}
            showInfo={false}
            style={{ flex: 1, minWidth: 60 }}
          />
          <span style={{ fontSize: 11, color: 'var(--color-neutral-600)', whiteSpace: 'nowrap' }}>
            {Math.round(progress)}%
          </span>
        </div>
      ),
    },
  ], [t, handleViewLog]);

  // R3: 导出按钮挪到 FilterBar 搜索按钮右侧（device-list-and-group-improvements-20260520.md R3）。
  // ListPageLayout.extra 不再承载，让筛选区与导出动作在视觉上一行对齐。
  const exportButton = (
    <Button
      type="primary"
      icon={<ExportOutlined />}
      onClick={() => setExportModalOpen(true)}
    >
      {t('common.export')}
    </Button>
  );

  // 导出确认弹窗：用 antd Modal（主题感知，不会再白底白字）。固定 CSV 格式，
  // 用户决策 2026-06-02：导出只支持 CSV，不再提供格式选择。
  const exportConfirmModal = (
    <Modal
      open={exportModalOpen}
      title={t('common.exportConfirmTitle')}
      onCancel={() => setExportModalOpen(false)}
      onOk={() => handleExport('csv')}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
    >
      {t('common.exportConfirmContent')}
    </Modal>
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      <div style={{ flex: '1 1 100%', minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <ListPageLayout>
          <FilterBar
            filterId="device-list"
            fields={visibleFilterFields}
            onSearch={handleSearch}
            onReset={handleReset}
            collapsedRows={1}
            initialValues={filterParams}
            extra={exportButton}
          />

          <StatisticsPanel items={statsItems} style={{ marginBottom: 8 }} />

          {/* 设备列表卡片。
              注意:不要在外层套 <Spin>。Antd Spin 内部的
              .ant-spin-nested-loading + .ant-spin-container 是 display:block
              不是 flex,会断 ListPageLayout → Card 的 flex 高度链,导致 Card
              收缩到内容自然高度(~161px),DataTable.autoFitHeight 算出 y=1px,
              所有列表行被压到 1px 高度内不可见(分页栏看似贴上来盖住列表)。
              DataTable 自己已接 loading={isLoading},无需外层 Spin。 */}
          <Card
            size="small"
            variant="outlined"
            style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
            styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
          >
            <DataTable<Device>
              tableId="device-list-table"
              columns={columns}
              onHiddenColumnsChange={setHiddenColumnKeys}
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
              defaultDensity="default"
              scroll={{ x: 'max-content' }}
              autoFitHeight
              onRealtimeRefreshChange={setAutoRefresh}
              showRowNumber
              rowNumberTitle={t('table.rowNumber')}
            />
          </Card>
        </ListPageLayout>
      </div>

      {/* 收集任务抽屉 */}
      <Drawer
        title={collectDrawerTitle || t('task.collectProgress')}
        placement="right"
        size={480}
        open={collectDrawerOpen}
        onClose={() => setCollectDrawerOpen(false)}
        styles={{
          body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
        }}
      >
        {/* 任务统计 */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--color-border)', flexShrink: 0 }}>
          <Space size={16}>
            <span>
              {t('task.total')}: {collectTasks.length}
            </span>
            <span style={{ color: 'var(--color-primary-600)' }}>
              {t('task.status.running')}: {collectTasks.filter((item) => item.status === 'running').length}
            </span>
            <span style={{ color: '#52c41a' }}>
              {t('task.status.completed')}: {collectTasks.filter((item) => item.status === 'success').length}
            </span>
            <span style={{ color: '#ff4d4f' }}>
              {t('task.status.failed')}: {collectTasks.filter((item) => item.status === 'failed').length}
            </span>
          </Space>
        </div>

        {/* 任务列表 */}
        <div style={{ flex: 1, overflow: 'hidden', padding: 8 }}>
          <Table<LocalTask>
            dataSource={collectTasks}
            columns={taskColumns}
            rowKey="id"
            size="small"
            pagination={false}
            scroll={{ x: 630, y: 'calc(100vh - 180px)' }}
            locale={{ emptyText: t('common.noData') }}
            style={{ fontSize: 12 }}
          />
        </div>
      </Drawer>

      {/* 日志详情弹窗 */}
      <Modal
        title={t('task.logDetail')}
        open={logModalOpen}
        onCancel={() => setLogModalOpen(false)}
        footer={[
          <Button key="close" onClick={() => setLogModalOpen(false)}>
            {t('common.close')}
          </Button>,
        ]}
        width={640}
      >
        {currentLogTask && (
          <div>
            <div style={{ marginBottom: 12, display: 'flex', gap: 16, fontSize: 13 }}>
              <span><strong>SN:</strong> <code style={{ fontFamily: 'monospace' }}>{currentLogTask.sn}</code></span>
              <span><strong>{t('alarm.deviceName')}:</strong> {currentLogTask.deviceName}</span>
              <span>
                <strong>{t('table.status')}:</strong>{' '}
                <Tag color={currentLogTask.status === 'success' ? 'success' : 'error'} style={{ marginLeft: 4 }}>
                  {currentLogTask.status === 'success' ? t('task.status.completed') : t('task.status.failed')}
                </Tag>
              </span>
            </div>
            <div
              style={{
                backgroundColor: '#1e1e1e',
                color: '#d4d4d4',
                padding: 16,
                borderRadius: 6,
                fontFamily: 'monospace',
                fontSize: 12,
                lineHeight: 1.6,
                maxHeight: 400,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
              }}
            >
              {currentLogTask.logContent || t('common.noData')}
            </div>
          </div>
        )}
      </Modal>

      {exportConfirmModal}
    </div>
  );
}
