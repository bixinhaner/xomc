import { useState, useCallback, useEffect } from 'react';
import { Card, Tag, Typography, message } from 'antd';
import { AppstoreOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import {
  DeviceTree,
  CommandTree,
  TerminalPanel,
  CommandInput,
  BatchSnModal,
} from './components';
import AddTemplateModal from './components/AddTemplateModal';
import {
  useDeviceSelection,
  useCommandSelection,
  useCommandExecution,
} from './hooks';
import type { MMLCommand } from '@/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import { useCreateMMLScript } from '@/hooks/api/useMML';

type CommandInputTab = 'control' | 'paramPath';
type CommandParameters = Record<string, string | number | boolean>;

function getOperationType(command: MMLCommand | null): string {
  const operationType = command?.operationType?.trim().toUpperCase();
  if (operationType) {
    return operationType;
  }

  return command?.commandCode?.trim().split(/\s+/)[0]?.toUpperCase() || 'LST';
}

export default function MMLConsole() {
  const t = useT();
  const token = useThemeToken();
  const queryClient = useQueryClient();

  const [batchSnModalOpen, setBatchSnModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<CommandInputTab>('control');
  const [commandLineText, setCommandLineText] = useState('');
  const [isManualEdit, setIsManualEdit] = useState(false);
  const [currentCommandLabel, setCurrentCommandLabel] = useState('');
  const [selectedFields, setSelectedFields] = useState<string[]>([]);
  const [parameters, setParameters] = useState<CommandParameters>({});
  const [operationType, setOperationType] = useState('');
  const [paramPaths, setParamPaths] = useState<string[]>(['']);
  const [addTemplateModalOpen, setAddTemplateModalOpen] = useState(false);
  const [addTemplateScope, setAddTemplateScope] = useState<'public' | 'private'>('private');

  const deviceSelection = useDeviceSelection();
  const commandSelection = useCommandSelection();
  const commandExecution = useCommandExecution();
  const createScriptMutation = useCreateMMLScript();

  useEffect(() => {
    const selectedCommand = commandSelection.selectedCommand;

    if (!selectedCommand) {
      setCommandLineText('');
      setIsManualEdit(false);
      setCurrentCommandLabel('');
      setSelectedFields([]);
      setParameters({});
      setOperationType('');
      setParamPaths(['']);
      return;
    }

    setActiveTab('control');
    setCommandLineText(selectedCommand.commandCode);
    setIsManualEdit(false);
    setSelectedFields([]);
    setParameters({});
    setOperationType(getOperationType(selectedCommand));
    setParamPaths(['']);
    setCurrentCommandLabel(selectedCommand.commandCode);
  }, [commandSelection.selectedCommand]);

  useEffect(() => {
    const selectedCommand = commandSelection.selectedCommand;
    if (!selectedCommand || isManualEdit) {
      return;
    }

    const code = selectedCommand.commandCode;
    const nextOperationType = getOperationType(selectedCommand);

    if (activeTab !== 'control') {
      setCommandLineText(code);
      return;
    }

    if (nextOperationType === 'LST' || nextOperationType === 'DSP') {
      setCommandLineText(selectedFields.length > 0 ? `${code}:${selectedFields.join(',')}` : code);
      return;
    }

    if (['MOD', 'ADD', 'RMV', 'DEL'].includes(nextOperationType)) {
      const paramStr = Object.entries(parameters)
        .filter(([, value]) => value !== undefined && value !== '')
        .map(([key, value]) => `${key}=${value}`)
        .join(',');

      setCommandLineText(paramStr ? `${code}:${paramStr}` : code);
      return;
    }

    setCommandLineText(code);
  }, [activeTab, commandSelection.selectedCommand, isManualEdit, parameters, selectedFields]);

  const handleCommandSelect = useCallback(async (command: MMLCommand | null) => {
    await commandSelection.selectCommand(command);
  }, [commandSelection]);

  const handleParamChange = useCallback((values: Record<string, unknown>) => {
    if (activeTab === 'control') {
      setSelectedFields(Array.isArray(values.selectedFields) ? values.selectedFields.map(String) : []);
      setParameters(
        values.parameters && typeof values.parameters === 'object'
          ? values.parameters as CommandParameters
          : {}
      );
      setOperationType(getOperationType(commandSelection.selectedCommand));
      return;
    }

    setParamPaths(Array.isArray(values.paramPaths) ? values.paramPaths.map(String) : ['']);
    setOperationType(typeof values.operationType === 'string' ? values.operationType : getOperationType(commandSelection.selectedCommand));
  }, [activeTab, commandSelection.selectedCommand]);

  const handleCommandLineChange = useCallback((value: string) => {
    setCommandLineText(value);
    setIsManualEdit(true);
  }, []);

  const handleExecute = useCallback(() => {
    commandExecution.executeCommand({
      activeTab,
      command: commandSelection.selectedCommand,
      commandLineText,
      devices: deviceSelection.selectedDevices,
      isManualEdit,
      operationType,
      paramPaths,
      parameters,
      selectedFields,
    });
  }, [
    activeTab,
    commandExecution,
    commandLineText,
    commandSelection.selectedCommand,
    deviceSelection.selectedDevices,
    isManualEdit,
    operationType,
    paramPaths,
    parameters,
    selectedFields,
  ]);

  const handleReset = useCallback(() => {
    deviceSelection.clearSelection();
    commandSelection.clearSelection();
    commandExecution.clearOutput();
    setActiveTab('control');
    setCommandLineText('');
    setIsManualEdit(false);
    setCurrentCommandLabel('');
    setSelectedFields([]);
    setParameters({});
    setOperationType('');
    setParamPaths(['']);
  }, [deviceSelection, commandSelection, commandExecution]);

  const handleBatchSnConfirm = useCallback((sns: string[]) => {
    deviceSelection.addDevicesBySns(sns);
    setBatchSnModalOpen(false);
  }, [deviceSelection]);

  const handleSaveScript = useCallback(() => {
    const cmd = commandSelection.selectedCommand;
    if (!cmd) {
      void message.warning(t('mml.selectCommandFirst') || '请先选择命令');
      return;
    }
    const lines = [commandLineText.trim()];
    createScriptMutation.mutate(
      {
        scriptName: `${cmd.commandName}_${new Date().toISOString().slice(0, 10)}`,
        description: `Saved from MML console: ${cmd.commandCode}`,
        content: lines.join('\n'),
        deviceType: '',
        creator: '',
        tags: [],
      },
      {
        onSuccess: () => void message.success(t('mml.scriptSaved') || '脚本已保存'),
        onError: (err) => void message.error(t('mml.scriptSaveFailed') || '保存失败: ' + (err instanceof Error ? err.message : 'Unknown')),
      },
    );
  }, [commandSelection.selectedCommand, commandLineText, createScriptMutation, t]);

  const canExecute = deviceSelection.selectedDevices.length > 0 && commandLineText.trim().length > 0;
  const executeButtonText = `${deviceSelection.selectedDevices.length} 设备 · ${currentCommandLabel || '未选命令'}`;

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: token.colorBgLayout,
        padding: 16,
        gap: 12,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px',
          background: token.colorBgContainer,
          borderRadius: 8,
          border: `1px solid ${token.colorBorderSecondary}`,
          boxShadow: '0 1px 4px rgba(0, 0, 0, 0.04)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 6,
              background: token.colorPrimaryBg,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <AppstoreOutlined style={{ fontSize: 18, color: token.colorPrimary }} />
          </div>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {t('nav.mml.console')}
          </Typography.Title>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Tag
            style={{
              background: token.colorPrimaryBg,
              border: `1px solid ${token.colorPrimaryBorder}`,
              color: token.colorPrimary,
              borderRadius: 12,
              padding: '2px 10px',
            }}
          >
            设备: {deviceSelection.selectedDevices.length}
          </Tag>
          {commandSelection.selectedCommand && (
            <Tag
              style={{
                background: token.colorPrimaryBg,
                border: `1px solid ${token.colorPrimaryBorder}`,
                color: token.colorPrimary,
                fontFamily: 'monospace',
                borderRadius: 12,
                padding: '2px 10px',
              }}
            >
              {commandSelection.selectedCommand.commandCode}
            </Tag>
          )}
        </div>
      </div>

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

        <Card
          size="small"
          styles={{
            body: { padding: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', flex: 1 },
          }}
          style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        >
          <CommandTree
            selectedCommand={commandSelection.selectedCommand}
            treeData={commandSelection.treeData}
            categoryOptions={commandSelection.categoryOptions}
            searchText={commandSelection.searchText}
            categoryFilter={commandSelection.categoryFilter}
            isLoading={commandSelection.isLoading}
            onSearchChange={commandSelection.setSearchText}
            onFilterChange={commandSelection.setCategoryFilter}
            onSelectCommand={handleCommandSelect}
            onAddPublicTemplate={() => {
              setAddTemplateScope('public');
              setAddTemplateModalOpen(true);
            }}
            onAddPrivateTemplate={() => {
              setAddTemplateScope('private');
              setAddTemplateModalOpen(true);
            }}
          />
        </Card>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            minHeight: 0,
            minWidth: 0,
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              height: '50%',
              minHeight: 120,
              overflow: 'hidden',
            }}
          >
            <TerminalPanel
              lines={commandExecution.outputLines}
              onClear={commandExecution.clearOutput}
              onDownload={commandExecution.downloadOutput}
            />
          </div>

          <div
            style={{
              flex: 1,
              minHeight: 0,
              overflow: 'hidden',
            }}
          >
            <CommandInput
              activeTab={activeTab}
              canExecute={canExecute}
              commandLineText={commandLineText}
              currentCommandLabel={currentCommandLabel}
              executeButtonText={executeButtonText}
              loading={commandExecution.isExecuting}
              onActiveTabChange={setActiveTab}
              onCommandLineChange={handleCommandLineChange}
              onExecute={handleExecute}
              onParamChange={handleParamChange}
              onReset={handleReset}
              onSaveScript={handleSaveScript}
              selectedCommand={commandSelection.selectedCommand}
              selectedDevices={deviceSelection.selectedDevices}
            />
          </div>
        </div>
      </div>

      <BatchSnModal
        open={batchSnModalOpen}
        onClose={() => setBatchSnModalOpen(false)}
        onConfirm={handleBatchSnConfirm}
        existingSns={new Set(deviceSelection.selectedDevices.map((d) => d.sn))}
        allDeviceSns={deviceSelection.allDeviceSns}
      />

      <AddTemplateModal
        open={addTemplateModalOpen}
        scope={addTemplateScope}
        onClose={() => setAddTemplateModalOpen(false)}
        onSuccess={() => {
          void queryClient.invalidateQueries({ queryKey: ['mml', 'templates'] });
        }}
      />
    </div>
  );
}
