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
import type { MMLCommand, MMLParamRef } from '@core/types/mml';
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
  const [selectedParams, setSelectedParams] = useState<string[]>([]);
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
      setSelectedParams([]);
      setParameters({});
      setOperationType('');
      setParamPaths(['']);
      commandExecution.clearOutput();
      return;
    }

    setIsManualEdit(false);
    commandExecution.clearOutput();
    setSelectedFields([]);
    setParameters({});

    const opType = resolveOperationType(selectedCommand);
    setOperationType(opType);
    setParamPaths(['']);

    // Auto-populate selectedParams with all paramRef codes when available
    const allParamCodes = selectedCommand.paramRefs?.map((p) => p.paramCode) ?? [];
    setSelectedParams(allParamCodes);

    // Assemble command line text with param string
    if (allParamCodes.length > 0) {
      if (opType === 'MOD' || opType === 'ADD') {
        // MOD/ADD 需要值：首次选中命令时值为空，给 <value> 占位提示。
        const pairs = allParamCodes.map((codeName) => `${codeName}=<value>`);
        setCommandLineText(`${selectedCommand.commandCode}:modId={${pairs.join(',')}};`);
      } else {
        setCommandLineText(`${selectedCommand.commandCode}:lstId={${allParamCodes.join(',')}};`);
      }
    } else {
      setCommandLineText(selectedCommand.commandCode);
    }

    setActiveTab('control');
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

    // Params format:
    //   LST/DSP: CMD:lstId={code1,code2,...};            —— 仅列字段，GetParameterValues
    //   MOD/ADD: CMD:modId={code1=val1,code2=val2,...};  —— 带值，SetParameterValues
    //
    // 初次选择命令时可能出现 selectedParams 尚未通过 ParamFormRenderer.onChange
    // 回传给父层，但 selectedCommand.paramRefs 已就绪的情况——此时直接把
    // paramRefs 的全集装配进命令行，避免用户第一次看到的只是"LST DEVICE_INFO"
    // 不带参数（to-do-list #5）。
    const paramRefs = selectedCommand.paramRefs ?? [];
    const effectiveParams =
      selectedParams.length > 0
        ? selectedParams
        : paramRefs.map((p) => p.paramCode);

    if (effectiveParams.length > 0 && paramRefs.length > 0) {
      if (nextOperationType === 'MOD' || nextOperationType === 'ADD') {
        // MOD/ADD 拼接 code=val；未填值的参数用占位符 <value> 提示用户补值。
        const pairs = effectiveParams.map((codeName) => {
          const raw = parameters[codeName];
          if (raw === undefined || raw === '') return `${codeName}=<value>`;
          return `${codeName}=${raw}`;
        });
        setCommandLineText(`${code}:modId={${pairs.join(',')}};`);
        return;
      }
      setCommandLineText(`${code}:lstId={${effectiveParams.join(',')}};`);
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
  }, [activeTab, commandSelection.selectedCommand, isManualEdit, parameters, selectedFields, selectedParams]);

  const handleCommandSelect = useCallback(async (command: MMLCommand | null, _param?: MMLParamRef | null) => {
    await commandSelection.selectCommand(command);
  }, [commandSelection]);

  const handleParamChange = useCallback((values: Record<string, unknown>) => {
    if (activeTab === 'control') {
      // 只更新 payload 中出现的字段，避免 ParamFormRenderer 发 {parameters: ...} 时
      // 把 selectedParams 误清空（MOD 场景下两者需要并存）。
      if (Array.isArray(values.selectedFields)) {
        setSelectedFields(values.selectedFields.map(String));
      }
      if (Array.isArray(values.selectedParams)) {
        setSelectedParams(values.selectedParams.map(String));
      }
      if (values.parameters && typeof values.parameters === 'object') {
        setParameters(values.parameters as CommandParameters);
      }
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
      selectedParams,
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
    selectedParams,
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
    setSelectedParams([]);
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
            commandName: template.commandName,
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
            // 模板没有单独的 selectedParams 概念；buildExecutePayload 会在
            // selectedParams 为空时自动使用 Object.keys(parameters) 作为兜底。
            selectedParams: [],
          });
        }}
      />

      {/* 保存脚本 → mml_scripts 表（to-do-list #3）。历史上复用过 ScriptTaskDrawer，
          但 Drawer 面向 mml_tasks 执行实例，语义不符。SaveScriptModal 只采集
          脚本名 + 描述 + 只读内容预览，提交走 useCreateMMLScript。*/}
      <SaveScriptModal
        open={saveScriptModalOpen}
        onClose={() => setSaveScriptModalOpen(false)}
        defaultName={saveScriptDefaultName}
        content={saveScriptDefaultContent}
      />
    </div>
  );
}
