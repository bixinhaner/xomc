import { useEffect, useMemo, useState } from 'react';
import { Card, Row, Col } from 'antd';
import { DeviceTree, CommandTree, RightPanel, BatchSnModal } from './components';
import StepBar from './components/StepBar';
import { useDeviceSelection } from './hooks';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';

export default function MMLConsole() {
  const dev = useDeviceSelection();
  const setSelectedDeviceSns = useMmlConsoleStore((s) => s.setSelectedDeviceSns);
  const setProductClassFilter = useMmlConsoleStore((s) => s.setProductClassFilter);
  const statements = useMmlConsoleStore((s) => s.statements);
  const [batchModalOpen, setBatchModalOpen] = useState(false);

  useEffect(() => {
    setSelectedDeviceSns(dev.selectedDevices.map((d) => d.sn));
  }, [dev.selectedDevices, setSelectedDeviceSns]);

  // R-8.5：把 useDeviceSelection 的 productClassFilter（useState）单向镜像到 store，
  // 让兄弟组件 CommandTree 能订阅当前选中的 product_class 调 useCommandCompatibility。
  // useState 仍是设备列表逻辑的真相源，store 只读不写回。
  useEffect(() => {
    setProductClassFilter(dev.productClassFilter);
  }, [dev.productClassFilter, setProductClassFilter]);

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
      {/* 2026-05-27 用户决策:把更多空间留给"终端输出 + 操作面板",大屏(xl ≥ 1200)
          下三栏从 6/8/10 调整为 5/7/12。lg 及更窄屏幕维持 6/8/10 不挤压设备/命令列表。 */}
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col xs={24} lg={6} xl={5}>
          <DeviceTree
            selectedDevices={dev.selectedDevices}
            filteredDevices={dev.filteredDevices}
            paginatedDevices={dev.paginatedDevices}
            searchText={dev.searchText}
            productClassFilter={dev.productClassFilter}
            currentPage={dev.currentPage}
            isAllSelected={dev.isAllSelected}
            isIndeterminate={dev.isIndeterminate}
            totalFiltered={dev.totalFiltered}
            totalPages={dev.totalPages}
            onSearchChange={dev.setSearchText}
            onFilterChange={dev.setProductClassFilter}
            onPageChange={dev.setCurrentPage}
            onToggleDevice={dev.toggleDevice}
            onToggleSelectAll={dev.toggleSelectAll}
            onRemoveDevice={dev.removeDevice}
            onClearSelection={dev.clearSelection}
            onBatchInput={() => setBatchModalOpen(true)}
          />
        </Col>
        <Col xs={24} lg={8} xl={7}>
          <CommandTree />
        </Col>
        <Col xs={24} lg={10} xl={12}>
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
