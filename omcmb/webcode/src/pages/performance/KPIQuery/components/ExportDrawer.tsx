import React, { useState, useCallback, useMemo } from 'react';
import {
  Drawer,
  Tabs,
  Table,
  Button,
  Input,
  Space,
  Progress,
  Tag,
  App,
  Typography,
  Tooltip,
  Card,
  Divider,
} from 'antd';
import {
  SearchOutlined,
  DownloadOutlined,
  DeleteOutlined,
  StopOutlined,
  ReloadOutlined,
  PlusOutlined,
  FileExcelOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

const { Text } = Typography;

// 导出任务状态
type TaskStatus = 'waiting' | 'processing' | 'completed' | 'failed';

// 导出任务项
interface ExportTaskItem {
  id: string;
  fileName: string;
  progress: number;
  status: TaskStatus;
  createTime: string;
  fileSize?: string;
}

// Mock 数据 - 手动生成任务
const MOCK_MANUAL_TASKS: ExportTaskItem[] = [
  { id: '1', fileName: 'eNB基础KPI_20260403_103045.xlsx', progress: 100, status: 'completed', createTime: '2026-04-03 10:30:45', fileSize: '2.3 MB' },
  { id: '2', fileName: 'eNB基础KPI_20260403_104512.xlsx', progress: 75, status: 'processing', createTime: '2026-04-03 10:45:12' },
  { id: '3', fileName: 'eNB基础KPI_20260403_102030.xlsx', progress: 0, status: 'waiting', createTime: '2026-04-03 10:20:30' },
  { id: '4', fileName: 'gNB性能指标_20260403_095512.xlsx', progress: 45, status: 'failed', createTime: '2026-04-03 09:55:12' },
  { id: '5', fileName: '小区吞吐量_20260403_093000.xlsx', progress: 100, status: 'completed', createTime: '2026-04-03 09:30:00', fileSize: '1.8 MB' },
];

// Mock 数据 - 定时报表任务
const MOCK_TIMED_TASKS: ExportTaskItem[] = [
  { id: '6', fileName: '定时报表_20260403_080000.xlsx', progress: 100, status: 'completed', createTime: '2026-04-03 08:00:00', fileSize: '3.1 MB' },
  { id: '7', fileName: '定时报表_20260402_080000.xlsx', progress: 100, status: 'completed', createTime: '2026-04-02 08:00:00', fileSize: '2.9 MB' },
  { id: '8', fileName: '定时报表_20260401_080000.xlsx', progress: 100, status: 'completed', createTime: '2026-04-01 08:00:00', fileSize: '2.7 MB' },
];

interface ExportDrawerProps {
  open: boolean;
  onClose: () => void;
  templateName: string;
  reportPeriod: string;
}

const ExportDrawer: React.FC<ExportDrawerProps> = ({
  open,
  onClose,
  templateName,
  reportPeriod,
}) => {
  const t = useT();
  const token = useThemeToken();
  const { message, modal } = App.useApp();

  const [activeTab, setActiveTab] = useState<'manual' | 'timed'>('manual');
  const [searchText, setSearchText] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [manualTasks, setManualTasks] = useState<ExportTaskItem[]>(MOCK_MANUAL_TASKS);
  const [timedTasks, setTimedTasks] = useState<ExportTaskItem[]>(MOCK_TIMED_TASKS);
  const [generating, setGenerating] = useState(false);

  // 获取粒度显示文本
  const getGranularityText = (value: string) => {
    const map: Record<string, string> = {
      '15': '15Min',
      '60': '60Min',
      '1440': '24Hour',
      '10080': 'Week',
      '43200': 'Month',
    };
    return map[value] || value;
  };

  // 获取当前显示的任务列表
  const currentTasks = useMemo(() => {
    const tasks = activeTab === 'manual' ? manualTasks : timedTasks;
    if (!searchText.trim()) return tasks;
    return tasks.filter((task) =>
      task.fileName.toLowerCase().includes(searchText.toLowerCase())
    );
  }, [activeTab, manualTasks, timedTasks, searchText]);

  // 获取状态标签配置
  const getStatusConfig = (status: TaskStatus) => {
    const configs: Record<TaskStatus, { color: string; text: string }> = {
      waiting: { color: 'default', text: t('perf.export.statusWaiting') },
      processing: { color: 'processing', text: t('perf.export.statusProcessing') },
      completed: { color: 'success', text: t('perf.export.statusCompleted') },
      failed: { color: 'error', text: t('perf.export.statusFailed') },
    };
    return configs[status];
  };

  // 获取进度条状态
  const getProgressStatus = (status: TaskStatus): 'exception' | 'normal' | 'active' | 'success' => {
    if (status === 'failed') return 'exception';
    if (status === 'completed') return 'success';
    if (status === 'processing') return 'active';
    return 'normal';
  };

  // 生成报表
  const handleGenerate = useCallback(() => {
    setGenerating(true);
    setTimeout(() => {
      const newTask: ExportTaskItem = {
        id: `new-${Date.now()}`,
        fileName: `${templateName}_${new Date().toISOString().slice(0, 19).replace(/[-:T]/g, '')}.xlsx`,
        progress: 0,
        status: 'waiting',
        createTime: new Date().toLocaleString('zh-CN'),
      };
      setManualTasks((prev) => [newTask, ...prev]);
      setGenerating(false);
      void message.success(t('perf.export.generateSuccess'));
    }, 1000);
  }, [templateName, message, t]);

  // 下载文件
  const handleDownload = useCallback((record: ExportTaskItem) => {
    void message.success(t('perf.export.downloadStart', { name: record.fileName }));
  }, [message, t]);

  // 终止任务
  const handleTerminate = useCallback((record: ExportTaskItem) => {
    modal.confirm({
      title: t('perf.export.terminateConfirm'),
      content: t('perf.export.terminateConfirmMsg'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => {
        setManualTasks((prev) =>
          prev.map((task) =>
            task.id === record.id ? { ...task, status: 'failed' as TaskStatus, progress: 0 } : task
          )
        );
        void message.success(t('common.success'));
      },
    });
  }, [modal, message, t]);

  // 删除文件
  const handleDelete = useCallback((record: ExportTaskItem) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('perf.export.deleteConfirmMsg'),
      okText: t('common.confirm'),
      okType: 'danger',
      cancelText: t('common.cancel'),
      onOk: () => {
        if (activeTab === 'manual') {
          setManualTasks((prev) => prev.filter((task) => task.id !== record.id));
        } else {
          setTimedTasks((prev) => prev.filter((task) => task.id !== record.id));
        }
        setSelectedRowKeys((prev) => prev.filter((key) => key !== record.id));
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [activeTab, modal, message, t]);

  // 批量删除
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('perf.export.selectAtLeastOne'));
      return;
    }

    modal.confirm({
      title: t('perf.export.batchDeleteConfirm'),
      content: t('perf.export.batchDeleteConfirmMsg', { count: selectedRowKeys.length }),
      okText: t('common.confirm'),
      okType: 'danger',
      cancelText: t('common.cancel'),
      onOk: () => {
        if (activeTab === 'manual') {
          setManualTasks((prev) => prev.filter((task) => !selectedRowKeys.includes(task.id)));
        } else {
          setTimedTasks((prev) => prev.filter((task) => !selectedRowKeys.includes(task.id)));
        }
        setSelectedRowKeys([]);
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [selectedRowKeys, activeTab, modal, message, t]);

  // 刷新列表
  const handleRefresh = useCallback(() => {
    void message.success(t('common.refreshSuccess'));
  }, [message, t]);

  // 表格列定义
  const columns: ColumnsType<ExportTaskItem> = useMemo(() => [
    {
      title: t('perf.export.fileName'),
      dataIndex: 'fileName',
      key: 'fileName',
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <Space>
            <FileExcelOutlined style={{ color: '#52c41a' }} />
            <span>{text}</span>
          </Space>
        </Tooltip>
      ),
    },
    {
      title: t('perf.export.fileSize'),
      dataIndex: 'fileSize',
      key: 'fileSize',
      width: 100,
      render: (size?: string) => size || '-',
    },
    {
      title: t('perf.export.progress'),
      dataIndex: 'progress',
      key: 'progress',
      width: 180,
      render: (progress: number, record: ExportTaskItem) => (
        <Progress
          percent={progress}
          status={getProgressStatus(record.status)}
          size="small"
          style={{ width: 150 }}
        />
      ),
    },
    {
      title: t('perf.export.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: TaskStatus) => {
        const config = getStatusConfig(status);
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('perf.export.createTime'),
      dataIndex: 'createTime',
      key: 'createTime',
      width: 170,
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: ExportTaskItem) => (
        <Space size={4}>
          {record.status === 'completed' && (
            <Tooltip title={t('common.download')}>
              <Button
                type="link"
                size="small"
                icon={<DownloadOutlined />}
                onClick={() => handleDownload(record)}
                style={{ padding: '0 4px' }}
              />
            </Tooltip>
          )}
          {record.status === 'processing' && (
            <Tooltip title={t('perf.export.terminate')}>
              <Button
                type="link"
                size="small"
                danger
                icon={<StopOutlined />}
                onClick={() => handleTerminate(record)}
                style={{ padding: '0 4px' }}
              />
            </Tooltip>
          )}
          {record.status !== 'processing' && (
            <Tooltip title={t('common.delete')}>
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleDelete(record)}
                style={{ padding: '0 4px' }}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ], [t, handleDownload, handleTerminate, handleDelete]);

  // Tab 切换时清空选择
  const handleTabChange = (key: string) => {
    setActiveTab(key as 'manual' | 'timed');
    setSelectedRowKeys([]);
    setSearchText('');
  };

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => setSelectedRowKeys(newSelectedRowKeys),
    getCheckboxProps: (record: ExportTaskItem) => ({
      disabled: record.status === 'processing',
    }),
  };

  // 关闭时重置状态
  const handleClose = () => {
    setSearchText('');
    setSelectedRowKeys([]);
    onClose();
  };

  // 渲染头部信息卡片
  const renderHeaderCard = (showGenerateButton: boolean) => (
    <Card
      size="small"
      style={{
        marginBottom: 16,
        background: token.colorPrimaryBg,
        borderColor: token.colorPrimaryBorder,
      }}
      styles={{ body: { padding: '12px 16px' } }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Space split={<Divider type="vertical" />} size="small">
          <Space size={4}>
            <Text type="secondary">{t('perf.export.currentTemplate')}:</Text>
            <Text strong>{templateName}</Text>
          </Space>
          <Space size={4}>
            <Text type="secondary">{t('perf.export.queryPeriod')}:</Text>
            <Text strong>{getGranularityText(reportPeriod)}</Text>
          </Space>
        </Space>
        {showGenerateButton && (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            loading={generating}
            onClick={handleGenerate}
          >
            {t('perf.export.generate')}
          </Button>
        )}
      </div>
    </Card>
  );

  // 渲染工具栏
  const renderToolbar = () => (
    <div style={{
      marginBottom: 16,
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
    }}>
      <Input
        placeholder={t('perf.export.searchPlaceholder')}
        prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        style={{ width: 280 }}
        allowClear
      />
      {selectedRowKeys.length > 0 && (
        <Space>
          <Text type="secondary">
            {t('perf.export.selectedCount', { count: selectedRowKeys.length })}
          </Text>
          <Button danger size="small" onClick={handleBatchDelete}>
            {t('common.batchDelete')}
          </Button>
        </Space>
      )}
    </div>
  );

  return (
    <Drawer
      title={
        <Space>
          <FileExcelOutlined />
          {t('perf.export.title')}
        </Space>
      }
      open={open}
      onClose={handleClose}
      width={960}
      styles={{
        body: { padding: 0 },
      }}
    >
      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        style={{ height: '100%' }}
        tabBarExtraContent={
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            size="small"
          >
            {t('common.refresh')}
          </Button>
        }
        items={[
          {
            key: 'manual',
            label: (
              <Space>
                <PlusOutlined />
                {t('perf.export.manualGenerate')}
              </Space>
            ),
            children: (
              <div style={{ padding: 16 }}>
                {renderHeaderCard(true)}
                {renderToolbar()}
                <Table
                  rowKey="id"
                  columns={columns}
                  dataSource={currentTasks}
                  rowSelection={rowSelection}
                  pagination={{
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => t('common.totalItems', { count: total.toString() }),
                    defaultPageSize: 10,
                  }}
                  size="middle"
                  scroll={{ x: 900 }}
                />
              </div>
            ),
          },
          {
            key: 'timed',
            label: (
              <Space>
                <ClockCircleOutlined />
                {t('perf.export.timedReport')}
              </Space>
            ),
            children: (
              <div style={{ padding: 16 }}>
                {renderHeaderCard(false)}
                {renderToolbar()}
                <Table
                  rowKey="id"
                  columns={columns}
                  dataSource={currentTasks}
                  rowSelection={rowSelection}
                  pagination={{
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => t('common.totalItems', { count: total.toString() }),
                    defaultPageSize: 10,
                  }}
                  size="middle"
                  scroll={{ x: 900 }}
                />
              </div>
            ),
          },
        ]}
      />
    </Drawer>
  );
};

export default ExportDrawer;
