import React, { useCallback, useMemo, useState } from 'react';
import { Alert, Button, Input, Modal, Progress, Tag, Tooltip, Typography, Upload } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import { CheckCircleOutlined, CheckOutlined, CloseOutlined, DownloadOutlined, EditOutlined, InboxOutlined, UploadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import StatusIndicator from '@/components/StatusIndicator';
import type { Device, EngStatus } from '@/types/device';

const { Dragger } = Upload;
const { Title, Text } = Typography;

export interface DeviceListPanelProps {
  devices: Device[];
  total: number;
  loading: boolean;
  selectedDeviceIds: React.Key[];
  currentPage: number;
  pageSize: number;
  selectedGroupName: string | undefined;
  batchActions: BatchAction[];
  onSelectionChange: (keys: React.Key[]) => void;
  onPageChange: (page: number, size: number) => void;
  onRefresh: () => void;
  onExport: () => void;
  onImport: (fileList: UploadFile[]) => void;
  onDownloadTemplate: () => void;
  onEditDevice: (device: Device) => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

export default function DeviceListPanel({
  devices,
  total,
  loading,
  selectedDeviceIds,
  currentPage,
  pageSize,
  selectedGroupName,
  batchActions,
  onSelectionChange,
  onPageChange,
  onRefresh,
  onExport,
  onImport,
  onDownloadTemplate,
  onEditDevice,
  t,
}: DeviceListPanelProps) {
  // 批量导入弹窗状态
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importDone, setImportDone] = useState(false);

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      if (!file.name.endsWith('.csv')) {
        void Modal.error({ title: t('common.error'), content: t('device.fileFormatError') });
        return false;
      }
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        void Modal.error({ title: t('common.error'), content: t('device.fileSizeError') });
        return false;
      }
      setFileList([file]);
      setImportDone(false);
      return false;
    },
    onRemove: () => {
      setFileList([]);
      setImportDone(false);
    },
  };

  const handleImportClick = useCallback(() => {
    setFileList([]);
    setImporting(false);
    setImportProgress(0);
    setImportDone(false);
    setImportModalOpen(true);
  }, [t]);

  const handleImportConfirm = useCallback(async () => {
    if (fileList.length === 0) {
      void Modal.warning({ title: t('common.warning'), content: t('device.selectFileFirst') });
      return;
    }
    setImporting(true);
    setImportProgress(0);
    // 模拟导入进度
    for (let p = 0; p <= 100; p += 10) {
      await new Promise<void>((resolve) => setTimeout(resolve, 150));
      setImportProgress(p);
    }
    setImporting(false);
    setImportDone(true);
    onImport(fileList);
    // 延迟关闭弹窗，让用户看到成功提示
    await new Promise<void>((resolve) => setTimeout(resolve, 1000));
    setImportModalOpen(false);
  }, [fileList, onImport, t]);

  const handleImportCancel = useCallback(() => {
    if (!importing) {
      setImportModalOpen(false);
    }
  }, [importing]);

  const handleDownloadTemplate = useCallback(() => {
    onDownloadTemplate();
  }, [onDownloadTemplate]);

  const calculateOfflineDays = useCallback((lastOnlineTime: string): number => {
    if (!lastOnlineTime) return 0;
    const lastOnline = new Date(lastOnlineTime);
    const now = new Date();
    const diffMs = now.getTime() - lastOnline.getTime();
    return Math.max(0, Math.floor(diffMs / (1000 * 60 * 60 * 24)));
  }, []);

  // Remark 列头自定义标签
  const [remarkLabel, setRemarkLabel] = useState(() => {
    return localStorage.getItem('omc_grouping_remark_label') || t('device.remark');
  });
  const [editingRemark, setEditingRemark] = useState(false);
  const [remarkInput, setRemarkInput] = useState('');

  const handleRemarkLabelSave = useCallback(() => {
    const val = remarkInput.trim();
    if (!val) return;
    setRemarkLabel(val);
    setEditingRemark(false);
    localStorage.setItem('omc_grouping_remark_label', val);
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
            style={{ width: 100 }}
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
          <span style={{ maxWidth: 80, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
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

  const columns = useMemo(
    (): DataTableColumn<Device>[] => [
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 80,
        fixed: 'left',
        render: (_val, record) => (
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => onEditDevice(record)}
          >
            {t('common.edit')}
          </Button>
        ),
      },
      {
        key: 'connStatus',
        title: t('device.connStatus'),
        dataIndex: 'connStatus',
        width: 90,
        render: (val: string) => (
          <StatusIndicator
            status={val === 'online' ? 'online' : 'offline'}
            text={val === 'online' ? t('status.online') : t('status.offline')}
          />
        ),
      },
      {
        key: 'engStatus',
        title: t('device.installStatus'),
        dataIndex: 'engStatus',
        width: 100,
        render: (val: EngStatus) => {
          const statusMap: Record<EngStatus, { label: string; color: string }> = {
            commissioned: { label: t('device.engStatus.commissioned'), color: 'green' },
            uncommissioned: { label: t('device.engStatus.uncommissioned'), color: 'orange' },
            decommissioned: { label: t('device.engStatus.decommissioned'), color: 'red' },
          };
          const { label, color } = statusMap[val] || { label: val, color: 'default' };
          return <Tag color={color}>{label}</Tag>;
        },
      },
      {
        key: 'sn',
        title: t('device.serialNumber'),
        dataIndex: 'sn',
        width: 150,
        mono: true,
        copyable: true,
      },
      { key: 'name', title: t('device.stationName'), dataIndex: 'name', width: 160, ellipsis: true },
      { key: 'macAddress', title: t('device.macAddress'), dataIndex: 'macAddress', width: 150, mono: true },
      { key: 'groupName', title: t('device.groupName'), dataIndex: 'groupName', width: 140, ellipsis: true },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 100 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 100 },
      { key: 'gpsHeight', title: t('device.height'), dataIndex: 'gpsHeight', width: 80 },
      {
        key: 'offlineDays',
        title: t('device.offlineDays'),
        width: 100,
        render: (_val, record) => {
          if (record.connStatus === 'online') return '-';
          return calculateOfflineDays(record.lastOnlineTime);
        },
      },
      {
        key: 'remark',
        title: remarkLabel,
        dataIndex: 'remark',
        width: 140,
        ellipsis: true,
        headerRender: remarkHeaderRender,
      },
    ],
    [t, calculateOfflineDays, onEditDevice, remarkLabel, remarkHeaderRender]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 16, gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Title level={5} style={{ margin: 0 }}>
          {selectedGroupName ?? t('common.all')}
          <Text type="secondary" style={{ fontSize: 13, marginLeft: 8, fontWeight: 400 }}>
            {t('table.total')} {total}
          </Text>
        </Title>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button
            icon={<UploadOutlined />}
            onClick={handleImportClick}
          >
            {t('common.batchImport')}
          </Button>
          <Button
            type="primary"
            icon={<DownloadOutlined />}
            onClick={onExport}
          >
            {t('common.export')}
          </Button>
        </div>
      </div>

      <div className="device-list-table-wrapper" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <DataTable<Device>
          tableId="device-grouping-table"
          columns={columns}
          dataSource={devices}
          loading={loading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedDeviceIds}
          onSelectionChange={onSelectionChange}
          batchActions={batchActions}
          total={total}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={onPageChange}
          onRefresh={onRefresh}
          defaultDensity="compact"
        />
      </div>
      <style>{`
        .device-list-table-wrapper {
          flex: 1;
          min-height: 0;
          display: flex;
          flex-direction: column;
        }
        .device-list-table-wrapper > div[class*="dataTableWrapper"] {
          flex: 1;
          min-height: 0;
          display: flex;
          flex-direction: column;
        }
        .device-list-table-wrapper > div[class*="dataTableWrapper"] > div[class*="tableContainer"] {
          flex: 1;
          min-height: 0;
          overflow: auto;
        }
        .device-list-table-wrapper .ant-table-wrapper {
          height: 100%;
        }
        .device-list-table-wrapper .ant-table-wrapper .ant-table {
          height: 100%;
        }
        .device-list-table-wrapper .ant-table-wrapper .ant-table-container {
          height: 100%;
          display: flex;
          flex-direction: column;
        }
        .device-list-table-wrapper .ant-table-wrapper .ant-table-body {
          flex: 1;
          overflow: auto !important;
        }
      `}</style>

      {/* 批量导入弹窗 */}
      <Modal
        title={t('common.batchImport')}
        open={importModalOpen}
        onCancel={handleImportCancel}
        footer={null}
        width={520}
        maskClosable={!importing}
        closable={!importing}
      >
        <div style={{ marginBottom: 12 }}>
          <Button
            icon={<DownloadOutlined />}
            size="small"
            onClick={handleDownloadTemplate}
          >
            {t('device.downloadImportTemplate')}
          </Button>
        </div>

        <Dragger {...uploadProps} style={{ marginBottom: 16 }}>
          <p className="ant-upload-drag-icon">
            <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
          </p>
          <p className="ant-upload-text">{t('common.upload')}</p>
          <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
            .csv
          </p>
        </Dragger>

        {importing && (
          <div style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ fontSize: 13 }}>
              {t('common.loading')}
            </Text>
            <Progress percent={importProgress} status="active" />
          </div>
        )}

        {importDone && (
          <Alert
            type="success"
            showIcon
            icon={<CheckCircleOutlined />}
            message={t('device.importSuccess')}
            style={{ marginBottom: 16 }}
          />
        )}

        <Button
          type="primary"
          icon={<UploadOutlined />}
          loading={importing}
          onClick={() => void handleImportConfirm()}
          disabled={fileList.length === 0}
          block
        >
          {t('common.import')}
        </Button>
      </Modal>
    </div>
  );
}
