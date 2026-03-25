import { useState, useCallback } from 'react';
import { Card, Tag, Typography } from 'antd';
import { AppstoreOutlined } from '@ant-design/icons';
import {
  DeviceTree,
  CommandTree,
  TerminalPanel,
  CommandInput,
  BatchSnModal,
} from './components';
import {
  useDeviceSelection,
  useCommandSelection,
  useCommandExecution,
} from './hooks';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

/**
 * MML 控制台 - 三栏布局
 *
 * 布局结构:
 * ┌─────────────┬─────────────┬───────────────────────────────────────┐
 * │   左栏      │    中栏     │                右栏                   │
 * │  设备选择   │   命令树    │  ┌─────────────────────────────────┐  │
 * │             │             │  │        终端输出 (40%)           │  │
 * │             │             │  └─────────────────────────────────┘  │
 * │             │             │  ┌─────────────────────────────────┐  │
 * │             │             │  │   操作面板 / 参数配置 (60%)     │  │
 * │             │             │  └─────────────────────────────────┘  │
 * └─────────────┴─────────────┴───────────────────────────────────────┘
 *
 * 列宽比例: 1fr : 1fr : 2fr
 */
export default function MMLConsole() {
  const t = useT();
  const token = useThemeToken();

  // 批量输入弹窗状态
  const [batchSnModalOpen, setBatchSnModalOpen] = useState(false);

  // 参数值状态
  const [paramValues, setParamValues] = useState<Record<string, string | number | boolean>>({});

  // 设备选择
  const deviceSelection = useDeviceSelection();

  // 命令选择
  const commandSelection = useCommandSelection();

  // 命令执行
  const commandExecution = useCommandExecution();

  // 执行命令
  const handleExecute = useCallback(() => {
    commandExecution.executeCommand(
      deviceSelection.selectedDevices,
      commandSelection.selectedCommand,
      paramValues
    );
  }, [commandExecution, deviceSelection.selectedDevices, commandSelection.selectedCommand, paramValues]);

  // 重置
  const handleReset = useCallback(() => {
    deviceSelection.clearSelection();
    commandSelection.clearSelection();
    commandExecution.clearOutput();
    setParamValues({});
  }, [deviceSelection, commandSelection, commandExecution]);

  // 批量输入确认
  const handleBatchSnConfirm = useCallback((sns: string[]) => {
    deviceSelection.addDevicesBySns(sns);
    setBatchSnModalOpen(false);
  }, [deviceSelection]);

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: token.colorBgLayout,
        padding: 12,
        gap: 12,
      }}
    >
      {/* 顶部工具栏 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 12px',
          background: token.colorBgContainer,
          borderRadius: 6,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            <AppstoreOutlined style={{ marginRight: 8 }} />
            {t('nav.mml.console')}
          </Typography.Title>
        </div>

        {/* 状态指示器 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Tag color={deviceSelection.selectedDevices.length > 0 ? 'blue' : 'default'}>
            设备: {deviceSelection.selectedDevices.length}
          </Tag>
          {commandSelection.selectedCommand && (
            <Tag color="green" style={{ fontFamily: 'monospace' }}>
              {commandSelection.selectedCommand.commandCode}
            </Tag>
          )}
        </div>
      </div>

      {/* 主内容区 - 三栏布局 */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr 2fr',
          flex: 1,
          gap: 12,
          minHeight: 0,
          overflow: 'hidden',
        }}
      >
        {/* 左栏：设备选择 */}
        <Card
          size="small"
          styles={{
            body: { padding: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: 1 },
          }}
          style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        >
          <DeviceTree
            selectedDevices={deviceSelection.selectedDevices}
            filteredDevices={deviceSelection.filteredDevices}
            paginatedDevices={deviceSelection.paginatedDevices}
            searchText={deviceSelection.searchText}
            productTypeFilter={deviceSelection.productTypeFilter}
            currentPage={deviceSelection.currentPage}
            isAllSelected={deviceSelection.isAllSelected}
            isIndeterminate={deviceSelection.isIndeterminate}
            totalFiltered={deviceSelection.totalFiltered}
            totalPages={deviceSelection.totalPages}
            onSearchChange={deviceSelection.setSearchText}
            onFilterChange={deviceSelection.setProductTypeFilter}
            onPageChange={deviceSelection.setCurrentPage}
            onToggleDevice={deviceSelection.toggleDevice}
            onToggleSelectAll={deviceSelection.toggleSelectAll}
            onRemoveDevice={deviceSelection.removeDevice}
            onClearSelection={deviceSelection.clearSelection}
            onBatchInput={() => setBatchSnModalOpen(true)}
          />
        </Card>

        {/* 中栏：命令树 */}
        <Card
          size="small"
          styles={{
            body: { padding: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: 1 },
          }}
          style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        >
          <CommandTree
            selectedCommand={commandSelection.selectedCommand}
            commands={commandSelection.filteredCommands}
            categories={commandSelection.categories}
            commandsByCategory={commandSelection.commandsByCategory}
            searchText={commandSelection.searchText}
            categoryFilter={commandSelection.categoryFilter}
            onSearchChange={commandSelection.setSearchText}
            onFilterChange={commandSelection.setCategoryFilter}
            onSelectCommand={commandSelection.selectCommand}
          />
        </Card>

        {/* 右栏：终端 + 操作面板 */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            height: '100%',
            minWidth: 0,
          }}
        >
          {/* 终端输出 - 固定50%高度 */}
          <div style={{ height: '50%', minHeight: 150 }}>
            <TerminalPanel
              lines={commandExecution.outputLines}
              onClear={commandExecution.clearOutput}
              onDownload={commandExecution.downloadOutput}
            />
          </div>

          {/* 命令输入和参数配置 - 占据剩余空间 */}
          <div style={{ flex: 1, minHeight: 200 }}>
            <CommandInput
              selectedDevices={deviceSelection.selectedDevices}
              selectedCommand={commandSelection.selectedCommand}
              paramValues={paramValues}
              onParamChange={setParamValues}
              onExecute={handleExecute}
              onReset={handleReset}
              loading={commandExecution.isExecuting}
            />
          </div>
        </div>
      </div>

      {/* 批量输入弹窗 */}
      <BatchSnModal
        open={batchSnModalOpen}
        onClose={() => setBatchSnModalOpen(false)}
        onConfirm={handleBatchSnConfirm}
        existingSns={new Set(deviceSelection.selectedDevices.map((d) => d.sn))}
      />
    </div>
  );
}
