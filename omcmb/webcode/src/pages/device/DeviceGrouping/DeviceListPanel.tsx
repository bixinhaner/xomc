import React, { useCallback, useMemo, useState } from 'react';
import { Button, Card, Dropdown, Typography } from 'antd';
import type { MenuProps } from 'antd';
import { DownloadOutlined, DownOutlined, UploadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import SearchInput from '@/components/SearchInput';
import type { BatchAction } from '@/components/DataTable';
import type { BatchImportResponse, BatchPreRegisterResponse, Device } from '@core/types/device';
import BatchImportModal from './BatchImportModal';
import BatchPreRegisterModal from './BatchPreRegisterModal';
import { useDeviceColumns } from './useDeviceColumns';

const { Title, Text } = Typography;

export interface DeviceListPanelProps {
  devices: Device[];
  total: number;
  loading: boolean;
  selectedDeviceIds: React.Key[];
  currentPage: number;
  pageSize: number;
  selectedGroupId: string | null;
  selectedGroupName: string | undefined;
  batchActions: BatchAction[];
  onSelectionChange: (keys: React.Key[]) => void;
  onPageChange: (page: number, size: number) => void;
  /** SN / 设备名称 模糊搜索（多个以逗号分隔），回车或点搜索触发。 */
  onSearch: (value: string) => void;
  onExport: () => void | Promise<void>;
  /**
   * 批量导入完成回调。
   */
  onImport: (result: BatchImportResponse) => void | Promise<void>;
  onDownloadTemplate: () => void;
  /** 批量预登记完成回调。 */
  onPreRegister: (result: BatchPreRegisterResponse) => void | Promise<void>;
  onDownloadPreRegisterTemplate: () => void;
  t: (id: string, values?: Record<string, string | number>) => string;
}

export default function DeviceListPanel({
  devices,
  total,
  loading,
  selectedDeviceIds,
  currentPage,
  pageSize,
  selectedGroupId,
  selectedGroupName,
  batchActions,
  onSelectionChange,
  onPageChange,
  onSearch,
  onExport,
  onImport,
  onDownloadTemplate,
  onPreRegister,
  onDownloadPreRegisterTemplate,
  t,
}: DeviceListPanelProps) {
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [preRegModalOpen, setPreRegModalOpen] = useState(false);

  const handleImportClick = useCallback(() => {
    setImportModalOpen(true);
  }, []);

  const handleImportClose = useCallback(() => {
    setImportModalOpen(false);
  }, []);

  const handlePreRegClick = useCallback(() => {
    setPreRegModalOpen(true);
  }, []);

  const handlePreRegClose = useCallback(() => {
    setPreRegModalOpen(false);
  }, []);

  const columns = useDeviceColumns({ t });

  // 搜索框放到工具栏最左、批量操作（移动/回收站/删除）按钮之前（DataTable.extraToolbarLeft）。
  const searchBox = useMemo(
    () => (
      <SearchInput
        allowClear
        placeholder={t('device.searchSnNamePlaceholder')}
        // 2026-06-03:SearchInput 已用 Tooltip(hover+focus)展示完整 placeholder,无需再设 title。
        // enterButton 对齐 product/products 搜索框样式(带明显「搜索」按钮)。
        enterButton
        onSearch={onSearch}
      />
    ),
    [onSearch, t],
  );

  // 导出 / 导入按钮放到工具栏「删除」按钮之后（DataTable.extraToolbarAfterBatch）。
  // Import 升级为下拉：「更新名称/备注」和「批量预登记」两个选项。
  const importMenuItems: MenuProps['items'] = useMemo(() => [
    { key: 'import', label: t('common.import'), icon: <UploadOutlined />, onClick: handleImportClick },
    { key: 'preregister', label: t('device.batchPreRegister'), icon: <UploadOutlined />, onClick: handlePreRegClick },
  ], [handleImportClick, handlePreRegClick, t]);

  const importExportButtons = useMemo(
    () => (
      <>
        <Button size="small" type="primary" icon={<DownloadOutlined />} onClick={onExport}>
          {t('common.export')}
        </Button>
        <Dropdown menu={{ items: importMenuItems }} trigger={['click']}>
          <Button size="small" icon={<UploadOutlined />}>
            {t('common.import')} <DownOutlined />
          </Button>
        </Dropdown>
      </>
    ),
    [importMenuItems, onExport, t],
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 16, gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
        <Title level={5} style={{ margin: 0 }}>
          {selectedGroupName ?? t('common.all')}
          <Text type="secondary" style={{ fontSize: 13, marginLeft: 8, fontWeight: 400 }}>
            {t('table.total')} {total}
          </Text>
        </Title>
      </div>

      <div className="device-list-table-wrapper" style={{ flex: 1, minHeight: 0 }}>
        <Card
          size="small"
          variant="outlined"
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
            extraToolbarLeft={searchBox}
            extraToolbarAfterBatch={importExportButtons}
            total={total}
            pageSize={pageSize}
            currentPage={currentPage}
            onPageChange={onPageChange}
            hideRealtime
            hideColumnSettings
            hideDensity
            defaultDensity="default"
            scroll={{ x: 'max-content', y: 100 }}
            showRowNumber
            rowNumberTitle={t('table.rowNumber')}
          />
        </Card>
      </div>
      <style>{`
        /* Device grouping table - flex layout auto-fits height, no hardcoded offset needed */
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
        selectedGroupId={selectedGroupId}
      />
      <BatchPreRegisterModal
        open={preRegModalOpen}
        onClose={handlePreRegClose}
        onPreRegister={onPreRegister}
        onDownloadTemplate={onDownloadPreRegisterTemplate}
        t={t}
      />
    </div>
  );
}
