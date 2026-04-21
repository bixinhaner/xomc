import { useState, useCallback, useEffect } from 'react';
import { Card, message } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import {
  DeviceTree,
  CommandTree,
  TerminalPanel,
  CommandInput,
  BatchSnModal,
  SaveScriptModal,
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
import { resolveOperationType } from './utils/resolveOperationType';

type CommandInputTab = 'control' | 'paramPath';
type CommandParameters = Record<string, string | number | boolean>;

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
  const [saveScriptModalOpen, setSaveScriptModalOpen] = useState(false);

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
    setOperationType(resolveOperationType(selectedCommand));
    setParamPaths(['']);
    setCurrentCommandLabel(selectedCommand.commandCode);
  }, [commandSelection.selectedCommand]);

  useEffect(() => {
    const selectedCommand = commandSelection.selectedCommand;
    if (!selectedCommand || isManualEdit) {
      return;
    }

    const code = selectedCommand.commandCode;
    const nextOperationType = resolveOperationType(selectedCommand);

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
      setOperationType(resolveOperationType(commandSelection.selectedCommand));
      return;
    }

    setParamPaths(Array.isArray(values.paramPaths) ? values.paramPaths.map(String) : ['']);
    setOperationType(typeof values.operationType === 'string' ? values.operationType : resolveOperationType(commandSelection.selectedCommand));
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
      void message.warning(t('mml.selectCommandFirst'));
      return;
    }
    setSaveScriptModalOpen(true);
  }, [commandSelection.selectedCommand, t]);

  const saveScriptDefaultName = commandSelection.selectedCommand
    ? `${commandSelection.selectedCommand.commandName}_${new Date().toISOString().slice(0, 10)}`
    : '';
  const saveScriptDefaultContent = commandLineText.trim();

  const canExecute = deviceSelection.selectedDevices.length > 0 && commandLineText.trim().length > 0;
  const executeButtonText = `${t('mml.console.deviceUnit', { count: deviceSelection.selectedDevices.length })} · ${currentCommandLabel || t('mml.console.noCommandSelected')}`;

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
            commands={commandSelection.commands}
            selectedCommand={commandSelection.selectedCommand}
            treeData={commandSelection.treeData}
            searchText={commandSelection.searchText}
            isLoading={commandSelection.isLoading}
            onSearchChange={commandSelection.setSearchText}
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
        onSaveAndExecute={(template) => {
          const devices = deviceSelection.selectedDevices;
          if (devices.length === 0) {
            void message.warning(t('mml.console.errorNoDevice'));
            return;
          }

          const syntheticCommand: MMLCommand = {
            id: `tmpl-${Date.now()}`,
            commandName: template.templateName,
            commandCode: template.commandCode,
            category: template.categoryGroup || '',
            description: template.description,
            params: [],
            productTypes: template.productTypes,
            operationType: template.operationType,
          };

          commandExecution.executeCommand({
            activeTab: 'control',
            command: syntheticCommand,
            commandLineText: template.commandCode,
            devices,
            isManualEdit: false,
            operationType: template.operationType,
            paramPaths: template.paramPaths,
            parameters: template.parameters as Record<string, string | number | boolean>,
            selectedFields: [],
          });
        }}
      />

      <SaveScriptModal
        open={saveScriptModalOpen}
        defaultName={saveScriptDefaultName}
        defaultContent={saveScriptDefaultContent}
        onClose={() => setSaveScriptModalOpen(false)}
      />
    </div>
  );
}
