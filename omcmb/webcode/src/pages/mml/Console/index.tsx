import { useEffect, useMemo, useState } from 'react';
import { Card, Row, Col } from 'antd';
import ErrorBoundary from '@/components/common/ErrorBoundary';
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

  // 把 useDeviceSelection 的 productClassFilter（useState）单向镜像到 store，
  // 让兄弟组件 CommandTree 能订阅当前选中的 product_class 传给 useGroupTree
  // (后端按 paramModel 过滤命令)。useState 仍是设备列表逻辑的真相源，store 只读不写回。
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
    // 2026-05-28 用户决策(path 列表贴浏览器右):
    //   AppShell .content 的 padding-right: 24 + Card body padding-right: 16 共 40px,
    //   让 path 列表滚动条距浏览器右 40px。仅 Console 页用 mr:-24 + bodyStyle 突破,
    //   把操作面板这一栏的滚动条推到浏览器右边缘(其他页面布局不受影响)。
    <div style={{ marginRight: -24 }}>
      <Card
        variant="borderless"
        styles={{ body: { paddingRight: 0 } }}
      >
      <StepBar current={current} />
      {/* 2026-05-27 用户决策:把更多空间留给"终端输出 + 操作面板",大屏(xl ≥ 1200)
          下三栏从 6/8/10 调整为 5/7/12。lg 及更窄屏幕维持 6/8/10 不挤压设备/命令列表。 */}
      {/* 三栏各自用 ErrorBoundary 包裹：任一面板（设备树 / 命令树 / 操作终端）
          数据获取或渲染异常时只影响该栏，其余两栏仍可用，避免整页崩溃。 */}
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col xs={24} lg={6} xl={5}>
          <ErrorBoundary>
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
          </ErrorBoundary>
        </Col>
        <Col xs={24} lg={8} xl={7}>
          <ErrorBoundary>
            <CommandTree />
          </ErrorBoundary>
        </Col>
        <Col xs={24} lg={10} xl={12}>
          <ErrorBoundary>
            <RightPanel />
          </ErrorBoundary>
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
    </div>
  );
}
