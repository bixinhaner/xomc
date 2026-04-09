import { useState, useMemo } from 'react';
import {
  Button,
  message,
  Drawer,
  Radio,
  Form,
  Table,
  Tag,
  Alert,
  Space,
  Divider,
} from 'antd';
import {
  UploadOutlined,
  DownloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import { Upload } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

// 产品类型选项
const PRODUCT_TYPE_OPTIONS = [
  'PM-B4860', 'QAFA', 'QATA', 'QAFB', 'RTD',
  'BaiBNX', 'BaiBNQ', 'BSC', 'BTS',
];

// 导入方式
type ImportMode = 'filename' | 'all';

// 匹配结果行
interface MatchRow {
  id: string;
  deviceSn: string;
  deviceName: string;
  productType: string;
  fileName: string;
  matched: boolean;
}

interface ConfigFileRow extends Record<string, unknown> {
  id: string;
  deviceSn: string;
  deviceName: string;
  productType: string;
  configFile: string;
  fileSize: string;
  updateTime: string;
}

// Mock 数据
const mockData: ConfigFileRow[] = [
  { id: '1', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', productType: 'PM-B4860', configFile: 'ENB00001_config_20260302.xml', fileSize: '128 KB', updateTime: '2026-03-02 10:15:00' },
  { id: '2', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', productType: 'PM-B4860', configFile: 'ENB00002_config_20260302.xml', fileSize: '115 KB', updateTime: '2026-03-02 10:12:00' },
  { id: '3', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', productType: 'QAFA', configFile: 'ENB00003_config_20260301.xml', fileSize: '96 KB', updateTime: '2026-03-01 14:30:00' },
  { id: '4', deviceSn: 'GNB00001', deviceName: '北京5G基站01', productType: 'BaiBNX', configFile: 'GNB00001_config_20260302.xml', fileSize: '256 KB', updateTime: '2026-03-02 08:45:00' },
  { id: '5', deviceSn: 'GNB00002', deviceName: '上海5G基站01', productType: 'BaiBNX', configFile: 'GNB00002_config_20260228.xml', fileSize: '245 KB', updateTime: '2026-02-28 16:20:00' },
  { id: '6', deviceSn: 'ENB00004', deviceName: '广州天河基站01', productType: 'QATA', configFile: 'ENB00004_config_20260302.xml', fileSize: '102 KB', updateTime: '2026-03-02 09:30:00' },
  { id: '7', deviceSn: 'ENB00005', deviceName: '深圳南山基站01', productType: 'QAFB', configFile: 'ENB00005_config_20260301.xml', fileSize: '88 KB', updateTime: '2026-03-01 11:00:00' },
  { id: '8', deviceSn: 'GNB00003', deviceName: '广州5G基站01', productType: 'BaiBNQ', configFile: 'GNB00003_config_20260302.xml', fileSize: '198 KB', updateTime: '2026-03-02 07:50:00' },
  { id: '9', deviceSn: 'ENB00006', deviceName: '杭州西湖基站01', productType: 'RTD', configFile: 'ENB00006_config_20260227.xml', fileSize: '72 KB', updateTime: '2026-02-27 13:45:00' },
  { id: '10', deviceSn: 'ENB00007', deviceName: '南京鼓楼基站01', productType: 'PM-B4860', configFile: 'ENB00007_config_20260302.xml', fileSize: '135 KB', updateTime: '2026-03-02 10:45:00' },
  { id: '11', deviceSn: 'GNB00004', deviceName: '深圳5G基站01', productType: 'BaiBNX', configFile: 'GNB00004_config_20260301.xml', fileSize: '262 KB', updateTime: '2026-03-01 15:30:00' },
  { id: '12', deviceSn: 'ENB00008', deviceName: '成都武侯基站01', productType: 'QAFA', configFile: 'ENB00008_config_20260228.xml', fileSize: '91 KB', updateTime: '2026-02-28 09:10:00' },
];

export default function BackupSchedule() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  // 导入抽屉状态
  const [importVisible, setImportVisible] = useState(false);
  const [importMode, setImportMode] = useState<ImportMode>('filename');
  const [importFileList, setImportFileList] = useState<any[]>([]);
  const [matchResult, setMatchResult] = useState<MatchRow[]>([]);

  // ========== 筛选条件 ==========
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: '基站编码/名称', type: 'input', placeholder: '请输入基站编码或名称' },
    {
      name: 'productType',
      label: '产品类型',
      type: 'select',
      placeholder: '请选择',
      options: [
        { label: '全部', value: 'all' },
        ...PRODUCT_TYPE_OPTIONS.map((p) => ({ label: p, value: p })),
      ],
    },
  ], []);

  // ========== 过滤数据 ==========
  const filteredData = useMemo(() => {
    return mockData.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.deviceSn.toLowerCase().includes(keyword) &&
            !row.deviceName.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      if (filters.productType && filters.productType !== 'all') {
        if (row.productType !== filters.productType) return false;
      }
      return true;
    });
  }, [filters]);

  // ========== 匹配结果统计 ==========
  const matchStats = useMemo(() => {
    const matched = matchResult.filter((r) => r.matched).length;
    const unmatched = matchResult.filter((r) => !r.matched).length;
    return { matched, unmatched, total: matchResult.length };
  }, [matchResult]);

  // ========== 列定义（无操作列）==========
  const columns: DataTableColumn<ConfigFileRow>[] = useMemo(() => [
    { key: 'deviceSn', title: '基站编码', dataIndex: 'deviceSn', width: 120 },
    { key: 'deviceName', title: '基站名称', dataIndex: 'deviceName', width: 180, ellipsis: true },
    { key: 'productType', title: '产品类型', dataIndex: 'productType', width: 100 },
    {
      key: 'configFile',
      title: '配置文件',
      dataIndex: 'configFile',
      ellipsis: true,
      render: (val: string) => (
        <Button type="link" size="small" style={{ padding: 0 }} onClick={() => handleDownloadFile(val)}>
          {val}
        </Button>
      ),
    },
    { key: 'updateTime', title: '更新时间', dataIndex: 'updateTime', width: 160 },
  ], []);

  // ========== 操作处理 ==========
  const handleDownloadFile = (fileName: string) => {
    void message.success(`开始下载: ${fileName}`);
  };

  const handleBatchExport = (keys: React.Key[]) => {
    if (keys.length === 0) {
      void message.warning('请选择要导出的配置文件');
      return;
    }
    const selected = filteredData.filter((d) => keys.includes(d.id));
    void message.success(`开始导出 ${selected.length} 个配置文件`);
  };

  // 打开导入抽屉
  const handleBatchImport = () => {
    setImportMode('filename');
    setImportFileList([]);
    setMatchResult([]);
    setImportVisible(true);
  };

  // 模拟匹配：上传文件后根据导入方式匹配设备
  const handleMatchPreview = () => {
    if (importFileList.length === 0) {
      setMatchResult([]);
      return;
    }

    // 获取已选设备（如果有选择则用已选，否则用全部）
    const targetDevices = selectedRowKeys.length > 0
      ? mockData.filter((d) => selectedRowKeys.includes(d.id))
      : mockData;

    if (importMode === 'filename') {
      // 按文件名匹配：从文件名中提取设备SN前缀进行匹配
      const result: MatchRow[] = [];
      for (const device of targetDevices) {
        const matchedFile = importFileList.find((f) => {
          const fileName = (f.name || '').toUpperCase();
          return fileName.includes(device.deviceSn.toUpperCase());
        });
        result.push({
          id: device.id,
          deviceSn: device.deviceSn,
          deviceName: device.deviceName,
          productType: device.productType,
          fileName: matchedFile ? matchedFile.name : '',
          matched: !!matchedFile,
        });
      }
      setMatchResult(result);
    } else {
      // 所有设备匹配一个配置文件：用第一个文件匹配所有设备
      const firstFile = importFileList[0];
      const result: MatchRow[] = targetDevices.map((device) => ({
        id: device.id,
        deviceSn: device.deviceSn,
        deviceName: device.deviceName,
        productType: device.productType,
        fileName: firstFile?.name || '',
        matched: true,
      }));
      setMatchResult(result);
    }
  };

  // 确认导入
  const handleImportConfirm = () => {
    if (importFileList.length === 0) {
      void message.warning('请选择要导入的文件');
      return;
    }
    const matchedCount = matchResult.filter((r) => r.matched).length;
    void message.success(`已成功导入配置文件，匹配设备 ${matchedCount} 台`);
    setImportVisible(false);
    setImportFileList([]);
    setMatchResult([]);
    setSelectedRowKeys([]);
  };

  // ========== 批量操作（与设备列表风格一致）==========
  const batchActions = useMemo((): BatchAction[] => [
    {
      key: 'batch-import',
      label: '导入文件',
      icon: <UploadOutlined />,
      onClick: handleBatchImport,
    },
    {
      key: 'batch-export',
      label: '导出文件',
      icon: <DownloadOutlined />,
      onClick: handleBatchExport,
    },
  ], [filteredData, selectedRowKeys]);

  // ========== 页面头部按钮 ==========
  const headerExtra = null;

  return (
    <ListPageLayout title={t('nav.backup.schedule')} extra={headerExtra}>
      {/* 筛选条件 */}
      <FilterBar
        filterId="config-file-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />

      {/* 列表 */}
      <DataTable<ConfigFileRow>
        tableId="config-file-list"
        columns={columns}
        dataSource={filteredData}
        rowKey="id"
        selectable
        selectedRowKeys={selectedRowKeys}
        onSelectionChange={(keys) => setSelectedRowKeys(keys)}
        total={filteredData.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        batchActions={batchActions}
        showRowNumber
        rowNumberTitle="序号"
        scroll={{ x: 800 }}
      />

      {/* 导入抽屉 */}
      <Drawer
        title="导入配置文件"
        placement="right"
        width={680}
        open={importVisible}
        onClose={() => { setImportVisible(false); setImportFileList([]); setMatchResult([]); }}
        footer={
          <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button onClick={() => { setImportVisible(false); setImportFileList([]); setMatchResult([]); }}>取消</Button>
            {matchResult.length > 0 && matchStats.matched > 0 && (
              <Button
                type="primary"
                onClick={handleImportConfirm}
              >
                导入匹配成功的文件（{matchStats.matched} 台）
              </Button>
            )}
          </Space>
        }
      >
        <Form layout="vertical" size="small">
          {/* 文件导入方式 */}
          <Form.Item label="文件导入方式" required>
            <Radio.Group
              value={importMode}
              onChange={(e) => {
                setImportMode(e.target.value);
                setMatchResult([]);
              }}
            >
              <Radio value="filename">按导入文件名匹配</Radio>
              <Radio value="all">所有设备匹配一个配置文件</Radio>
            </Radio.Group>
            <div style={{ marginTop: 4, color: '#999', fontSize: 12 }}>
              {importMode === 'filename'
                ? '根据文件名中的设备SN自动匹配对应设备'
                : '将选中的配置文件应用到所有已选设备'}
            </div>
          </Form.Item>

          {/* 导入配置文件 - 使用导入组件 */}
          <Form.Item label="导入配置文件" required>
            <Upload.Dragger
              multiple={importMode === 'filename'}
              accept=".xml,.zip"
              maxCount={importMode === 'filename' ? undefined : 1}
              fileList={importFileList}
              onChange={({ fileList }) => {
                setImportFileList(fileList);
                setMatchResult([]);
              }}
              beforeUpload={() => false}
              style={{ marginBottom: 8 }}
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined />
              </p>
              <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
              <p className="ant-upload-hint">
                支持 .xml、.zip 格式
                {importMode === 'filename' ? '，可同时上传多个文件按文件名匹配' : '，只能上传一个配置文件'}
              </p>
            </Upload.Dragger>
          </Form.Item>

          {/* 匹配按钮 */}
          {importFileList.length > 0 && matchResult.length === 0 && (
            <Form.Item>
              <Button type="primary" onClick={handleMatchPreview}>
                开始匹配
              </Button>
            </Form.Item>
          )}

          {/* 匹配结果 */}
          {matchResult.length > 0 && (
            <>
              <Divider />
              <Form.Item label={
                <span>
                  匹配结果
                  <Tag color="success" style={{ marginLeft: 8 }}>
                    <CheckCircleOutlined /> 匹配成功 {matchStats.matched}
                  </Tag>
                  {matchStats.unmatched > 0 && (
                    <Tag color="error" style={{ marginLeft: 4 }}>
                      <CloseCircleOutlined /> 未匹配 {matchStats.unmatched}
                    </Tag>
                  )}
                </span>
              }>
                {matchStats.matched > 0 && (
                  <Alert
                    type="success"
                    showIcon
                    message={`共匹配成功 ${matchStats.matched} 台设备${matchStats.unmatched > 0 ? `，${matchStats.unmatched} 台设备未匹配到配置文件` : ''}`}
                    style={{ marginBottom: 12 }}
                  />
                )}
                {matchStats.matched === 0 && (
                  <Alert
                    type="warning"
                    showIcon
                    message="没有匹配到任何设备，请检查文件名是否包含设备SN"
                    style={{ marginBottom: 12 }}
                  />
                )}
                <Table
                  size="small"
                  dataSource={matchResult}
                  rowKey="id"
                  pagination={matchResult.length > 10 ? { pageSize: 10 } : false}
                  scroll={{ y: 300 }}
                  columns={[
                    {
                      title: '基站编码',
                      dataIndex: 'deviceSn',
                      width: 110,
                    },
                    {
                      title: '基站名称',
                      dataIndex: 'deviceName',
                      ellipsis: true,
                    },
                    {
                      title: '产品类型',
                      dataIndex: 'productType',
                      width: 90,
                    },
                    {
                      title: '匹配文件',
                      dataIndex: 'fileName',
                      width: 200,
                      ellipsis: true,
                      render: (val: string) => val || '-',
                    },
                    {
                      title: '匹配状态',
                      width: 90,
                      dataIndex: 'matched',
                      render: (val: boolean) => val
                        ? <Tag color="success" icon={<CheckCircleOutlined />}>成功</Tag>
                        : <Tag color="error" icon={<CloseCircleOutlined />}>未匹配</Tag>,
                    },
                  ]}
                />
              </Form.Item>
            </>
          )}
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
