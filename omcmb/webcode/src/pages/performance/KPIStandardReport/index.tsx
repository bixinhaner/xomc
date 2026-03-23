import { useState, useMemo, useCallback } from 'react';
import { App, Button, Drawer, Form, Input, Modal, Radio, Select, Space, Tag, Tree, Typography } from 'antd';
import { PlusOutlined, SearchOutlined, DownloadOutlined, DeleteOutlined, EditOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// KPI 指标数据接口
interface KPIIndicatorRow extends Record<string, unknown> {
  kpiId: string;                    // 指标ID
  kpiName: string;                  // 指标名称
  productType: string;              // 产品类型
  custName: string;                 // 自定义指标名称
  indicatorLevel: 'device' | 'plmn'; // 等级
  unit: string;                     // 单位
  isCustomize: 0 | 1;               // 是否自定义指标
  isEnable: 0 | 1;                  // 是否启用测量
  indicatorType: 'counter' | 'kpi'; // 指标类型
  updater: string;                  // 更新人
  updateTime: string;               // 更新时间
}

// 指标功能集数据
interface FunctionSetItem {
  id: string;
  name: string;
  networkType: 'eNB' | 'gNB' | 'GSM';
  description: string;
  kpiCount: number;
}

const MOCK_FUNCTION_SETS: FunctionSetItem[] = [
  { id: 'fs-1', name: '接入类指标', networkType: 'eNB', description: 'RRC、ERAB等接入相关KPI', kpiCount: 15 },
  { id: 'fs-2', name: '切换类指标', networkType: 'eNB', description: '切换成功率相关KPI', kpiCount: 8 },
  { id: 'fs-3', name: '吞吐量指标', networkType: 'gNB', description: '上下行吞吐量KPI', kpiCount: 12 },
  { id: 'fs-4', name: '可用性指标', networkType: 'GSM', description: '基站可用性KPI', kpiCount: 6 },
];

// Mock KPI 指标数据
const MOCK_KPI_DATA: KPIIndicatorRow[] = [
  { kpiId: 'RRC_CONN_REQ', kpiName: 'RRC连接请求次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 1, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-20 10:30:00' },
  { kpiId: 'RRC_CONN_SUCC', kpiName: 'RRC连接成功次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 1, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-20 10:30:00' },
  { kpiId: 'RRC_SR', kpiName: 'RRC连接成功率', productType: 'BBU', custName: '接入成功率', indicatorLevel: 'plmn', unit: '%', isCustomize: 1, isEnable: 1, indicatorType: 'kpi', updater: 'user1', updateTime: '2024-03-21 14:20:00' },
  { kpiId: 'ERAB_SETUP_REQ', kpiName: 'ERAB建立请求次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 0, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-19 09:15:00' },
  { kpiId: 'ERAB_SETUP_SUCC', kpiName: 'ERAB建立成功次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 1, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-19 09:15:00' },
  { kpiId: 'HO_EXEC', kpiName: '切换执行次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 1, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-18 16:45:00' },
  { kpiId: 'HO_SUCC', kpiName: '切换成功次数', productType: 'BBU', custName: '', indicatorLevel: 'device', unit: '次', isCustomize: 0, isEnable: 1, indicatorType: 'counter', updater: 'admin', updateTime: '2024-03-18 16:45:00' },
  { kpiId: 'DL_THROUGHPUT', kpiName: '下行吞吐量', productType: 'BBU', custName: '', indicatorLevel: 'plmn', unit: 'Mbps', isCustomize: 0, isEnable: 1, indicatorType: 'kpi', updater: 'admin', updateTime: '2024-03-17 11:00:00' },
  { kpiId: 'UL_THROUGHPUT', kpiName: '上行吞吐量', productType: 'BBU', custName: '', indicatorLevel: 'plmn', unit: 'Mbps', isCustomize: 0, isEnable: 1, indicatorType: 'kpi', updater: 'admin', updateTime: '2024-03-17 11:00:00' },
  { kpiId: 'CUSTOM_KPI_001', kpiName: '自定义接入指标', productType: 'BBU', custName: '我的接入指标', indicatorLevel: 'device', unit: '%', isCustomize: 1, isEnable: 1, indicatorType: 'kpi', updater: 'user1', updateTime: '2024-03-22 08:30:00' },
];

// 二级节点配置
const SECOND_LEVEL_NODES = [
  { key: 'call', labelKey: 'kpi.tree.call' },
  { key: 'context', labelKey: 'kpi.tree.context' },
  { key: 'customize', labelKey: 'kpi.tree.customize' },
  { key: 'data', labelKey: 'kpi.tree.data' },
  { key: 'drb', labelKey: 'kpi.tree.drb' },
  { key: 'endc-mn', labelKey: 'kpi.tree.endcMn' },
  { key: 'eqpt', labelKey: 'kpi.tree.eqpt' },
  { key: 'erab', labelKey: 'kpi.tree.erab' },
  { key: 'ho', labelKey: 'kpi.tree.ho' },
  { key: 'custom', labelKey: 'kpi.tree.custom' },
];

// 产品类型选项
const PRODUCT_TYPE_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'BBU', value: 'BBU' },
  { label: 'RRU', value: 'RRU' },
  { label: 'AAU', value: 'AAU' },
];

// 等级选项
const LEVEL_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'Device', value: 'device' },
  { label: 'PLMN', value: 'plmn' },
];

// 树节点带悬停图标的渲染
interface TreeNodeTitleProps {
  title: string;
  nodeKey: string;
  isCustom?: boolean;
  onAdd?: (key: string) => void;
  onDelete?: (key: string) => void;
}

function TreeNodeTitle({ title, nodeKey, isCustom, onAdd, onDelete }: TreeNodeTitleProps) {
  const [hovered, setHovered] = useState(false);

  return (
    <div
      style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%', minHeight: 24 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <span>{title}</span>
      {hovered && onAdd && (
        <Space size={0}>
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

function useKPITreeData(
  t: (id: string) => string,
  customNodes: { key: string; parentKey: string; title: string }[],
  onAddNode: (parentKey: string) => void,
  onDeleteNode: (nodeKey: string) => void,
  searchValue: string
): DataNode[] {
  return useMemo(() => {
    const lowerSearch = searchValue.toLowerCase();

    // 检查节点是否匹配搜索
    const nodeMatchesSearch = (label: string): boolean => {
      if (!searchValue) return true;
      return label.toLowerCase().includes(lowerSearch);
    };

    // 构建二级节点
    const buildSecondLevelNodes = (parentKey: string): DataNode[] => {
      // 预定义节点
      const predefinedNodes: DataNode[] = SECOND_LEVEL_NODES.map((node) => {
        const nodeKey = `${parentKey}-${node.key}`;
        const isCustom = node.key === 'custom';
        const label = t(node.labelKey);

        // 查找该节点下的自定义子节点
        const childCustomNodes = customNodes.filter((cn) => cn.parentKey === nodeKey);

        // 过滤子节点
        const filteredChildren = childCustomNodes.filter((cn) => nodeMatchesSearch(cn.title));

        // 如果有搜索词，检查是否匹配
        const selfMatches = nodeMatchesSearch(label);
        const hasMatchingChildren = filteredChildren.length > 0;

        // 如果搜索词存在且节点和子节点都不匹配，则不显示
        if (searchValue && !selfMatches && !hasMatchingChildren) {
          return null;
        }

        return {
          title: (
            <TreeNodeTitle
              title={label}
              nodeKey={nodeKey}
              isCustom={isCustom}
              onAdd={onAddNode}
              onDelete={isCustom ? onDeleteNode : undefined}
            />
          ),
          key: nodeKey,
          children: filteredChildren.length > 0 ? filteredChildren.map((cn) => ({
            title: (
              <TreeNodeTitle
                title={cn.title}
                nodeKey={cn.key}
                isCustom
                onAdd={onAddNode}
                onDelete={onDeleteNode}
              />
            ),
            key: cn.key,
            isLeaf: true,
          })) : undefined,
        };
      }).filter(Boolean) as DataNode[];

      return predefinedNodes;
    };

    // 构建一级节点
    const buildFirstLevelNodes = (): DataNode[] => {
      const nodes = [
        { key: 'enb-set', labelKey: 'kpi.tree.enbSet' },
        { key: 'gnb-set', labelKey: 'kpi.tree.gnbSet' },
        { key: 'gsm-set', labelKey: 'kpi.tree.gsmSet' },
      ];

      return nodes.map((node) => {
        const children = buildSecondLevelNodes(node.key);
        const label = t(node.labelKey);

        // 如果有搜索词，检查一级节点标签或子节点是否匹配
        if (searchValue) {
          const labelMatches = nodeMatchesSearch(label);
          const hasChildren = children.length > 0;
          if (!labelMatches && !hasChildren) {
            return null;
          }
        }

        return {
          title: label,
          key: node.key,
          children: children.length > 0 ? children : undefined,
        };
      }).filter(Boolean) as DataNode[];
    };

    const firstLevelNodes = buildFirstLevelNodes();

    // 检查根节点是否匹配
    if (searchValue) {
      const rootLabel = t('kpi.tree.all');
      if (!nodeMatchesSearch(rootLabel) && firstLevelNodes.length === 0) {
        return [];
      }
    }

    return [
      {
        title: t('kpi.tree.all'),
        key: 'all',
        children: firstLevelNodes,
      },
    ];
  }, [t, customNodes, onAddNode, onDeleteNode, searchValue]);
}

export default function KPIStandardReport() {
  const t = useT();
  const { message, modal } = App.useApp();
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [treeSearchValue, setTreeSearchValue] = useState('');
  const [tableSearchValue, setTableSearchValue] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // 筛选状态
  const [productType, setProductType] = useState<string>('');
  const [indicatorLevel, setIndicatorLevel] = useState<string>('');

  // 已选指标
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<KPIIndicatorRow[]>([]);

  // Add function set modal state
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [addForm] = Form.useForm<{ name: string; networkType: 'eNB' | 'gNB' | 'GSM'; description: string }>();
  const [functionSets, setFunctionSets] = useState<FunctionSetItem[]>(MOCK_FUNCTION_SETS);

  // Add indicator drawer state
  const [addDrawerOpen, setAddDrawerOpen] = useState(false);
  const [addIndicatorForm] = Form.useForm<{
    indicatorType: 'kpi' | 'counter';
    indicatorLevel: 'device' | 'plmn';
    kpiName: string;
    custName: string;
    catagoryId: string;
    unit: string;
    statisType: string;
    isEnable: string;
    definition: string;
  }>();
  const [currentIndicatorType, setCurrentIndicatorType] = useState<'kpi' | 'counter'>('kpi');

  // Custom tree nodes state
  const [customNodes, setCustomNodes] = useState<{ key: string; parentKey: string; title: string }[]>([]);

  // Add node modal state
  const [addNodeModalOpen, setAddNodeModalOpen] = useState(false);
  const [addNodeForm] = Form.useForm<{ name: string }>();
  const [currentParentKey, setCurrentParentKey] = useState<string>('');

  // 当前选中的功能集名称
  const selectedFunctionSetName = useMemo(() => {
    if (!selectedCategory) return t('kpi.allIndicators');
    // 从树节点 key 中提取名称
    return selectedCategory;
  }, [selectedCategory, t]);

  // Handle add node to tree - 打开新建指标抽屉
  const handleAddNode = useCallback((parentKey: string) => {
    setCurrentParentKey(parentKey);
    // 打开新建指标抽屉，而不是简单的 Modal
    setAddDrawerOpen(true);
    addIndicatorForm.resetFields();
    setCurrentIndicatorType('kpi');
  }, [addIndicatorForm]);

  // Handle delete node from tree
  const handleDeleteNode = useCallback((nodeKey: string) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirmDelete'),
      onOk: () => {
        setCustomNodes((prev) => prev.filter((n) => n.key !== nodeKey));
        void message.success(t('common.success'));
      },
    });
  }, [modal, message, t]);

  const kpiTreeData = useKPITreeData(t, customNodes, handleAddNode, handleDeleteNode, treeSearchValue);

  // 过滤数据
  const filteredData = useMemo(() => {
    let data = [...MOCK_KPI_DATA];

    // 按搜索词过滤
    if (tableSearchValue) {
      const lowerSearch = tableSearchValue.toLowerCase();
      data = data.filter(
        (row) =>
          row.kpiId.toLowerCase().includes(lowerSearch) ||
          row.kpiName.toLowerCase().includes(lowerSearch)
      );
    }

    // 按产品类型过滤
    if (productType) {
      data = data.filter((row) => row.productType === productType);
    }

    // 按等级过滤
    if (indicatorLevel) {
      data = data.filter((row) => row.indicatorLevel === indicatorLevel);
    }

    return data;
  }, [tableSearchValue, productType, indicatorLevel]);

  // 处理选中行变化
  const handleSelectChange = useCallback((newSelectedRowKeys: React.Key[], newSelectedRows: KPIIndicatorRow[]) => {
    setSelectedRowKeys(newSelectedRowKeys);
    setSelectedRows(newSelectedRows);
  }, []);

  // 删除自定义指标
  const handleDeleteIndicator = useCallback((kpiId: string) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('common.confirmDelete'),
      onOk: () => {
        void message.success(t('common.success'));
      },
    });
  }, [modal, message, t]);

  // 批量操作定义
  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'enableMeasure',
      label: t('kpi.measure'),
      icon: <CheckOutlined />,
      onClick: (keys: React.Key[]) => {
        modal.confirm({
          title: t('common.confirm'),
          content: t('kpi.confirmEnableMeasure'),
          onOk: () => {
            void message.success(t('common.success'));
            setSelectedRowKeys([]);
            setSelectedRows([]);
          },
        });
      },
    },
    {
      key: 'disableMeasure',
      label: t('kpi.cancelMeasure'),
      icon: <CloseOutlined />,
      onClick: (keys: React.Key[]) => {
        modal.confirm({
          title: t('common.confirm'),
          content: t('kpi.confirmDisableMeasure'),
          onOk: () => {
            void message.success(t('common.success'));
            setSelectedRowKeys([]);
            setSelectedRows([]);
          },
        });
      },
    },
  ], [modal, message, t]);

  // 表格列定义
  const columns: DataTableColumn<KPIIndicatorRow>[] = useMemo(() => [
    {
      key: 'operation',
      title: '',
      dataIndex: 'kpiId',
      width: 120,
      fixed: 'left',
      render: (_, row) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
          >
            {t('common.edit')}
          </Button>
          {row.isCustomize === 1 && (
            <Button
              type="link"
              size="small"
              icon={<DeleteOutlined />}
              danger
              onClick={() => handleDeleteIndicator(row.kpiId)}
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
      width: 60,
      render: (val) => (
        <Tag color={val === 1 ? 'success' : 'default'}>
          {val === 1 ? t('common.yes') : t('common.no')}
        </Tag>
      ),
    },
    {
      key: 'kpiId',
      title: t('kpi.counterId'),
      dataIndex: 'kpiId',
      width: 140,
      mono: true,
      copyable: true,
    },
    {
      key: 'kpiName',
      title: t('kpi.counterName'),
      dataIndex: 'kpiName',
      width: 180,
    },
    {
      key: 'productType',
      title: t('kpi.productType'),
      dataIndex: 'productType',
      width: 90,
    },
    {
      key: 'custName',
      title: t('kpi.customName'),
      dataIndex: 'custName',
      width: 140,
      render: (val) => val || '-',
    },
    {
      key: 'indicatorLevel',
      title: t('kpi.level'),
      dataIndex: 'indicatorLevel',
      width: 70,
      render: (val) => val === 'device' ? 'Device' : 'PLMN',
    },
    {
      key: 'unit',
      title: t('kpi.unit'),
      dataIndex: 'unit',
      width: 60,
    },
    {
      key: 'isCustomize',
      title: t('kpi.indicatorType'),
      dataIndex: 'isCustomize',
      width: 90,
      render: (val) => (
        <Tag color={val === 1 ? 'blue' : 'default'}>
          {val === 1 ? t('kpi.customIndicator') : t('kpi.baseIndicator')}
        </Tag>
      ),
    },
    {
      key: 'updater',
      title: t('kpi.updater'),
      dataIndex: 'updater',
      width: 80,
    },
    {
      key: 'updateTime',
      title: t('kpi.updateTime'),
      dataIndex: 'updateTime',
      width: 150,
    },
  ], [t, handleDeleteIndicator]);

  // Handle add function set
  const handleAddFunctionSet = useCallback(async () => {
    try {
      const values = await addForm.validateFields();

      // Check for duplicate name
      const isDuplicate = functionSets.some(
        (fs) => fs.name.toLowerCase() === values.name.toLowerCase()
      );
      if (isDuplicate) {
        void message.error(t('perf.functionSetNameDuplicate'));
        return;
      }

      // Add new function set
      const newItem: FunctionSetItem = {
        id: `fs-${Date.now()}`,
        name: values.name,
        networkType: values.networkType,
        description: values.description || '',
        kpiCount: 0,
      };
      setFunctionSets((prev) => [...prev, newItem]);
      void message.success(t('common.success'));
      setAddModalOpen(false);
      addForm.resetFields();
    } catch {
      // validation error
    }
  }, [addForm, functionSets, message, t]);

  // Handle add tree node
  const handleAddTreeNode = useCallback(async () => {
    try {
      const values = await addNodeForm.validateFields();
      const newNode = {
        key: `node-${Date.now()}`,
        parentKey: currentParentKey,
        title: values.name,
      };
      setCustomNodes((prev) => [...prev, newNode]);
      void message.success(t('common.success'));
      setAddNodeModalOpen(false);
      addNodeForm.resetFields();
    } catch {
      // validation error
    }
  }, [addNodeForm, currentParentKey, message, t]);

  // Handle add indicator
  const handleAddIndicator = useCallback(async () => {
    try {
      const values = await addIndicatorForm.validateFields();
      console.log('Add indicator values:', values);
      // TODO: Call API to add indicator
      void message.success(t('common.success'));
      setAddDrawerOpen(false);
      addIndicatorForm.resetFields();
      setCurrentIndicatorType('kpi');
    } catch {
      // validation error
    }
  }, [addIndicatorForm, message, t]);

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
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
            setAddDrawerOpen(true);
            addIndicatorForm.resetFields();
            setCurrentIndicatorType('kpi');
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
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedCategory(key ?? '');
          }}
          defaultExpandAll
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
              {t('table.total')} {filteredData.length}
            </Typography.Text>
          </Typography.Title>
          <Button type="primary" icon={<DownloadOutlined />} onClick={() => void message.info(t('common.exportInProgress'))}>
            {t('common.export')}
          </Button>
        </div>

        {/* 数据表格 */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <DataTable<KPIIndicatorRow>
            tableId="kpi-management"
            columns={columns}
            dataSource={filteredData}
            loading={false}
            rowKey="kpiId"
            total={filteredData.length}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            scroll={{ x: 1200 }}
            selectable
            selectedRowKeys={selectedRowKeys}
            onSelectionChange={(keys, rows) => {
              handleSelectChange(keys, rows as KPIIndicatorRow[]);
            }}
            batchActions={batchActions}
            extraToolbarLeft={
              <Space size={8}>
                <Input
                  size="small"
                  placeholder={t('kpi.searchPlaceholder')}
                  prefix={<SearchOutlined />}
                  value={tableSearchValue}
                  onChange={(e) => setTableSearchValue(e.target.value)}
                  allowClear
                  style={{ width: 200 }}
                />
                <Select
                  size="small"
                  placeholder={t('kpi.productType')}
                  value={productType}
                  onChange={setProductType}
                  options={PRODUCT_TYPE_OPTIONS}
                  style={{ width: 100 }}
                  allowClear
                />
                <Select
                  size="small"
                  placeholder={t('kpi.level')}
                  value={indicatorLevel}
                  onChange={setIndicatorLevel}
                  options={LEVEL_OPTIONS}
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
            label={t('perf.functionSetName')}
            rules={[
              { required: true, message: t('perf.functionSetNameRequired') },
              { max: 200, message: t('perf.functionSetNameMax') },
            ]}
          >
            <Input
              placeholder={t('perf.functionSetNamePlaceholder')}
              maxLength={200}
              showCount
            />
          </Form.Item>
          <Form.Item
            name="networkType"
            label={t('perf.networkType')}
            rules={[{ required: true, message: t('common.selectRequired') }]}
          >
            <Select
              placeholder={t('perf.networkTypePlaceholder')}
              options={[
                { label: 'eNB', value: 'eNB' },
                { label: 'gNB', value: 'gNB' },
                { label: 'GSM', value: 'GSM' },
              ]}
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

      {/* Add Indicator Drawer */}
      <Drawer
        title={t('kpi.addIndicator')}
        open={addDrawerOpen}
        onClose={() => {
          setAddDrawerOpen(false);
          addIndicatorForm.resetFields();
        }}
        width={600}
        destroyOnClose
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => {
              setAddDrawerOpen(false);
              addIndicatorForm.resetFields();
            }}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" onClick={() => void handleAddIndicator()}>
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
                <Radio value="kpi">Customize KPI</Radio>
                <Radio value="counter">Customize Counter</Radio>
              </Radio.Group>
            </Form.Item>

            {/* 等级 */}
            <Form.Item name="indicatorLevel" label={t('kpi.level')} style={{ marginBottom: 12 }}>
              <Radio.Group>
                <Radio value="device">Device</Radio>
                <Radio value="plmn">PLMN</Radio>
              </Radio.Group>
            </Form.Item>

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

            {/* 所属功能集 - 只读显示 */}
            <Form.Item
              name="catagoryId"
              label={t('kpi.functionSet')}
              style={{ marginBottom: 12 }}
            >
              <Input disabled placeholder={selectedFunctionSetName || t('kpi.functionSetPlaceholder')} />
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
              <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
                {t('kpi.calcFormula')}
              </div>
              <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
                {t('kpi.calcFormulaDesc')}
              </Typography.Text>

              {/* 公式显示区域 */}
              <div
                style={{
                  minHeight: 80,
                  padding: '12px',
                  background: 'var(--color-bg-container)',
                  borderRadius: 4,
                  border: '1px solid var(--color-border)',
                  marginBottom: 12,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                }}
              >
                <Typography.Text type="secondary">
                  {t('kpi.calcFormulaPlaceholder')}
                </Typography.Text>
              </div>

              {/* 运算符按钮 */}
              <Space wrap size={8} style={{ marginBottom: 12 }}>
                {['+', '-', '*', '/', '(', ')', '0-9', 'Duration'].map((op) => (
                  <Button key={op} size="small" style={{ minWidth: 40 }}>
                    {op}
                  </Button>
                ))}
                <Button size="small" danger>
                  {t('common.clear')}
                </Button>
              </Space>

              {/* 指标选择提示 */}
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('kpi.selectIndicatorTip')}
              </Typography.Text>
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
    </>
  );
}
