import { useEffect, useMemo, useState } from 'react';
import { Card, Row, Col } from 'antd';
import { DeviceTree, CommandTree, RightPanel, BatchSnModal } from './components';
import StepBar from './components/StepBar';
import { useDeviceSelection } from './hooks';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';

export default function MMLConsole() {
  const dev = useDeviceSelection();
  const setSelectedDeviceSns = useMmlConsoleStore((s) => s.setSelectedDeviceSns);
  const statements = useMmlConsoleStore((s) => s.statements);
  const [batchModalOpen, setBatchModalOpen] = useState(false);

  useEffect(() => {
    setSelectedDeviceSns(dev.selectedDevices.map((d) => d.sn));
  }, [dev.selectedDevices, setSelectedDeviceSns]);

  const current: 1 | 2 | 3 | 4 = useMemo(() => {
    if (dev.selectedDevices.length === 0) return 1;
    if (statements.length === 0) return 2;
    return 3;
  }, [dev.selectedDevices.length, statements.length]);

  const existingSns = useMemo(
    () => new Set(dev.selectedDevices.map((d) => d.sn)),
    [dev.selectedDevices],
  );

  return (
    <Card variant="borderless">
      <StepBar current={current} />
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col span={6}>
          <DeviceTree
            selectedDevices={dev.selectedDevices}
            filteredDevices={dev.filteredDevices}
            paginatedDevices={dev.paginatedDevices}
            searchText={dev.searchText}
            productTypeFilter={dev.productTypeFilter}
            currentPage={dev.currentPage}
            isAllSelected={dev.isAllSelected}
            isIndeterminate={dev.isIndeterminate}
            totalFiltered={dev.totalFiltered}
            totalPages={dev.totalPages}
            onSearchChange={dev.setSearchText}
            onFilterChange={dev.setProductTypeFilter}
            onPageChange={dev.setCurrentPage}
            onToggleDevice={dev.toggleDevice}
            onToggleSelectAll={dev.toggleSelectAll}
            onRemoveDevice={dev.removeDevice}
            onClearSelection={dev.clearSelection}
            onBatchInput={() => setBatchModalOpen(true)}
          />
        </Col>
        <Col span={8}>
          <CommandTree />
        </Col>
        <Col span={10}>
          <RightPanel />
        </Col>
      </Row>
      <BatchSnModal
        open={batchModalOpen}
        onClose={() => setBatchModalOpen(false)}
        onConfirm={(sns) => {
          dev.addDevicesBySns(sns);
          setBatchModalOpen(false);
        }}
        existingSns={existingSns}
        allDeviceSns={dev.allDeviceSns}
      />
    </Card>
  );
}
