import { useState, useMemo } from 'react';
import {
  Tag,
  message,
  Progress,
  Checkbox,
  Alert,
  Space,
  Modal,
  Input,
  Button,
  Tabs,
  Select,
  Statistic,
} from 'antd';
import { PlayCircleOutlined, WarningOutlined, PlusOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

// 升级类型枚举
type UpgradeType = 'immediate' | 'scheduled' | 'manual';
// 升级结果枚举
type UpgradeResult = 'success' | 'failed' | 'partial' | 'running' | 'pending';

interface UpgradePlanRow extends Record<string, unknown> {
  id: string;
  deviceSn: string;
  deviceName: string;
  deviceGroup: string;
  sourceVersion: string;
  targetVersion: string;
  upgradeType: UpgradeType;
  productType: string;
  keepConfig: boolean;
  progress: number;
  result: UpgradeResult;
  failureReason: string;
  operator: string;
  operateTime: string;
  startTime: string;
  endTime: string;
}

// 升级类型映射
const UPGRADE_TYPE_MAP: Record<UpgradeType, { color: string; text: string }> = {
  immediate: { color: 'green', text: '立即升级' },
  scheduled: { color: 'blue', text: '定时升级' },
  manual: { color: 'orange', text: '手动升级' },
};

// 升级结果映射
const UPGRADE_RESULT_MAP: Record<UpgradeResult, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
  partial: { color: 'warning', text: '部分成功' },
  running: { color: 'processing', text: '升级中' },
  pending: { color: 'default', text: '等待中' },
};

// Mock 数据
const mockData: UpgradePlanRow[] = [
  { id: '1', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:15:00' },
  { id: '2', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', deviceGroup: '北京移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 09:50:00', startTime: '2026-03-25 10:00:00', endTime: '2026-03-25 10:12:00' },
  { id: '3', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', deviceGroup: '上海移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFA', keepConfig: true, progress: 75, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '' },
  { id: '4', deviceSn: 'GNB00001', deviceName: '北京5G基站01', deviceGroup: '北京移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNX', keepConfig: false, progress: 100, result: 'failed', failureReason: '固件校验失败', operator: 'lisi', operateTime: '2026-03-25 09:25:00', startTime: '2026-03-25 09:30:00', endTime: '2026-03-25 09:45:00' },
  { id: '5', deviceSn: 'GNB00002', deviceName: '上海5G基站01', deviceGroup: '上海移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNX', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:00:00', startTime: '', endTime: '' },
  { id: '6', deviceSn: 'ENB00004', deviceName: '广州天河基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'partial', failureReason: '部分配置恢复失败', operator: 'wangwu', operateTime: '2026-03-25 07:55:00', startTime: '2026-03-25 08:00:00', endTime: '2026-03-25 08:30:00' },
  { id: '7', deviceSn: 'ENB00005', deviceName: '深圳南山基站01', deviceGroup: '广东移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 50, result: 'running', failureReason: '', operator: 'zhangsan', operateTime: '2026-03-25 11:25:00', startTime: '2026-03-25 11:30:00', endTime: '' },
  { id: '8', deviceSn: 'GNB00003', deviceName: '广州5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'immediate', productType: 'BaiBNQ', keepConfig: false, progress: 100, result: 'success', failureReason: '', operator: 'admin', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:20:00' },
  { id: '9', deviceSn: 'ENB00006', deviceName: '杭州西湖基站01', deviceGroup: '浙江移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'manual', productType: 'RTD', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '' },
  { id: '10', deviceSn: 'ENB00007', deviceName: '南京鼓楼基站01', deviceGroup: '江苏移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'PM-B4860', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 10:25:00', startTime: '2026-03-25 10:30:00', endTime: '2026-03-25 10:45:00' },
  { id: '11', deviceSn: 'GNB00004', deviceName: '深圳5G基站01', deviceGroup: '广东移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'scheduled', productType: 'BaiBNX', keepConfig: true, progress: 30, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 11:55:00', startTime: '2026-03-25 12:00:00', endTime: '' },
  { id: '12', deviceSn: 'ENB00008', deviceName: '成都武侯基站01', deviceGroup: '四川移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QAFA', keepConfig: false, progress: 100, result: 'failed', failureReason: '网络连接超时', operator: 'zhangsan', operateTime: '2026-03-25 08:25:00', startTime: '2026-03-25 08:30:00', endTime: '2026-03-25 08:50:00' },
  { id: '13', deviceSn: 'ENB00009', deviceName: '武汉洪山基站01', deviceGroup: '湖北移动', sourceVersion: 'V1.1.5', targetVersion: 'V1.3.0', upgradeType: 'immediate', productType: 'QATA', keepConfig: true, progress: 100, result: 'success', failureReason: '', operator: 'lisi', operateTime: '2026-03-25 08:55:00', startTime: '2026-03-25 09:00:00', endTime: '2026-03-25 09:18:00' },
  { id: '14', deviceSn: 'GNB00005', deviceName: '成都5G基站01', deviceGroup: '四川移动', sourceVersion: 'V2.0.0', targetVersion: 'V2.1.0', upgradeType: 'manual', productType: 'BaiBNQ', keepConfig: true, progress: 0, result: 'pending', failureReason: '', operator: 'wangwu', operateTime: '2026-03-25 12:00:00', startTime: '', endTime: '' },
  { id: '15', deviceSn: 'ENB00010', deviceName: '西安雁塔基站01', deviceGroup: '陕西移动', sourceVersion: 'V1.2.0', targetVersion: 'V1.3.0', upgradeType: 'scheduled', productType: 'QAFB', keepConfig: true, progress: 60, result: 'running', failureReason: '', operator: 'admin', operateTime: '2026-03-25 10:55:00', startTime: '2026-03-25 11:00:00', endTime: '' },
];

export default function UpgradePlan() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [selectedRows, setSelectedRows] = useState<UpgradePlanRow[]>([]);
  // 批量输入相关状态
  const [batchInputVisible, setBatchInputVisible] = useState(false);
  const [batchInputTab, setBatchInputTab] = useState<'sn' | 'type'>('sn');
  const [batchInputValue, setBatchInputValue] = useState('');
  const [selectedProductType, setSelectedProductType] = useState<string | undefined>(undefined);
  const [batchInputPreview, setBatchInputPreview] = useState<{
    matched: UpgradePlanRow[];
    notFound: string[];
    mixedTypes: string[];
  }>({ matched: [], notFound: [], mixedTypes: [] });

  // 计算已选设备的产品类型
  const selectedTypes = useMemo(() => {
    const types = new Set(selectedRows.map((row) => row.productType));
    return Array.from(types);
  }, [selectedRows]);

  // 是否选择了不同类型的设备
  const hasMixedTypes = selectedTypes.length > 1;

  // 解析批量输入的设备SN
  const parseBatchInput = (input: string): string[] => {
    // 支持换行、逗号、分号、空格分隔
    return input
      .split(/[\n,;,\s]+/)
      .map((s) => s.trim().toUpperCase())
      .filter((s) => s.length > 0);
  };

  // 批量输入预览
  const handleBatchInputPreview = () => {
    const sns = parseBatchInput(batchInputValue);
    if (sns.length === 0) {
      setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
      return;
    }

    const matched: UpgradePlanRow[] = [];
    const notFound: string[] = [];
    const foundSns = new Set<string>();

    // 在过滤后的数据中查找匹配的设备
    for (const sn of sns) {
      const row = filteredData.find(
        (r) => r.deviceSn.toUpperCase() === sn
      );
      if (row) {
        matched.push(row);
        foundSns.add(sn);
      } else {
        notFound.push(sn);
      }
    }

    // 检查产品类型是否一致
    const typeSet = new Set(matched.map((r) => r.productType));
    const mixedTypes = Array.from(typeSet);

    setBatchInputPreview({ matched, notFound, mixedTypes });
  };

  // 确认批量输入
  const handleBatchInputConfirm = () => {
    const { matched, mixedTypes } = batchInputPreview;

    if (matched.length === 0) {
      void message.warning('没有匹配到任何设备');
      return;
    }

    if (mixedTypes.length > 1) {
      void message.error('批量升级只能选择相同产品类型的设备，请重新输入');
      return;
    }

    // 设置选中状态
    const keys = matched.map((r) => r.id);
    setSelectedKeys(keys);
    setSelectedRows(matched);
    setBatchInputVisible(false);
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });

    void message.success(`已匹配 ${matched.length} 个设备，产品类型: ${mixedTypes[0]}`);
  };

  // 打开批量输入弹窗
  const handleOpenBatchInput = () => {
    setBatchInputValue('');
    setBatchInputPreview({ matched: [], notFound: [], mixedTypes: [] });
    setBatchInputVisible(true);
  };

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '基站编码/名称', type: 'input', placeholder: '请输入基站编码或名称' },
    {
      name: 'productType',
      label: '产品类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        // eNB 产品类型
        { label: 'PM-B4860', value: 'PM-B4860' },
        { label: 'QAFA', value: 'QAFA' },
        { label: 'QATA', value: 'QATA' },
        { label: 'QAFB', value: 'QAFB' },
        { label: 'RTD', value: 'RTD' },
        // gNB 产品类型
        { label: 'BaiBNX', value: 'BaiBNX' },
        { label: 'BaiBNQ', value: 'BaiBNQ' },
        // GSM 产品类型
        { label: 'BSC', value: 'BSC' },
        { label: 'BTS', value: 'BTS' },
      ],
    },
    {
      name: 'sourceVersion',
      label: '初始版本',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: 'V1.1.5', value: 'V1.1.5' },
        { label: 'V1.2.0', value: 'V1.2.0' },
        { label: 'V2.0.0', value: 'V2.0.0' },
      ],
    },
    {
      name: 'targetVersion',
      label: '升级版本',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: 'V1.3.0', value: 'V1.3.0' },
        { label: 'V2.1.0', value: 'V2.1.0' },
      ],
    },
    {
      name: 'deviceGroup',
      label: '设备组',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '北京移动', value: '北京移动' },
        { label: '上海移动', value: '上海移动' },
        { label: '广东移动', value: '广东移动' },
        { label: '浙江移动', value: '浙江移动' },
        { label: '江苏移动', value: '江苏移动' },
        { label: '四川移动', value: '四川移动' },
        { label: '湖北移动', value: '湖北移动' },
        { label: '陕西移动', value: '陕西移动' },
      ],
    },
    {
      name: 'upgradeType',
      label: '升级类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '立即升级', value: 'immediate' },
        { label: '定时升级', value: 'scheduled' },
        { label: '手动升级', value: 'manual' },
      ],
    },
    {
      name: 'result',
      label: '结果',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        { label: '成功', value: 'success' },
        { label: '失败', value: 'failed' },
        { label: '部分成功', value: 'partial' },
        { label: '升级中', value: 'running' },
        { label: '等待中', value: 'pending' },
      ],
    },
  ], []);

  // 过滤数据
  const filteredData = useMemo(() => {
    return mockData.filter((row) => {
      // 关键字搜索（基站编码或名称）
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.deviceSn.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      // 产品类型
      if (filters.productType && filters.productType !== 'all') {
        if (row.productType !== filters.productType) return false;
      }
      // 初始版本
      if (filters.sourceVersion && filters.sourceVersion !== 'all') {
        if (row.sourceVersion !== filters.sourceVersion) return false;
      }
      // 升级版本
      if (filters.targetVersion && filters.targetVersion !== 'all') {
        if (row.targetVersion !== filters.targetVersion) return false;
      }
      // 设备组
      if (filters.deviceGroup && filters.deviceGroup !== 'all') {
        if (row.deviceGroup !== filters.deviceGroup) return false;
      }
      // 升级类型
      if (filters.upgradeType && filters.upgradeType !== 'all') {
        if (row.upgradeType !== filters.upgradeType) return false;
      }
      // 结果
      if (filters.result && filters.result !== 'all') {
        if (row.result !== filters.result) return false;
      }
      return true;
    });
  }, [filters]);

  // 批量升级
  const handleBatchUpgrade = (keys: React.Key[]) => {
    if (keys.length === 0) {
      void message.warning('请先选择要升级的设备');
      return;
    }
    // 校验产品类型是否一致
    if (hasMixedTypes) {
      void message.error('批量升级只能选择相同产品类型的设备');
      return;
    }
    void message.success(`已开始对 ${keys.length} 个 ${selectedTypes[0] ?? ''} 设备执行批量升级`);
    setSelectedKeys([]);
    setSelectedRows([]);
  };

  // 批量操作配置
  const batchActions: BatchAction[] = useMemo(() => [
    {
      key: 'batch-upgrade',
      label: '批量升级',
      icon: <PlayCircleOutlined />,
      onClick: handleBatchUpgrade,
    },
  ], [hasMixedTypes, selectedTypes]);

  // 选择变化处理
  const handleSelectionChange = (keys: React.Key[], rows: UpgradePlanRow[]) => {
    setSelectedKeys(keys);
    setSelectedRows(rows);
  };

  // 工具栏左侧内容 - 显示已选类型信息
  const extraToolbarLeft = useMemo(() => {
    if (selectedKeys.length === 0) return null;

    return hasMixedTypes ? (
      <Alert
        type="warning"
        showIcon
        icon={<WarningOutlined />}
        message={
          <Space>
            <span>已选择 {selectedKeys.length} 台设备</span>
            <Tag color="warning">类型不一致: {selectedTypes.join(', ')}</Tag>
            <span style={{ color: '#faad14' }}>批量升级需选择相同产品类型</span>
          </Space>
        }
        style={{ padding: '4px 12px' }}
      />
    ) : (
      <Space>
        <span>已选择 {selectedKeys.length} 台设备</span>
        <Tag color="blue">类型: {selectedTypes[0]}</Tag>
      </Space>
    );
  }, [selectedKeys.length, hasMixedTypes, selectedTypes]);

  // 工具栏右侧内容 - 批量输入按钮（在实时刷新按钮左侧）
  const extraToolbarRight = useMemo(() => (
    <Button
      size="small"
      icon={<PlusOutlined />}
      onClick={handleOpenBatchInput}
    >
      批量输入
    </Button>
  ), []);

  const columns: DataTableColumn<UpgradePlanRow>[] = useMemo(() => [
    { key: 'deviceSn', title: '基站编码', dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: '基站名称', dataIndex: 'deviceName', ellipsis: true },
    { key: 'deviceGroup', title: '设备组', dataIndex: 'deviceGroup', width: 100 },
    { key: 'sourceVersion', title: '初始版本', dataIndex: 'sourceVersion', width: 100 },
    { key: 'targetVersion', title: '升级版本', dataIndex: 'targetVersion', width: 100 },
    {
      key: 'upgradeType',
      title: '升级类型',
      dataIndex: 'upgradeType',
      width: 100,
      render: (val: UpgradeType) => {
        const cfg = UPGRADE_TYPE_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    {
      key: 'keepConfig',
      title: '保留配置',
      dataIndex: 'keepConfig',
      width: 90,
      align: 'center',
      render: (val: boolean) => <Checkbox checked={val} />,
    },
    {
      key: 'progress',
      title: '升级进度',
      dataIndex: 'progress',
      width: 120,
      render: (val: number) => <Progress percent={val} size="small" status={val === 100 ? 'success' : 'active'} />,
    },
    {
      key: 'result',
      title: '结果',
      dataIndex: 'result',
      width: 100,
      render: (val: UpgradeResult) => {
        const cfg = UPGRADE_RESULT_MAP[val] ?? { color: 'default', text: val };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'failureReason', title: '失败原因', dataIndex: 'failureReason', width: 150, ellipsis: true },
    { key: 'operator', title: '操作人', dataIndex: 'operator', width: 100 },
    { key: 'operateTime', title: '操作时间', dataIndex: 'operateTime', width: 160 },
    { key: 'startTime', title: '开始时间', dataIndex: 'startTime', width: 160 },
    { key: 'endTime', title: '结束时间', dataIndex: 'endTime', width: 160 },
  ], []);

  return (
    <ListPageLayout title={t('nav.software.upgradePlan')}>
      <FilterBar
        filterId="upgrade-plan-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable<UpgradePlanRow>
        tableId="upgrade-plan-list"
        columns={columns}
        dataSource={filteredData}
        rowKey="id"
        total={filteredData.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        scroll={{ x: 1900 }}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={handleSelectionChange}
        batchActions={batchActions}
        extraToolbarLeft={extraToolbarLeft}
        extraToolbarRight={extraToolbarRight}
      />

      {/* 批量输入弹窗 */}
      <Modal
        title="批量输入设备SN"
        open={batchInputVisible}
        onCancel={() => setBatchInputVisible(false)}
        onOk={handleBatchInputConfirm}
        okText="确认选择"
        cancelText="取消"
        width={600}
        okButtonProps={{
          disabled: batchInputPreview.matched.length === 0 || batchInputPreview.mixedTypes.length > 1,
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Input.TextArea
            placeholder="请输入设备SN，支持换行、逗号、分号、空格分隔&#10;例如：&#10;ENB00001&#10;ENB00002, ENB00003; GNB00001"
            rows={6}
            value={batchInputValue}
            onChange={(e) => setBatchInputValue(e.target.value)}
            onBlur={handleBatchInputPreview}
          />
        </div>

        {/* 预览结果 */}
        {batchInputPreview.matched.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Alert
              type={batchInputPreview.mixedTypes.length > 1 ? 'warning' : 'success'}
              showIcon
              icon={batchInputPreview.mixedTypes.length > 1 ? <WarningOutlined /> : undefined}
              message={
                <Space direction="vertical" size="small">
                  <span>
                    匹配到 <strong>{batchInputPreview.matched.length}</strong> 个设备
                    {batchInputPreview.mixedTypes.length === 1 && (
                      <Tag color="blue" style={{ marginLeft: 8 }}>产品类型: {batchInputPreview.mixedTypes[0]}</Tag>
                    )}
                  </span>
                  {batchInputPreview.mixedTypes.length > 1 && (
                    <span style={{ color: '#faad14' }}>
                      <WarningOutlined style={{ marginRight: 4 }} />
                      检测到多种产品类型: {batchInputPreview.mixedTypes.join(', ')}，批量升级只能选择相同产品类型
                    </span>
                  )}
                </Space>
              }
              style={{ marginBottom: 8 }}
            />
          </div>
        )}

        {/* 未找到的设备 */}
        {batchInputPreview.notFound.length > 0 && (
          <Alert
            type="warning"
            showIcon
            message={
              <div>
                <div>以下 {batchInputPreview.notFound.length} 个设备SN未找到:</div>
                <div style={{ maxHeight: 80, overflow: 'auto', marginTop: 4 }}>
                  {batchInputPreview.notFound.map((sn) => (
                    <Tag key={sn} style={{ margin: '2px' }}>{sn}</Tag>
                  ))}
                </div>
              </div>
            }
            style={{ marginBottom: 8 }}
          />
        )}
      </Modal>
    </ListPageLayout>
  );
}
