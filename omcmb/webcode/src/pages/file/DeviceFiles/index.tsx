import { useState, useMemo } from 'react';
import { Button, Tabs, Tag, Space, Switch, message } from 'antd';
import { DownloadOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

interface DeviceConfigFile {
  id: string;
  fileName: string;
  deviceSn: string;
  deviceName: string;
  fileSize: number;
  version: string;
  source: 'device' | 'user' | 'nms';
  status: 'active' | 'backup' | 'deprecated';
  uploadTime: string;
}

interface LogUploadConfig {
  id: string;
  deviceSn: string;
  deviceName: string;
  logType: string;
  enabled: boolean;
  interval: number;
  retentionDays: number;
  lastUpload?: string;
}

interface DeviceLogFile {
  id: string;
  fileName: string;
  deviceSn: string;
  deviceName: string;
  fileSize: number;
  logType: string;
  collectTime: string;
  uploadTime: string;
}

const mockDeviceConfigFiles: DeviceConfigFile[] = [
  { id: 'dcf-001', fileName: 'ENB00001_running_config.xml', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 131072, version: 'v1.2.3', source: 'device', status: 'active', uploadTime: '2024-06-01T08:00:00.000Z' },
  { id: 'dcf-002', fileName: 'ENB00001_backup_config.xml', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 126976, version: 'v1.2.2', source: 'device', status: 'backup', uploadTime: '2024-05-25T10:00:00.000Z' },
  { id: 'dcf-003', fileName: 'GNB00001_user_config.xml', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', fileSize: 262144, version: 'v2.0.1', source: 'user', status: 'active', uploadTime: '2024-06-02T09:30:00.000Z' },
  { id: 'dcf-004', fileName: 'ENB00002_nms_config.xml', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', fileSize: 98304, version: 'v1.1.0', source: 'nms', status: 'active', uploadTime: '2024-06-01T14:00:00.000Z' },
];

const mockLogUploadConfigs: LogUploadConfig[] = [
  { id: 'luc-001', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', logType: '运行日志', enabled: true, interval: 24, retentionDays: 30, lastUpload: '2024-06-01T00:00:00.000Z' },
  { id: 'luc-002', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', logType: '告警日志', enabled: true, interval: 1, retentionDays: 90, lastUpload: '2024-06-01T23:00:00.000Z' },
  { id: 'luc-003', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', logType: '运行日志', enabled: false, interval: 24, retentionDays: 30, lastUpload: undefined },
];

const mockDeviceLogFiles: DeviceLogFile[] = [
  { id: 'dlf-001', fileName: 'ENB00001_runtime_20240601.log', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 1024 * 12, logType: '运行日志', collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T01:00:00.000Z' },
  { id: 'dlf-002', fileName: 'ENB00001_alarm_20240601.log', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 512, logType: '告警日志', collectTime: '2024-06-01T23:00:00.000Z', uploadTime: '2024-06-01T23:30:00.000Z' },
  { id: 'dlf-003', fileName: 'GNB00001_runtime_20240601.log', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', fileSize: 1024 * 1024 * 8, logType: '运行日志', collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T01:00:00.000Z' },
];

const _sourceColorMap: Record<string, string> = { device: 'blue', user: 'green', nms: 'orange' };
const sourceLabelMap: Record<string, string> = { device: 'device', user: 'user', nms: 'NMS' };
const statusColorMap: Record<string, string> = { active: 'green', backup: 'default', deprecated: 'red' };
const statusLabelKeyMap: Record<string, string> = { active: 'status.online', backup: 'status.disabled', deprecated: 'status.failed' };

function ConfigFilesTab({ source }: { source: 'device' | 'user' | 'nms' }) {
  const t = useT();
  const filtered = mockDeviceConfigFiles.filter((f) => f.source === source);
  const columns: DataTableColumn<DeviceConfigFile & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, mono: true },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    { key: 'version', title: t('table.version'), dataIndex: 'version', width: 80 },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 80,
      render: (val) => <Tag color={statusColorMap[String(val)]}>{t(statusLabelKeyMap[String(val)])}</Tag>,
    },
    { key: 'uploadTime', title: t('table.time'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right' as const,
      render: () => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EyeOutlined />}>{t('common.detail')}</Button>
          <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>
        </Space>
      ),
    },
  ], [t]);
  return (
    <DataTable
      tableId={`config-files-${source}`}
      columns={columns}
      dataSource={filtered as (DeviceConfigFile & Record<string, unknown>)[]}
      loading={false}
      rowKey="id"
      total={filtered.length}
      pageSize={20}
      currentPage={1}
      scroll={{ x: 900 }}
    />
  );
}

function LogUploadConfigTab() {
  const t = useT();
  const [configs, setConfigs] = useState<LogUploadConfig[]>(mockLogUploadConfigs);
  const columns: DataTableColumn<LogUploadConfig & Record<string, unknown>>[] = useMemo(() => [
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, mono: true },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', ellipsis: true },
    { key: 'logType', title: t('table.type'), dataIndex: 'logType', width: 100 },
    {
      key: 'enabled', title: t('table.status'), dataIndex: 'enabled', width: 100,
      render: (val, record) => {
        const cfg = record as LogUploadConfig;
        return (
          <Switch
            checked={Boolean(val)}
            size="small"
            onChange={(checked) => {
              setConfigs((prev) => prev.map((c) => c.id === cfg.id ? { ...c, enabled: checked } : c));
              void message.success(checked ? t('common.enable') : t('common.disable'));
            }}
          />
        );
      },
    },
    { key: 'interval', title: t('table.time'), dataIndex: 'interval', width: 130 },
    { key: 'retentionDays', title: t('table.description'), dataIndex: 'retentionDays', width: 100 },
    {
      key: 'lastUpload', title: t('table.createTime'), dataIndex: 'lastUpload', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
  ], [t]);
  return (
    <DataTable
      tableId="log-upload-config"
      columns={columns}
      dataSource={configs as (LogUploadConfig & Record<string, unknown>)[]}
      loading={false}
      rowKey="id"
      total={configs.length}
      pageSize={20}
      currentPage={1}
    />
  );
}

function LogFilesListTab() {
  const t = useT();
  const columns: DataTableColumn<DeviceLogFile & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, mono: true },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    { key: 'logType', title: t('table.type'), dataIndex: 'logType', width: 100 },
    { key: 'collectTime', title: t('table.time'), dataIndex: 'collectTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    { key: 'uploadTime', title: t('table.createTime'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right' as const,
      render: () => (
        <Space size={4}>
          <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
        </Space>
      ),
    },
  ], [t]);
  return (
    <DataTable
      tableId="device-log-files"
      columns={columns}
      dataSource={mockDeviceLogFiles as (DeviceLogFile & Record<string, unknown>)[]}
      loading={false}
      rowKey="id"
      total={mockDeviceLogFiles.length}
      pageSize={20}
      currentPage={1}
      scroll={{ x: 1000 }}
    />
  );
}

export default function DeviceFiles() {
  const t = useT();
  const [activeTab, setActiveTab] = useState('config');

  return (
    <ListPageLayout title={t('nav.file.deviceFiles')}>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'config',
            label: t('nav.file.configRetrieval'),
            children: (
              <Tabs
                size="small"
                defaultActiveKey="device"
                items={[
                  { key: 'device', label: sourceLabelMap['device'], children: <ConfigFilesTab source="device" /> },
                  { key: 'user', label: sourceLabelMap['user'], children: <ConfigFilesTab source="user" /> },
                  { key: 'nms', label: sourceLabelMap['nms'], children: <ConfigFilesTab source="nms" /> },
                ]}
              />
            ),
          },
          {
            key: 'log',
            label: t('nav.file.logRetrieval'),
            children: (
              <Tabs
                size="small"
                defaultActiveKey="uploadConfig"
                items={[
                  { key: 'uploadConfig', label: t('common.deploy'), children: <LogUploadConfigTab /> },
                  { key: 'logFiles', label: t('table.total'), children: <LogFilesListTab /> },
                ]}
              />
            ),
          },
        ]}
      />
    </ListPageLayout>
  );
}
