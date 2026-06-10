import React, { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { App, Button, Drawer, Form, Input, Modal, Radio, Select, Space, Tag, Tree, Typography } from 'antd';
import { PlusOutlined, SearchOutlined, DownloadOutlined, DeleteOutlined, EditOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useIndicatorGroupTree,
  useIndicatorList,
  useCreateIndicator,
  useUpdateIndicator,
  useDeleteIndicator,
  useEnableIndicators,
  useDisableIndicators,
  useExportIndicators,
  useUpdateCustName,
} from '@core/hooks/api/useIndicator';
import { indicatorApi } from '@core/services/api/indicatorApi';
import { useAppStore } from '@core/store/appStore';
import { useUserStore } from '@core/store/userStore';
import type { IndicatorGroup, PerfIndicator } from '@core/types/indicator';

const { Link } = Typography;

// KPI 指标行数据接口（适配后端 PerfIndicator）
interface KPIIndicatorRow extends Record<string, unknown> {
  kpiId: string;
  kpiName: string;
  productClass: string;
  custName: string;
  indicatorLevel: string;
  unit: string;
  isCustomize: boolean;
  isEnable: boolean;
  indicatorType: string;
  catagoryId: string;
  catagoryName: string;
  updater: string;
  updateTime: string;
  arithmetic: string;
  definition: string;
  statisType: string;
  calculatingStatus: string;
}

function getLocalizedIndicatorName(ind: PerfIndicator, locale: string): string {
  if (locale === 'en-US') return ind.kpiNameEn || ind.kpiName || ind.kpiNameZh || '-';
  return ind.kpiNameZh || ind.kpiName || ind.kpiNameEn || '-';
}

function getLocalizedDefinition(ind: PerfIndicator, locale: string): string {
  if (locale === 'en-US') return ind.definitionEn || ind.definition || ind.definitionZh || '';
  return ind.definitionZh || ind.definition || ind.definitionEn || '';
}

function getLocalizedGroupName(group: IndicatorGroup, locale: string): string {
  if (locale === 'en-US') return group.enName || group.cnName || group.id;
  return group.cnName || group.enName || group.id;
}

// 将 PerfIndicator 映射为行数据
function toRow(ind: PerfIndicator, locale: string): KPIIndicatorRow {
  return {
    kpiId: ind.kpiId,
    kpiName: getLocalizedIndicatorName(ind, locale),
    productClass: ind.productClass,
    custName: ind.custName,
    indicatorLevel: ind.indicatorLevel,
    unit: ind.unit,
    isCustomize: ind.isCustomize,
    isEnable: ind.isEnable,
    indicatorType: ind.indicatorType,
    catagoryId: ind.catagoryId,
    catagoryName: ind.catagoryName,
    updater: ind.updater,
    updateTime: ind.updateTime,
    arithmetic: ind.arithmetic,
    definition: getLocalizedDefinition(ind, locale),
    statisType: ind.statisType,
    calculatingStatus: ind.calculatingStatus,
  };
}

function formatDateTime(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  if (isNaN(d.getTime())) return value;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

// 设备类型标签页映射
type DeviceTypeTab = 'ENB' | 'GSM' | 'GNB';
const DEVICE_TABS: { key: DeviceTypeTab; labelKey: string }[] = [
  { key: 'ENB', labelKey: 'kpi.tree.enbSet' },
  { key: 'GSM', labelKey: 'kpi.tree.gsmSet' },
  { key: 'GNB', labelKey: 'kpi.tree.gnbSet' },
];

// 树节点带悬停图标的渲染
interface TreeNodeTitleProps {
  title: string;
  nodeKey: string;
  isCustom?: boolean;
  onAdd?: (key: string) => void;
  onEdit?: (key: string, title: string) => void;
  onDelete?: (key: string) => void;
}

function TreeNodeTitle({ title, nodeKey, isCustom, onAdd, onEdit, onDelete }: TreeNodeTitleProps) {
  const [hovered, setHovered] = useState(false);

  return (
    <div
      style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%', minHeight: 24 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{title}</span>
      {onAdd && (
        <Space size={0} style={{ visibility: hovered ? 'visible' : 'hidden', flexShrink: 0 }}>
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={(e) => {
              e.stopPropagation();
              onAdd(nodeKey);
            }}
            style={{ fontSize: 12, padding: '0 4px' }}
          />
          {isCustom && onEdit && (
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                onEdit(nodeKey, title);
              }}
              style={{ fontSize: 12, padding: '0 4px' }}
            />
          )}
          {isCustom && onDelete && (
            <Button
              type="text"
              size="small"
              icon={<DeleteOutlined />}
              danger
              onClick={(e) => {
                e.stopPropagation();
                onDelete(nodeKey);
              }}
              style={{ fontSize: 12, padding: '0 4px' }}
            />
          )}
        </Space>
      )}
    </div>
  );
}

/**
 * 将后端 IndicatorGroup 树转换为 antd DataNode 树
 */
function useGroupTreeData(
  t: (id: string) => string,
  groups: IndicatorGroup[],
  _customNodes: { key: string; parentKey: string; title: string }[],
  onAddNode: (parentKey: string) => void,
  onEditNode: (nodeKey: string, title: string) => void,
  onDeleteNode: (nodeKey: string) => void,
  searchValue: string,
  locale: string,
  deviceType: DeviceTypeTab,
): DataNode[] {
  return useMemo(() => {
    const lowerSearch = searchValue.toLowerCase();

    const nodeMatchesSearch = (label: string): boolean => {
      if (!searchValue) return true;
      return label.toLowerCase().includes(lowerSearch);
    };

    const convertGroup = (group: IndicatorGroup): DataNode | null => {
      const isRoot = group.parentId === group.id;
      const rootLabelKey =
        deviceType === 'ENB' ? 'kpi.tree.enbSet'
          : deviceType === 'GSM' ? 'kpi.tree.gsmSet'
            : 'kpi.tree.gnbSet';
      const label = isRoot ? t(rootLabelKey) : getLocalizedGroupName(group, locale);
      const isCustom = !group.isBuildIn;
      const childNodes = (group.children || [])
        .map(convertGroup)
        .filter(Boolean) as DataNode[];

      const selfMatches = nodeMatchesSearch(label);
      const hasMatchingChildren = childNodes.length > 0;

      if (searchValue && !selfMatches && !hasMatchingChildren) {
        return null;
      }

      return {
        title: (
          <TreeNodeTitle
            title={label}
            nodeKey={group.id}
            isCustom={isCustom}
            onAdd={onAddNode}
            onEdit={isCustom ? onEditNode : undefined}
            onDelete={isCustom ? onDeleteNode : undefined}
          />
        ),
        key: group.id,
        children: childNodes.length > 0 ? childNodes : undefined,
      };
    };

    return groups.map(convertGroup).filter(Boolean) as DataNode[];
  }, [groups, onAddNode, onEditNode, onDeleteNode, searchValue, locale, t, deviceType]);
}

export default function KPIStandardReport() {
  const t = useT();
  const navigate = useNavigate();
  const locale = useAppStore((s) => s.locale);
  const currentUser = useUserStore((s) => s.currentUser);
  const { message, modal } = App.useApp();
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [treeSearchValue, setTreeSearchValue] = useState('');
  const [tableSearchValue, setTableSearchValue] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 筛选状态
  const [productClass, setProductClass] = useState<string>('');
  const [indicatorLevel, setIndicatorLevel] = useState<string>('');
  const [indicatorTypeFilter, setIndicatorTypeFilter] = useState<string>('');
  const [isEnableFilter, setIsEnableFilter] = useState<string>('');

  // 设备类型标签页
  const [deviceType, setDeviceType] = useState<DeviceTypeTab>('ENB');

  // 已选指标
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // Custom tree nodes state
  const [customNodes] = useState<{ key: string; parentKey: string; title: string }[]>([]);

  const levelOptions = useMemo(() => [
    { label: t('kpi.all'), value: '' },
    { label: 'Device', value: 'device' },
    { label: 'PLMN', value: 'plmn' },
  ], [t]);

  // ── Mutations ──────────────────────────────────────────────────────────────
  const createIndicator = useCreateIndicator();
  const updateIndicator = useUpdateIndicator();
  const deleteIndicator = useDeleteIndicator();
  const enableIndicators = useEnableIndicators();
  const disableIndicators = useDisableIndicators();
  const updateCustName = useUpdateCustName();
  const exportIndicators = useExportIndicators();

  // ── Queries: 功能集树 ───────────────────────────────────────────────────────
  const { data: groupTreeData = [], isLoading: groupTreeLoading, refetch: refetchGroupTree } = useIndicatorGroupTree({
    deviceType,
  });

  // ── Queries: 指标列表 ───────────────────────────────────────────────────────
  // 树节点 key 直接使用 API 返回的 group ID，可直接传给后端
  const selectedGroupId = useMemo(() => {
    // 根节点（自引用 parent_id==id）不作为过滤条件
    if (!selectedCategory) return undefined;
    const rootGroup = groupTreeData.find(g => g.id === selectedCategory && g.parentId === g.id);
    if (rootGroup) return undefined;
    return selectedCategory;
  }, [selectedCategory, groupTreeData]);

  const indicatorListParams = useMemo(() => ({
    deviceType,
    catagoryId: selectedGroupId,
    searchText: tableSearchValue || undefined,
    productClass: productClass || undefined,
    indicatorType: indicatorTypeFilter || undefined,
    isEnable: isEnableFilter || undefined,
    indicatorLevel: indicatorLevel || undefined,
    page,
    rows: pageSize,
  }), [deviceType, selectedGroupId, tableSearchValue, productClass, indicatorTypeFilter, isEnableFilter, indicatorLevel, page, pageSize]);

  const { data: indicatorPage, isLoading: indicatorLoading, refetch: refetchIndicatorList } = useIndicatorList(indicatorListParams);

  // 将后端数据转换为行数据
  const tableData: KPIIndicatorRow[] = useMemo(() => {
    return (indicatorPage?.items || []).map((indicator) => toRow(indicator, locale));
  }, [indicatorPage, locale]);

  const totalCount = indicatorPage?.total ?? 0;

  // ── 编辑指标详情查询 ────────────────────────────────────────────────────────

  // Add indicator drawer state
  const [addDrawerOpen, setAddDrawerOpen] = useState(false);
  const [addIndicatorForm] = Form.useForm<{
    indicatorType: 'kpi' | 'counter';
    indicatorLevel: string;
    kpiName: string;
    productClass: string;
    custName: string;
    catagoryId: string;
    unit: string;
    statisType: string;
    isEnable: string;
    definition: string;
  }>();
  const [currentIndicatorType, setCurrentIndicatorType] = useState<'kpi' | 'counter'>('kpi');
  const [calcFormula, setCalcFormula] = useState<string>('');
  const [formulaSearchValue, setFormulaSearchValue] = useState<string>('');
  const [formulaSelectedCategory, setFormulaSelectedCategory] = useState<string>('');
  const [formulaProductClass, setFormulaProductClass] = useState<string>('');

  // Edit indicator drawer state
  const [editDrawerOpen, setEditDrawerOpen] = useState(false);
  const [editIndicatorForm] = Form.useForm<{
    indicatorType: 'kpi' | 'counter';
    indicatorLevel: string;
    kpiName: string;
    productClass: string;
    custName: string;
    catagoryId: string;
    unit: string;
    statisType: string;
    isEnable: string;
    definition: string;
  }>();
  const [editIndicatorType, setEditIndicatorType] = useState<'kpi' | 'counter'>('kpi');
  const [editCalcFormula, setEditCalcFormula] = useState<string>('');
  const [editingIndicator, setEditingIndicator] = useState<KPIIndicatorRow | null>(null);

  // Add function set modal state
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [addForm] = Form.useForm<{ name: string; description: string }>();

  // Add node modal state
  const [addNodeModalOpen, setAddNodeModalOpen] = useState(false);
  const [addNodeForm] = Form.useForm<{ name: string }>();
  const [currentParentKey, setCurrentParentKey] = useState<string>('');

  // Edit node modal state
  const [editNodeModalOpen, setEditNodeModalOpen] = useState(false);
  const [editNodeForm] = Form.useForm<{ name: string; description: string }>();
  const [editingNodeKey, setEditingNodeKey] = useState<string>('');

  // 获取树节点名称的辅助函数
  const getTreeNodeName = useCallback((nodeKey: string): string => {
    // 从 API 树数据中查找
    const findName = (groups: IndicatorGroup[]): string => {
      for (const g of groups) {
        if (g.id === nodeKey) return getLocalizedGroupName(g, locale);
        if (g.children) {
          const found = findName(g.children);
          if (found) return found;
        }
      }
      return '';
    };
    const name = findName(groupTreeData);
    return name || nodeKey;
  }, [groupTreeData, locale]);

  // 当前选中的功能集名称
  const selectedFunctionSetName = useMemo(() => {
    if (!selectedCategory) return t('kpi.allIndicators');
    return getTreeNodeName(selectedCategory);
  }, [selectedCategory, getTreeNodeName, t]);

  // Handle add node to tree - 打开新建指标抽屉
  const handleAddNode = useCallback((parentKey: string) => {
    setCurrentParentKey(parentKey);
    setAddDrawerOpen(true);
    addIndicatorForm.resetFields();
    setCurrentIndicatorType('kpi');
  }, [addIndicatorForm]);

  // Handle delete node from tree
  const handleDeleteNode = useCallback((nodeKey: string) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirmDelete'),
      onOk: async () => {
        try {
          await indicatorApi.deleteIndicatorGroup(nodeKey, deviceType);
          void message.success(t('common.success'));
          await refetchGroupTree();
        } catch (err) {
          void message.error(t('common.deleteFailed'));
          console.error('Delete group failed:', err);
        }
      },
    });
  }, [modal, message, t, deviceType]);

  // Handle edit node from tree
  const handleEditNode = useCallback((nodeKey: string, title: string) => {
    setEditingNodeKey(nodeKey);
    editNodeForm.setFieldsValue({
      name: title,
      description: '',
    });
    setEditNodeModalOpen(true);
  }, [editNodeForm]);

  const kpiTreeData = useGroupTreeData(t, groupTreeData, customNodes, handleAddNode, handleEditNode, handleDeleteNode, treeSearchValue, locale, deviceType);

  // 处理选中行变化
  const handleSelectChange = useCallback((newSelectedRowKeys: React.Key[]) => {
    setSelectedRowKeys(newSelectedRowKeys);
  }, []);

  // 删除自定义指标
  const handleDeleteIndicator = useCallback((row: KPIIndicatorRow) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirmDelete'),
      onOk: async () => {
        try {
          await deleteIndicator.mutateAsync({ id: row.kpiId, deviceType });
          void message.success(t('common.success'));
        } catch (err) {
          void message.error(t('common.deleteFailed'));
          console.error('Delete indicator failed:', err);
        }
      },
    });
  }, [modal, message, t, deleteIndicator, deviceType]);

  // 批量操作定义
  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'enableMeasure',
      label: t('kpi.measure'),
      icon: <CheckOutlined />,
      onClick: () => {
        if (selectedRowKeys.length === 0) {
          void message.warning(t('common.selectAtLeastOne'));
          return;
        }
        modal.confirm({
          title: t('common.confirm'),
          content: t('kpi.confirmEnableMeasure'),
          onOk: async () => {
            try {
              await enableIndicators.mutateAsync({
                deviceType,
                operatorCode: 'default',
                indicatorIds: selectedRowKeys as string[],
                enable: true,
              });
              void message.success(t('common.success'));
              setSelectedRowKeys([]);
            } catch (err) {
              void message.error(t('common.operationFailed'));
              console.error('Enable indicators failed:', err);
            }
          },
        });
      },
    },
    {
      key: 'disableMeasure',
      label: t('kpi.cancelMeasure'),
      icon: <CloseOutlined />,
      onClick: () => {
        if (selectedRowKeys.length === 0) {
          void message.warning(t('common.selectAtLeastOne'));
          return;
        }
        modal.confirm({
          title: t('common.confirm'),
          content: t('kpi.confirmDisableMeasure'),
          onOk: async () => {
            try {
              await disableIndicators.mutateAsync({
                deviceType,
                operatorCode: 'default',
                indicatorIds: selectedRowKeys as string[],
                enable: false,
              });
              void message.success(t('common.success'));
              setSelectedRowKeys([]);
            } catch (err) {
              void message.error(t('common.operationFailed'));
              console.error('Disable indicators failed:', err);
            }
          },
        });
      },
    },
  ], [modal, message, t, enableIndicators, disableIndicators, deviceType, selectedRowKeys]);

  // 编辑抽屉公式处理函数
  const handleEditAddOperator = useCallback((op: string) => {
    setEditCalcFormula((prev) => prev + ' ' + op + ' ');
  }, []);

  const handleEditAddIndicatorToFormula = useCallback((indicatorId: string) => {
    setEditCalcFormula((prev) => prev + `[${indicatorId}]`);
  }, []);

  const handleEditClearFormula = useCallback(() => {
    setEditCalcFormula('');
  }, []);

  // 编辑指标处理函数
  const handleEditIndicator = useCallback((row: KPIIndicatorRow) => {
    setEditingIndicator(row);
    const indicatorType = row.indicatorType === 'kpi' ? 'kpi' : 'counter';
    setEditIndicatorType(indicatorType);
    setEditCalcFormula(row.arithmetic || '');
    editIndicatorForm.setFieldsValue({
      indicatorLevel: row.indicatorLevel,
      kpiName: row.kpiName,
      productClass: row.productClass || '',
      custName: row.custName || '',
      unit: row.unit,
      statisType: row.statisType,
      isEnable: row.isEnable ? '1' : '0',
      definition: row.definition || '',
    });
    setEditDrawerOpen(true);
  }, [editIndicatorForm]);

  // 保存编辑指标
  const handleSaveEditIndicator = useCallback(async () => {
    try {
      const values = await editIndicatorForm.validateFields();

      if (!editingIndicator?.isCustomize) {
        // 内置指标：只允许修改自定义名称和测量开关
        const promises: Promise<unknown>[] = [];

        // 自定义名称
        const newCustName = values.custName || '';
        if (newCustName !== (editingIndicator?.custName || '')) {
          const dt = (deviceType || 'ENB') as 'ENB' | 'GSM' | 'GNB';
          promises.push(updateCustName.mutateAsync({
            deviceType: dt,
            operatorCode: 'default',
            perfId: editingIndicator?.kpiId || '',
            custName: newCustName,
          }));
        }

        // 测量开关
        const newEnabled = values.isEnable;
        const originalEnabled = editingIndicator?.isEnable ? '1' : '0';
        if (newEnabled !== undefined && newEnabled !== originalEnabled) {
          const dt = (deviceType || 'ENB') as 'ENB' | 'GSM' | 'GNB';
          if (newEnabled === '1') {
            promises.push(enableIndicators.mutateAsync({
              deviceType: dt,
              operatorCode: 'default',
              indicatorIds: [editingIndicator?.kpiId || ''],
              enable: true,
            }));
          } else {
            promises.push(disableIndicators.mutateAsync({
              deviceType: dt,
              operatorCode: 'default',
              indicatorIds: [editingIndicator?.kpiId || ''],
              enable: false,
            }));
          }
        }

        if (promises.length > 0) {
          await Promise.all(promises);
        }
      } else {
        // 自定义指标：走完整 addOrModify 流程
        if (editIndicatorType === 'kpi' && !editCalcFormula.trim()) {
          void message.warning(t('kpi.calcFormulaRequired'));
          return;
        }
        await updateIndicator.mutateAsync({
          kpiId: editingIndicator?.kpiId,
          indicatorType: editIndicatorType,
          kpiName: values.kpiName,
          catagoryId: editingIndicator?.catagoryId || '',
          productClass: values.productClass,
          unit: values.unit,
          statisType: values.statisType,
          isEnable: values.isEnable,
          definition: values.definition,
          custName: values.custName,
          indicatorLevel: values.indicatorLevel,
          arithmetic: editCalcFormula,
          updater: currentUser?.username || 'system',
          deviceType,
        });
      }

      void message.success(t('common.success'));
      setEditDrawerOpen(false);
      editIndicatorForm.resetFields();
      setEditingIndicator(null);
      setFormulaSearchValue('');
      setFormulaSelectedCategory('');
      setFormulaProductClass('');
    } catch (err) {
      const axiosErr = err as AxiosError & { userMessage?: string };
      const errMsg = axiosErr?.userMessage || axiosErr?.message || String(err);
      void message.error(errMsg);
    }
  }, [editIndicatorForm, editCalcFormula, editingIndicator, updateIndicator, updateCustName, enableIndicators, disableIndicators, deviceType, message, t, editIndicatorType, currentUser]);

  // 表格列定义
  const isGNB = deviceType === 'GNB';
  const columns: DataTableColumn<KPIIndicatorRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'kpiId',
      width: 120,
      fixed: 'right',
      render: (_: unknown, row: KPIIndicatorRow) => (
        <Space size={4} style={{ display: 'inline-flex', flexWrap: 'nowrap', whiteSpace: 'nowrap' }}>
          <Button
            type="link"
            size="small"
            onClick={() => handleEditIndicator(row)}
          >
            {t('common.edit')}
          </Button>
          {Boolean(row.isCustomize) && (
            <Button
              type="link"
              size="small"
              danger
              onClick={() => handleDeleteIndicator(row)}
            >
              {t('common.delete')}
            </Button>
          )}
        </Space>
      ),
    },
    {
      key: 'isEnable',
      title: t('kpi.measure'),
      dataIndex: 'isEnable',
      width: 55,
      render: (val: unknown) => (
        <Tag color={val ? 'success' : 'default'}>
          {val ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      key: 'kpiId',
      title: t('kpi.indicatorId'),
      dataIndex: 'kpiId',
      width: 130,
      mono: true,
      copyable: true,
      render: (val: unknown) => (
        <Link
          style={{ fontFamily: 'monospace', fontSize: 11, fontWeight: 500 }}
          onClick={() => void navigate(`/performance/kpi-standard/detail/${deviceType}/${String(val)}`)}
        >
          {String(val)}
        </Link>
      ),
    },
    {
      key: 'kpiName',
      title: t('kpi.indicatorName'),
      dataIndex: 'kpiName',
      ellipsis: true,
    },
    {
      key: 'custName',
      title: t('kpi.customName'),
      dataIndex: 'custName',
      width: 120,
      ellipsis: true,
      render: (val: unknown) => val || '-',
    },
    {
      key: 'productClass',
      title: t('kpi.productClass'),
      dataIndex: 'productClass',
      width: 70,
    },
    {
      key: 'indicatorLevel',
      title: t('kpi.level'),
      dataIndex: 'indicatorLevel',
      width: 65,
      render: (val: unknown) => val === 'device' ? 'Device' : 'PLMN',
    },
    {
      key: 'unit',
      title: t('kpi.unit'),
      dataIndex: 'unit',
      width: 55,
    },
    {
      key: 'isCustomize',
      title: t('kpi.indicatorType'),
      width: 120,
      render: (_: unknown, row: KPIIndicatorRow) => (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 0 }}>
          <Tag style={{ marginRight: 0 }} color={row.indicatorType === 'counter' ? 'green' : 'blue'}>
            {row.indicatorType === 'counter' ? 'Counter' : 'KPI'}
          </Tag>
          <Tag style={{ marginLeft: 2 }} color={row.isCustomize ? 'orange' : 'default'}>
            {row.isCustomize ? t('kpi.custom') : t('kpi.system')}
          </Tag>
        </span>
      ),
    },
    {
      key: 'updater',
      title: t('kpi.updater'),
      dataIndex: 'updater',
      width: 70,
    },
    {
      key: 'updateTime',
      title: t('kpi.updateTime'),
      dataIndex: 'updateTime',
      width: 160,
      render: (val: unknown) => formatDateTime(val as string | undefined),
    },
  ].filter((col) => {
    if (isGNB && (col.key === 'productClass' || col.key === 'indicatorLevel')) return false;
    return true;
  }) as DataTableColumn<KPIIndicatorRow>[], [t, handleDeleteIndicator, handleEditIndicator, navigate, deviceType, isGNB]);

  // Handle add tree node (新增功能集)
  const handleAddTreeNode = useCallback(async () => {
    try {
      const values = await addNodeForm.validateFields();
      await indicatorApi.addIndicatorGroup({
        deviceType,
        enName: values.name,
        cnName: values.name,
        parentId: currentParentKey,
      });
      void message.success(t('common.success'));
      await refetchGroupTree();
      setAddNodeModalOpen(false);
      addNodeForm.resetFields();
    } catch (err) {
      if (err && typeof err === 'object' && 'message' in err) {
        void message.error(String((err as Error).message));
      }
    }
  }, [addNodeForm, currentParentKey, deviceType, message, t]);

  // Handle add function set (top-level "Add" button)
  const handleAddFunctionSet = useCallback(async () => {
    try {
      const values = await addForm.validateFields();
      const rootGroup = groupTreeData.find(g => g.parentId === g.id);
      await indicatorApi.addIndicatorGroup({
        deviceType,
        enName: values.name,
        cnName: values.name,
        parentId: rootGroup?.id || '',
        operatorCode: '',
      });
      void message.success(t('common.success'));
      await refetchGroupTree();
      setAddModalOpen(false);
      addForm.resetFields();
    } catch (err) {
      if (err && typeof err === 'object' && 'message' in err) {
        void message.error(String((err as Error).message));
      }
    }
  }, [addForm, groupTreeData, message, t]);

  // Handle edit tree node
  const handleEditTreeNode = useCallback(async () => {
    try {
      const values = await editNodeForm.validateFields();
      await indicatorApi.modifyIndicatorGroup(
        editingNodeKey,
        {
          cnName: values.name,
          enName: values.name,
          description: values.description,
        },
        deviceType,
      );
      void message.success(t('common.success'));
      await refetchGroupTree();
      setEditNodeModalOpen(false);
      editNodeForm.resetFields();
    } catch (err) {
      if (err && typeof err === 'object' && 'message' in err) {
        void message.error(String((err as Error).message));
      }
    }
  }, [editNodeForm, editingNodeKey, deviceType, message, t]);

  // Handle add indicator
  const handleAddIndicator = useCallback(async () => {
    try {
      const values = await addIndicatorForm.validateFields();
      if (currentIndicatorType === 'kpi' && !calcFormula.trim()) {
        void message.warning(t('kpi.calcFormulaRequired'));
        return;
      }
      await createIndicator.mutateAsync({
        indicatorType: values.indicatorType,
        kpiName: values.kpiName,
        catagoryId: currentParentKey || selectedGroupId || '',
        productClass: values.productClass,
        unit: values.unit,
        statisType: values.statisType,
        isEnable: values.isEnable,
        definition: values.definition,
        custName: values.custName,
        indicatorLevel: values.indicatorLevel,
        arithmetic: calcFormula,
        updater: currentUser?.username || 'system',
        deviceType,
      });
      void message.success(t('common.success'));
      setAddDrawerOpen(false);
      addIndicatorForm.resetFields();
      setCurrentIndicatorType('kpi');
      setCalcFormula('');
      setFormulaSearchValue('');
      setFormulaSelectedCategory('');
      setFormulaProductClass('');
    } catch (err) {
      const axiosErr = err as AxiosError & { userMessage?: string };
      const errMsg = axiosErr?.userMessage || axiosErr?.message || String(err);
      void message.error(errMsg);
    }
  }, [addIndicatorForm, createIndicator, deviceType, selectedGroupId, calcFormula, currentIndicatorType, message, t, currentUser]);

  // Formula handling functions
  const handleAddOperator = useCallback((op: string) => {
    setCalcFormula((prev) => prev + ' ' + op + ' ');
  }, []);

  const handleAddIndicatorToFormula = useCallback((indicatorId: string) => {
    setCalcFormula((prev) => prev + `[${indicatorId}]`);
  }, []);

  const handleClearFormula = useCallback(() => {
    setCalcFormula('');
  }, []);

  // 导出
  const handleExport = useCallback(async () => {
    try {
      const blob = await exportIndicators.mutateAsync({
        deviceType,
        groupId: selectedGroupId,
      });
      // 创建下载链接
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `indicators_${deviceType}_${new Date().toISOString().slice(0, 10)}.xlsx`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      void message.success(t('common.success'));
    } catch (err) {
      void message.error(t('common.exportFailed'));
      console.error('Export failed:', err);
    }
  }, [exportIndicators, deviceType, selectedGroupId, message, t]);

  // 计算公式区域的功能集树数据（从 API 数据转换）
  const formulaTreeData = useMemo((): DataNode[] => {
    const convertGroup = (group: IndicatorGroup): DataNode => ({
      title: getLocalizedGroupName(group, locale),
      key: group.id,
      children: (group.children || []).map(convertGroup),
    });
    return groupTreeData.map(convertGroup);
  }, [groupTreeData, locale]);

  // 公式区域使用的指标列表（复用当前列表数据）
  const formulaFilteredIndicators = useMemo(() => {
    let data = tableData;

    // 按产品类型过滤
    if (formulaProductClass) {
      data = data.filter((row) => row.productClass === formulaProductClass);
    }

    return data;
  }, [tableData, formulaProductClass]);

  // 根据搜索词过滤指标
  const searchedIndicators = useMemo(() => {
    if (!formulaSearchValue) {
      return formulaFilteredIndicators;
    }
    const lowerSearch = formulaSearchValue.toLowerCase();
    return formulaFilteredIndicators.filter(
      (row) =>
        row.kpiId.toLowerCase().includes(lowerSearch) ||
        row.kpiName.toLowerCase().includes(lowerSearch)
    );
  }, [formulaFilteredIndicators, formulaSearchValue]);

  // 设备类型切换时重置选中状态
  const handleDeviceTypeChange = useCallback((newDeviceType: DeviceTypeTab) => {
    setDeviceType(newDeviceType);
    setSelectedCategory('');
    setPage(1);
    setSelectedRowKeys([]);
  }, []);

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* 设备类型标签页 */}
      <div style={{ padding: '8px 8px 0' }}>
        <Radio.Group
          value={deviceType}
          onChange={(e) => handleDeviceTypeChange(e.target.value)}
          size="small"
          optionType="button"
          buttonStyle="solid"
          style={{ width: '100%', display: 'flex' }}
        >
          {DEVICE_TABS.map((tab) => (
            <Radio.Button key={tab.key} value={tab.key} style={{ flex: 1, textAlign: 'center', whiteSpace: 'nowrap', overflow: 'hidden' }}>
              {t(tab.labelKey)}
            </Radio.Button>
          ))}
        </Radio.Group>
      </div>
      <div
        style={{
          padding: '12px 12px 8px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Typography.Title level={5} style={{ margin: 0, fontSize: 14 }}>
          {t('perf.functionSet')}
        </Typography.Title>
        <Button
          type="text"
          size="small"
          icon={<PlusOutlined />}
          onClick={() => {
            setAddModalOpen(true);
            addForm.resetFields();
          }}
        >
          {t('common.add')}
        </Button>
      </div>
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={treeSearchValue}
          onChange={(e) => setTreeSearchValue(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
        <style>{`
          .kpi-tree {
            background: transparent;
            width: 100%;
          }
          .kpi-tree .ant-tree-list {
            width: 100%;
          }
          .kpi-tree .ant-tree-treenode {
            display: flex;
            align-items: center;
            width: 100%;
          }
          .kpi-tree .ant-tree-node-content-wrapper {
            flex: 1;
            min-width: 0;
            display: flex;
            align-items: center;
          }
          .kpi-tree .ant-tree-title {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding-right: 1px;
          }
        `}</style>
        <Tree
          className="kpi-tree"
          treeData={kpiTreeData}
          expandedKeys={useMemo(() => {
            const allKeys: React.Key[] = [];
            const collect = (nodes: DataNode[]) => {
              for (const node of nodes) {
                allKeys.push(node.key);
                if (node.children) collect(node.children);
              }
            };
            collect(kpiTreeData);
            return allKeys;
          }, [kpiTreeData])}
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedCategory(key ?? '');
            setPage(1);
          }}
          showLine
        />
      </div>
    </div>
  );

  return (
    <>
      <TreeListPageLayout tree={treePanel}>
        {/* 标题行 */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '16px 16px 12px' }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {selectedFunctionSetName}
            <Typography.Text type="secondary" style={{ fontSize: 13, marginLeft: 8, fontWeight: 400 }}>
              {t('table.total')} {totalCount}
            </Typography.Text>
          </Typography.Title>
          <Button type="primary" icon={<DownloadOutlined />} onClick={() => void handleExport()} loading={exportIndicators.isPending}>
            {t('common.export')}
          </Button>
        </div>

        {/* 数据表格 */}
        <div className="kpi-table-wrapper" style={{ flex: 1, overflow: 'auto' }}>
          <style>{`
            .kpi-table-wrapper {
              display: flex;
              flex-direction: column;
              height: 100%;
            }
            .kpi-table-wrapper .ant-table-wrapper {
              flex: 1;
              overflow: visible !important;
            }
            .kpi-table-wrapper .ant-table {
              height: 100%;
            }
            .kpi-table-wrapper .ant-table-container {
              height: 100%;
              display: flex;
              flex-direction: column;
            }
            .kpi-table-wrapper [class*="tableContainer"] {
              overflow: visible !important;
            }
            .kpi-table-wrapper .ant-table-body {
              flex: 1;
              overflow-y: auto !important;
              overflow-x: scroll !important;
            }
            .kpi-table-wrapper .ant-table-body::-webkit-scrollbar {
              width: 8px;
              height: 8px;
            }
            .kpi-table-wrapper .ant-table-body::-webkit-scrollbar-thumb {
              background-color: rgba(0, 0, 0, 0.25);
              border-radius: 4px;
            }
            .kpi-table-wrapper .ant-table-body::-webkit-scrollbar-track {
              background-color: rgba(0, 0, 0, 0.05);
              border-radius: 4px;
            }
            .kpi-table-wrapper .ant-table-body::-webkit-scrollbar-corner {
              background-color: rgba(0, 0, 0, 0.05);
            }
          `}</style>
          <DataTable<KPIIndicatorRow>
            tableId="kpi-management"
            columns={columns}
            dataSource={tableData}
            loading={indicatorLoading || groupTreeLoading}
            rowKey="kpiId"
            total={totalCount}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            onRefresh={() => void refetchIndicatorList()}
            scroll={{ x: 1200, y: 'calc(100vh - 320px)' }}
            selectable
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
            selectedRowKeys={selectedRowKeys}
            onSelectionChange={(keys) => {
              handleSelectChange(keys);
            }}
            batchActions={batchActions}
            extraToolbarRight={
              <Space size={8}>
                <Input
                  size="small"
                  placeholder={t('kpi.searchPlaceholder')}
                  prefix={<SearchOutlined />}
                  value={tableSearchValue}
                  onChange={(e) => { setTableSearchValue(e.target.value); setPage(1); }}
                  allowClear
                  style={{ width: 200 }}
                />
                <Input
                  size="small"
                  placeholder={t('kpi.productClass')}
                  value={productClass}
                  onChange={(e) => { setProductClass(e.target.value); setPage(1); }}
                  style={{ width: 120 }}
                  allowClear
                />
                <Select
                  size="small"
                  placeholder={t('kpi.level')}
                  value={indicatorLevel}
                  onChange={(v) => { setIndicatorLevel(v); setPage(1); }}
                  options={levelOptions}
                  style={{ width: 100 }}
                  allowClear
                />
                <Select
                  size="small"
                  placeholder={t('kpi.indicatorType')}
                  value={indicatorTypeFilter}
                  onChange={(v) => { setIndicatorTypeFilter(v); setPage(1); }}
                  options={[
                    { label: t('kpi.all'), value: '' },
                    { label: t('kpi.counter'), value: '1' },
                    { label: t('kpi.kpi'), value: '0' },
                  ]}
                  style={{ width: 110 }}
                  allowClear
                />
                <Select
                  size="small"
                  placeholder={t('kpi.enableStatus')}
                  value={isEnableFilter}
                  onChange={(v) => { setIsEnableFilter(v); setPage(1); }}
                  options={[
                    { label: t('kpi.all'), value: '' },
                    { label: t('kpi.enabled'), value: '1' },
                    { label: t('kpi.disabled'), value: '0' },
                  ]}
                  style={{ width: 100 }}
                  allowClear
                />
              </Space>
            }
          />
        </div>
      </TreeListPageLayout>

      {/* Add Function Set Modal */}
      <Modal
        title={t('common.add')}
        open={addModalOpen}
        onOk={() => void handleAddFunctionSet()}
        onCancel={() => {
          setAddModalOpen(false);
          addForm.resetFields();
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={520}
      >
        <Form
          form={addForm}
          layout="vertical"
          requiredMark="optional"
        >
          <Form.Item
            name="name"
            label={t('kpi.indicatorName')}
            rules={[
              { required: true, message: t('kpi.nameRequired') },
              { max: 200, message: t('kpi.nameMax50') },
            ]}
          >
            <Input
              placeholder={t('kpi.namePlaceholder')}
              maxLength={200}
              showCount
            />
          </Form.Item>
          <Form.Item label={t('kpi.belongFunctionSet')} required>
            <Select
              disabled
              value={deviceType}
              options={DEVICE_TABS.map(dt => ({ label: t(dt.labelKey), value: dt.key }))}
            />
          </Form.Item>
          <Form.Item
            name="description"
            label={t('perf.description')}
          >
            <Input.TextArea
              placeholder={t('perf.descriptionPlaceholder')}
              rows={3}
              maxLength={500}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Tree Node Modal */}
      <Modal
        title={t('common.add')}
        open={addNodeModalOpen}
        onOk={() => void handleAddTreeNode()}
        onCancel={() => {
          setAddNodeModalOpen(false);
          addNodeForm.resetFields();
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={400}
      >
        <Form
          form={addNodeForm}
          layout="vertical"
        >
          <Form.Item
            name="name"
            label={t('common.name')}
            rules={[
              { required: true, message: t('common.nameRequired') },
              { max: 100, message: t('common.nameMax100') },
            ]}
          >
            <Input placeholder={t('common.namePlaceholder')} maxLength={100} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit Tree Node Modal */}
      <Modal
        title={t('common.edit')}
        open={editNodeModalOpen}
        onOk={() => void handleEditTreeNode()}
        onCancel={() => {
          setEditNodeModalOpen(false);
          editNodeForm.resetFields();
        }}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={400}
      >
        <Form
          form={editNodeForm}
          layout="vertical"
        >
          <Form.Item
            name="name"
            label={t('common.name')}
            rules={[
              { required: true, message: t('common.nameRequired') },
              { max: 100, message: t('common.nameMax100') },
            ]}
          >
            <Input placeholder={t('common.namePlaceholder')} maxLength={100} />
          </Form.Item>
          <Form.Item
            name="description"
            label={t('common.description')}
          >
            <Input.TextArea
              placeholder={t('common.descriptionPlaceholder')}
              rows={3}
              maxLength={500}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Add Indicator Drawer */}
      <Drawer
        title={`${t('kpi.addIndicator')} - ${selectedFunctionSetName || t('kpi.allIndicators')}`}
        open={addDrawerOpen}
        onClose={() => {
          setAddDrawerOpen(false);
          addIndicatorForm.resetFields();
          setCurrentIndicatorType('kpi');
          setCalcFormula('');
          setFormulaSearchValue('');
          setFormulaSelectedCategory('');
          setFormulaProductClass('');
        }}
        size={720}
        destroyOnHidden
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => {
              setAddDrawerOpen(false);
              addIndicatorForm.resetFields();
              setCurrentIndicatorType('kpi');
              setCalcFormula('');
              setFormulaSearchValue('');
              setFormulaSelectedCategory('');
              setFormulaProductClass('');
            }}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={() => void handleAddIndicator()} loading={createIndicator.isPending}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form
          form={addIndicatorForm}
          layout="vertical"
          initialValues={{
            indicatorType: 'kpi',
            indicatorLevel: 'device',
          }}
        >
          {/* 基本信息 */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8,
            marginBottom: 16
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('kpi.basicInfo')}
            </div>

            {/* 类型 */}
            <Form.Item name="indicatorType" label={t('kpi.type')} style={{ marginBottom: 12 }}>
              <Radio.Group onChange={(e) => setCurrentIndicatorType(e.target.value)}>
                <Radio value="kpi">{t('kpi.customKpi')}</Radio>
                <Radio value="counter">{t('kpi.customCounter')}</Radio>
              </Radio.Group>
            </Form.Item>

            {/* 等级 */}
            {!isGNB && (
            <Form.Item name="indicatorLevel" label={t('kpi.level')} style={{ marginBottom: 12 }}>
              <Radio.Group>
                <Radio value="device">{t('kpi.deviceLevel')}</Radio>
                <Radio value="plmn">{t('kpi.plmnLevel')}</Radio>
              </Radio.Group>
            </Form.Item>
            )}

            {/* 指标名称 */}
            <Form.Item
              name="kpiName"
              label={t('kpi.counterName')}
              rules={[
                { required: true, message: t('kpi.nameRequired') },
                { max: 50, message: t('kpi.nameMax50') },
              ]}
              style={{ marginBottom: 12 }}
            >
              <Input placeholder={t('kpi.namePlaceholder')} maxLength={50} showCount />
            </Form.Item>

            {!isGNB && (
            <Form.Item
              name="productClass"
              label={t('kpi.productClass')}
              rules={[{ max: 50, message: t('kpi.nameMax50') }]}
              style={{ marginBottom: 12 }}
            >
              <Input placeholder={t('kpi.productClass')} maxLength={50} showCount />
            </Form.Item>
            )}

            {/* 自定义名称 - 仅Counter类型显示 */}
            {currentIndicatorType === 'counter' && (
              <Form.Item
                name="custName"
                label={t('kpi.customName')}
                rules={[{ max: 50, message: t('kpi.nameMax50') }]}
                style={{ marginBottom: 12 }}
              >
                <Input placeholder={t('kpi.customNamePlaceholder')} maxLength={50} showCount />
              </Form.Item>
            )}

            {/* 单位 */}
            <Form.Item
              name="unit"
              label={t('kpi.unit')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 12 }}
            >
              <Select
                placeholder={t('kpi.unitPlaceholder')}
                options={[
                  { label: '%', value: '%' },
                  { label: t('kpi.unitTimes'), value: '次' },
                  { label: 'Mbps', value: 'Mbps' },
                  { label: 'ms', value: 'ms' },
                  { label: 'dBm', value: 'dBm' },
                  { label: 'W', value: 'W' },
                  { label: t('kpi.unitNone'), value: '' },
                ]}
              />
            </Form.Item>

            {/* 统计类型 */}
            <Form.Item
              name="statisType"
              label={t('kpi.statisType')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 12 }}
            >
              <Select
                placeholder={t('kpi.statisTypePlaceholder')}
                options={[
                  { label: t('kpi.statisSum'), value: 'sum' },
                  { label: t('kpi.statisAvg'), value: 'avg' },
                  { label: t('kpi.statisMax'), value: 'max' },
                  { label: t('kpi.statisMin'), value: 'min' },
                  { label: t('kpi.statisPct'), value: 'pct' },
                ]}
              />
            </Form.Item>

            {/* 测量 */}
            <Form.Item
              name="isEnable"
              label={t('kpi.measure')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 0 }}
            >
              <Select
                placeholder={t('kpi.measurePlaceholder')}
                options={[
                  { label: t('common.yes'), value: '1' },
                  { label: t('common.no'), value: '0' },
                ]}
              />
            </Form.Item>
          </div>

          {/* 计算公式 - 仅KPI类型显示 */}
          {currentIndicatorType === 'kpi' && (
            <div style={{
              padding: '16px',
              background: 'var(--color-fill-quaternary)',
              borderRadius: 8,
              marginBottom: 16
            }}>
              {/* 标题和说明 */}
              <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <span style={{ fontWeight: 500, color: 'var(--color-text)', marginRight: 8 }}>
                    {t('kpi.calcFormula')}
                  </span>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {t('kpi.calcFormulaDesc')}
                  </Typography.Text>
                </div>
                <Button size="small" danger onClick={handleClearFormula}>
                  {t('common.clear')}
                </Button>
              </div>

              {/* 公式显示区域 */}
              <div
                style={{
                  minHeight: 56,
                  maxHeight: 140,
                  padding: '10px 12px',
                  background: '#fafafa',
                  color: 'rgba(0, 0, 0, 0.88)',
                  borderRadius: 6,
                  border: '1px solid var(--color-border)',
                  marginBottom: 12,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  fontFamily: 'Consolas, Monaco, monospace',
                  fontSize: 14,
                  lineHeight: 1.5,
                }}
              >
                {calcFormula || (
                  <Typography.Text type="secondary" style={{ fontStyle: 'italic' }}>
                    {t('kpi.calcFormulaPlaceholder')}
                  </Typography.Text>
                )}
              </div>

              {/* 运算符和数字按钮 - 分行排列 */}
              <div style={{ marginBottom: 12 }}>
                <div style={{ marginBottom: 6, fontSize: 12, color: 'var(--color-text-secondary)' }}>
                  {t('kpi.operators')}
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {/* 第一行：数学运算符 + 括号 */}
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Space.Compact size="small">
                      {['+', '-', '*', '/'].map((op) => (
                        <Button key={op} style={{ width: 36 }} onClick={() => handleAddOperator(op)}>
                          {op}
                        </Button>
                      ))}
                    </Space.Compact>
                    <Space.Compact size="small">
                      <Button style={{ width: 36 }} onClick={() => handleAddOperator('(')}>(</Button>
                      <Button style={{ width: 36 }} onClick={() => handleAddOperator(')')}>)</Button>
                    </Space.Compact>
                  </div>
                  {/* 第二行：数字 + Duration */}
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <Space.Compact size="small">
                      {['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.'].map((num) => (
                        <Button key={num} style={{ width: 28, padding: '0 4px', fontSize: 12 }} onClick={() => handleAddOperator(num)}>
                          {num}
                        </Button>
                      ))}
                    </Space.Compact>
                    <Button size="small" onClick={() => handleAddOperator('Duration')}>
                      {t('kpi.duration')}
                    </Button>
                  </div>
                </div>
              </div>

              {/* 可选指标区域 */}
              <div>
                <div style={{ marginBottom: 6, fontSize: 12, color: 'var(--color-text-secondary)' }}>
                  {t('kpi.availableIndicators')}
                </div>
                {/* 搜索框 + 产品类型筛选 */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                  <Input
                    size="small"
                    placeholder={t('kpi.searchPlaceholder')}
                    prefix={<SearchOutlined />}
                    value={formulaSearchValue}
                    onChange={(e) => setFormulaSearchValue(e.target.value)}
                    allowClear
                    style={{ flex: 1 }}
                  />
                  <Input
                    size="small"
                    placeholder={t('kpi.productClass')}
                    value={formulaProductClass}
                    onChange={(e) => setFormulaProductClass(e.target.value)}
                    style={{ width: 100, flexShrink: 0 }}
                    allowClear
                  />
                </div>
                {/* 左右分栏 */}
                <div style={{ display: 'flex', gap: 8, height: 260 }}>
                  {/* 左侧：功能集树 */}
                  <div
                    style={{
                      width: 140,
                      flexShrink: 0,
                      border: '1px solid var(--color-border)',
                      borderRadius: 6,
                      background: 'var(--color-bg-container)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                    }}
                  >
                    <div style={{
                      padding: '6px 10px',
                      borderBottom: '1px solid var(--color-border)',
                      fontWeight: 500,
                      fontSize: 12,
                      background: 'var(--color-fill-quaternary)',
                      flexShrink: 0,
                    }}>
                      {t('perf.functionSet')}
                    </div>
                    <div style={{ flex: 1, overflow: 'auto' }}>
                      <Tree
                        treeData={formulaTreeData}
                        selectedKeys={formulaSelectedCategory ? [formulaSelectedCategory] : []}
                        onSelect={(keys) => {
                          setFormulaSelectedCategory((keys[0] as string) || '');
                        }}
                        defaultExpandAll
                        style={{ fontSize: 12, padding: '4px 0' }}
                      />
                    </div>
                  </div>
                  {/* 右侧：指标列表 */}
                  <div
                    style={{
                      flex: 1,
                      border: '1px solid var(--color-border)',
                      borderRadius: 6,
                      background: 'var(--color-bg-container)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                    }}
                  >
                    <div style={{
                      padding: '6px 10px',
                      borderBottom: '1px solid var(--color-border)',
                      fontWeight: 500,
                      fontSize: 12,
                      background: 'var(--color-fill-quaternary)',
                      flexShrink: 0,
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}>
                      <span>{t('kpi.performanceIndicator')}</span>
                      <span style={{ fontWeight: 400, color: 'var(--color-text-tertiary)' }}>
                        {searchedIndicators.length}
                      </span>
                    </div>
                    <div style={{ flex: 1, overflow: 'auto' }}>
                      {searchedIndicators.length > 0 ? (
                        searchedIndicators.map((indicator) => (
                          <div
                            key={indicator.kpiId}
                            style={{
                              padding: '5px 10px',
                              borderBottom: '1px solid var(--color-border-secondary)',
                              cursor: 'pointer',
                              transition: 'background 0.15s',
                            }}
                            onClick={() => handleAddIndicatorToFormula(indicator.kpiId)}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.background = 'var(--color-primary-bg)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.background = 'transparent';
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <code style={{
                                fontSize: 11,
                                color: 'var(--color-primary)',
                                background: 'var(--color-primary-bg)',
                                padding: '1px 4px',
                                borderRadius: 3,
                              }}>
                                {indicator.kpiId}
                              </code>
                              <span style={{
                                fontSize: 12,
                                color: 'var(--color-text)',
                                flex: 1,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                              }}>
                                {indicator.kpiName}
                              </span>
                              <Tag style={{ fontSize: 10, lineHeight: '16px', margin: 0 }}>
                                {indicator.unit || '-'}
                              </Tag>
                            </div>
                          </div>
                        ))
                      ) : (
                        <div style={{ padding: 24, textAlign: 'center', color: 'var(--color-text-tertiary)' }}>
                          {t('common.noData')}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* 说明 */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('kpi.definition')}
            </div>
            <Form.Item name="definition" style={{ marginBottom: 0 }}>
              <Input.TextArea
                placeholder={t('kpi.definitionPlaceholder')}
                rows={4}
                maxLength={2000}
                showCount
              />
            </Form.Item>
          </div>
        </Form>
      </Drawer>

      {/* Edit Indicator Drawer */}
      <Drawer
        title={`${t('common.edit')} - ${editingIndicator?.kpiName || ''}`}
        open={editDrawerOpen}
        onClose={() => {
          setEditDrawerOpen(false);
          editIndicatorForm.resetFields();
          setEditingIndicator(null);
          setFormulaSearchValue('');
          setFormulaSelectedCategory('');
          setFormulaProductClass('');
        }}
        size={720}
        destroyOnHidden
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => {
              setEditDrawerOpen(false);
              editIndicatorForm.resetFields();
              setEditingIndicator(null);
              setFormulaSearchValue('');
              setFormulaSelectedCategory('');
              setFormulaProductClass('');
            }}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={() => void handleSaveEditIndicator()} loading={updateIndicator.isPending}>
              {t('common.confirm')}
            </Button>
          </div>
        }
      >
        <Form
          form={editIndicatorForm}
          layout="vertical"
          initialValues={{
            indicatorType: 'kpi',
            indicatorLevel: 'device',
          }}
        >
          {/* 基本信息 */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8,
            marginBottom: 16
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('kpi.basicInfo')}
            </div>

            {/* 类型 - 只读 */}
            <Form.Item label={t('kpi.type')} style={{ marginBottom: 12 }}>
              <Space size={4}>
                <Tag color={editIndicatorType === 'counter' ? 'green' : 'blue'}>
                  {editIndicatorType === 'counter' ? 'Counter' : 'KPI'}
                </Tag>
                <Tag color={editingIndicator?.isCustomize ? 'orange' : 'default'}>
                  {editingIndicator?.isCustomize ? t('kpi.custom') : t('kpi.system')}
                </Tag>
              </Space>
            </Form.Item>

            {/* 等级 */}
            {!isGNB && (
            <Form.Item name="indicatorLevel" label={t('kpi.level')} style={{ marginBottom: 12 }}>
              <Radio.Group disabled={!editingIndicator?.isCustomize}>
                <Radio value="device">{t('kpi.deviceLevel')}</Radio>
                <Radio value="plmn">{t('kpi.plmnLevel')}</Radio>
              </Radio.Group>
            </Form.Item>
            )}

            {/* 指标名称 */}
            <Form.Item
              name="kpiName"
              label={t('kpi.counterName')}
              rules={[
                { required: true, message: t('kpi.nameRequired') },
                { max: 50, message: t('kpi.nameMax50') },
              ]}
              style={{ marginBottom: 12 }}
            >
              <Input placeholder={t('kpi.namePlaceholder')} maxLength={50} showCount disabled={!editingIndicator?.isCustomize} />
            </Form.Item>

            {!isGNB && (
            <Form.Item
              name="productClass"
              label={t('kpi.productClass')}
              rules={[{ max: 50, message: t('kpi.nameMax50') }]}
              style={{ marginBottom: 12 }}
            >
              <Input
                placeholder={t('kpi.productClass')}
                maxLength={50}
                showCount
                disabled={!editingIndicator?.isCustomize}
              />
            </Form.Item>
            )}

            {/* 自定义名称 */}
            <Form.Item
              name="custName"
              label={t('kpi.customName')}
              rules={[{ max: 50, message: t('kpi.nameMax50') }]}
              style={{ marginBottom: 12 }}
            >
              <Input placeholder={t('kpi.customNamePlaceholder')} maxLength={50} showCount />
            </Form.Item>

            {/* 单位 */}
            <Form.Item
              name="unit"
              label={t('kpi.unit')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 12 }}
            >
              <Select
                placeholder={t('kpi.unitPlaceholder')}
                disabled={!editingIndicator?.isCustomize}
                options={[
                  { label: '%', value: '%' },
                  { label: t('kpi.unitTimes'), value: '次' },
                  { label: 'Mbps', value: 'Mbps' },
                  { label: 'ms', value: 'ms' },
                  { label: 'dBm', value: 'dBm' },
                  { label: 'W', value: 'W' },
                  { label: t('kpi.unitNone'), value: '' },
                ]}
              />
            </Form.Item>

            {/* 统计类型 */}
            <Form.Item
              name="statisType"
              label={t('kpi.statisType')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 12 }}
            >
              <Select
                placeholder={t('kpi.statisTypePlaceholder')}
                disabled={!editingIndicator?.isCustomize}
                options={[
                  { label: t('kpi.statisSum'), value: 'sum' },
                  { label: t('kpi.statisAvg'), value: 'avg' },
                  { label: t('kpi.statisMax'), value: 'max' },
                  { label: t('kpi.statisMin'), value: 'min' },
                  { label: t('kpi.statisPct'), value: 'pct' },
                ]}
              />
            </Form.Item>

            {/* 测量 */}
            <Form.Item
              name="isEnable"
              label={t('kpi.measure')}
              rules={[{ required: true, message: t('common.selectRequired') }]}
              style={{ marginBottom: 0 }}
            >
              <Select
                placeholder={t('kpi.measurePlaceholder')}
                options={[
                  { label: t('common.yes'), value: '1' },
                  { label: t('common.no'), value: '0' },
                ]}
              />
            </Form.Item>
          </div>

          {/* 计算公式 - 仅KPI类型显示 */}
          {editIndicatorType === 'kpi' && (
            <div style={{
              padding: '16px',
              background: 'var(--color-fill-quaternary)',
              borderRadius: 8,
              marginBottom: 16
            }}>
              {/* 标题和说明 */}
              <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <span style={{ fontWeight: 500, color: 'var(--color-text)', marginRight: 8 }}>
                    {t('kpi.calcFormula')}
                  </span>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {t('kpi.calcFormulaDesc')}
                  </Typography.Text>
                </div>
                {editingIndicator?.isCustomize && (
                  <Button size="small" danger onClick={handleEditClearFormula}>
                    {t('common.clear')}
                  </Button>
                )}
              </div>

              {/* 公式显示区域 */}
              <div
                style={{
                  minHeight: 56,
                  maxHeight: 140,
                  padding: '10px 12px',
                  background: '#fafafa',
                  color: 'rgba(0, 0, 0, 0.88)',
                  borderRadius: 6,
                  border: '1px solid var(--color-border)',
                  marginBottom: 12,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  fontFamily: 'Consolas, Monaco, monospace',
                  fontSize: 14,
                  lineHeight: 1.5,
                }}
              >
                {editCalcFormula || (
                  <Typography.Text type="secondary" style={{ fontStyle: 'italic' }}>
                    {t('kpi.calcFormulaPlaceholder')}
                  </Typography.Text>
                )}
              </div>

              {/* 运算符和数字按钮 - 分行排列 */}
              <div style={{ marginBottom: 12 }}>
                <div style={{ marginBottom: 6, fontSize: 12, color: 'var(--color-text-secondary)' }}>
                  {t('kpi.operators')}
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {/* 第一行：数学运算符 + 括号 */}
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Space.Compact size="small">
                      {['+', '-', '*', '/'].map((op) => (
                        <Button key={op} style={{ width: 36 }} onClick={() => handleEditAddOperator(op)} disabled={!editingIndicator?.isCustomize}>
                          {op}
                        </Button>
                      ))}
                    </Space.Compact>
                    <Space.Compact size="small">
                      <Button style={{ width: 36 }} onClick={() => handleEditAddOperator('(')} disabled={!editingIndicator?.isCustomize}>(</Button>
                      <Button style={{ width: 36 }} onClick={() => handleEditAddOperator(')')} disabled={!editingIndicator?.isCustomize}>)</Button>
                    </Space.Compact>
                  </div>
                  {/* 第二行：数字 + Duration */}
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <Space.Compact size="small">
                      {['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.'].map((num) => (
                        <Button key={num} style={{ width: 28, padding: '0 4px', fontSize: 12 }} onClick={() => handleEditAddOperator(num)} disabled={!editingIndicator?.isCustomize}>
                          {num}
                        </Button>
                      ))}
                    </Space.Compact>
                    <Button size="small" onClick={() => handleEditAddOperator('Duration')} disabled={!editingIndicator?.isCustomize}>
                      {t('kpi.duration')}
                    </Button>
                  </div>
                </div>
              </div>

              {/* 可选指标区域 */}
              <div>
                <div style={{ marginBottom: 6, fontSize: 12, color: 'var(--color-text-secondary)' }}>
                  {t('kpi.availableIndicators')}
                </div>
                {/* 搜索框 + 产品类型筛选 */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                  <Input
                    size="small"
                    placeholder={t('kpi.searchPlaceholder')}
                    prefix={<SearchOutlined />}
                    value={formulaSearchValue}
                    onChange={(e) => setFormulaSearchValue(e.target.value)}
                    allowClear
                    style={{ flex: 1 }}
                  />
                  <Input
                    size="small"
                    placeholder={t('kpi.productClass')}
                    value={formulaProductClass}
                    onChange={(e) => setFormulaProductClass(e.target.value)}
                    style={{ width: 100, flexShrink: 0 }}
                    allowClear
                  />
                </div>
                {/* 左右分栏 */}
                <div style={{ display: 'flex', gap: 8, height: 260 }}>
                  {/* 左侧：功能集树 */}
                  <div
                    style={{
                      width: 140,
                      flexShrink: 0,
                      border: '1px solid var(--color-border)',
                      borderRadius: 6,
                      background: 'var(--color-bg-container)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                    }}
                  >
                    <div style={{
                      padding: '6px 10px',
                      borderBottom: '1px solid var(--color-border)',
                      fontWeight: 500,
                      fontSize: 12,
                      background: 'var(--color-fill-quaternary)',
                      flexShrink: 0,
                    }}>
                      {t('perf.functionSet')}
                    </div>
                    <div style={{ flex: 1, overflow: 'auto' }}>
                      <Tree
                        treeData={formulaTreeData}
                        selectedKeys={formulaSelectedCategory ? [formulaSelectedCategory] : []}
                        onSelect={(keys) => {
                          setFormulaSelectedCategory((keys[0] as string) || '');
                        }}
                        defaultExpandAll
                        style={{ fontSize: 12, padding: '4px 0' }}
                      />
                    </div>
                  </div>
                  {/* 右侧：指标列表 */}
                  <div
                    style={{
                      flex: 1,
                      border: '1px solid var(--color-border)',
                      borderRadius: 6,
                      background: 'var(--color-bg-container)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                    }}
                  >
                    <div style={{
                      padding: '6px 10px',
                      borderBottom: '1px solid var(--color-border)',
                      fontWeight: 500,
                      fontSize: 12,
                      background: 'var(--color-fill-quaternary)',
                      flexShrink: 0,
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                    }}>
                      <span>{t('kpi.performanceIndicator')}</span>
                      <span style={{ fontWeight: 400, color: 'var(--color-text-tertiary)' }}>
                        {searchedIndicators.length}
                      </span>
                    </div>
                    <div style={{ flex: 1, overflow: 'auto' }}>
                      {searchedIndicators.length > 0 ? (
                        searchedIndicators.map((indicator) => (
                          <div
                            key={indicator.kpiId}
                            style={{
                              padding: '5px 10px',
                              borderBottom: '1px solid var(--color-border-secondary)',
                              cursor: 'pointer',
                              transition: 'background 0.15s',
                            }}
                            onClick={() => editingIndicator?.isCustomize && handleEditAddIndicatorToFormula(indicator.kpiId)}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.background = 'var(--color-primary-bg)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.background = 'transparent';
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <code style={{
                                fontSize: 11,
                                color: 'var(--color-primary)',
                                background: 'var(--color-primary-bg)',
                                padding: '1px 4px',
                                borderRadius: 3,
                              }}>
                                {indicator.kpiId}
                              </code>
                              <span style={{
                                fontSize: 12,
                                color: 'var(--color-text)',
                                flex: 1,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                              }}>
                                {indicator.kpiName}
                              </span>
                              <Tag style={{ fontSize: 10, lineHeight: '16px', margin: 0 }}>
                                {indicator.unit || '-'}
                              </Tag>
                            </div>
                          </div>
                        ))
                      ) : (
                        <div style={{ padding: 24, textAlign: 'center', color: 'var(--color-text-tertiary)' }}>
                          {t('common.noData')}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* 说明 */}
          <div style={{
            padding: '16px',
            background: 'var(--color-fill-quaternary)',
            borderRadius: 8
          }}>
            <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
              {t('kpi.definition')}
            </div>
            <Form.Item name="definition" style={{ marginBottom: 0 }}>
              <Input.TextArea
                placeholder={t('kpi.definitionPlaceholder')}
                rows={4}
                maxLength={2000}
                disabled={!editingIndicator?.isCustomize}
                showCount
              />
            </Form.Item>
          </div>
        </Form>
      </Drawer>
    </>
  );
}
