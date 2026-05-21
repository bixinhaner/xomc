import React, { useCallback, useState } from 'react';
import { Button, Card, Typography } from 'antd';
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { BatchAction } from '@/components/DataTable';
import type { Device, BatchImportResponse } from '@core/types/device';
import BatchImportModal from './BatchImportModal';
import { useDeviceColumns } from './useDeviceColumns';

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
  /**
   * 批量导入完成回调（接收后端真实回执，含成功/失败统计）。
   * T-0202 后从原 fileList 改为 BatchImportResponse —— 解析与 POST 已下沉到 Modal。
   */
  onImport: (result: BatchImportResponse) => void | Promise<void>;
  onDownloadTemplate: () => void;
  onEditDevice: (device: Device) => void;
  t: (id: string, values?: Record<string, string | number>) => string;
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
  const [importModalOpen, setImportModalOpen] = useState(false);

  const handleImportClick = useCallback(() => {
    setImportModalOpen(true);
  }, []);

  const handleImportClose = useCallback(() => {
    setImportModalOpen(false);
  }, []);

  const columns = useDeviceColumns({ onEditDevice, t });

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

      <div className="device-list-table-wrapper" style={{ flex: 1, minHeight: 0 }}>
        <Card
          size="small"
          bordered
          style={{ height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
        >
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
            defaultDensity="default"
            scroll={{ x: 'max-content', y: 100 }}
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
          />
        </Card>
      </div>
      <style>{`
        /* 设备分组表格 - flex 布局自适应高度，无需硬编码偏移 */
        .device-list-table-wrapper {
          display: flex;
          flex-direction: column;
        }
        .device-list-table-wrapper .omc-data-table,
        .device-list-table-wrapper .ant-table-wrapper,
        .device-list-table-wrapper .ant-spin-nested-loading,
        .device-list-table-wrapper .ant-spin-nested-loading > div,
        .device-list-table-wrapper .ant-table,
        .device-list-table-wrapper .ant-table-container {
          display: flex !important;
          flex-direction: column !important;
          flex: 1 !important;
          min-height: 0 !important;
        }
        .device-list-table-wrapper .ant-table-body {
          flex: 1 !important;
          min-height: 0 !important;
          overflow-y: auto !important;
          max-height: none !important;
        }
        .device-list-table-wrapper .ant-table-thead > tr > th,
        .device-list-table-wrapper .ant-table-tbody > tr > td {
          font-size: 13px !important;
        }
      `}</style>

      <BatchImportModal
        open={importModalOpen}
        onClose={handleImportClose}
        onImport={onImport}
        onDownloadTemplate={onDownloadTemplate}
        t={t}
      />
    </div>
  );
}
