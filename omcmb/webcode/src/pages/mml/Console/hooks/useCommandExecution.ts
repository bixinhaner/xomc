import { useState, useCallback, useRef, useEffect } from 'react';
import type { TerminalLine } from '../types';
import type { ConsoleDevice } from '../types';
import type { MMLCommand, MMLTask } from '@core/types/mml';
import { useExecuteMMLCommand, useMMLTaskPolling } from '@core/hooks/api/useMML';
import { useT } from '@/hooks/useT';

const TERMINAL_STATES = new Set(['completed', 'failed', 'cancelled']);

type ExecutePayload = Record<string, unknown>;

type ExecuteCommandParams = {
  activeTab: 'control' | 'paramPath';
  command: MMLCommand | null;
  commandLineText: string;
  devices: ConsoleDevice[];
  isManualEdit: boolean;
  operationType: string;
  paramPaths: string[];
  parameters: Record<string, string | number | boolean>;
  selectedFields: string[];
  selectedParams: string[];
  taskName?: string;
};

export function useCommandExecution() {
  const t = useT();
  const [outputLines, setOutputLines] = useState<TerminalLine[]>([]);
  const [isExecuting, setIsExecuting] = useState(false);
  const [pollingTaskId, setPollingTaskId] = useState<string | null>(null);
  const lastPolledStatus = useRef<string>('');

  const executeMutation = useExecuteMMLCommand();

  const { data: polledTask } = useMMLTaskPolling(pollingTaskId, !!pollingTaskId);

  const addOutput = useCallback((line: TerminalLine) => {
    setOutputLines((prev) => [...prev, line]);
  }, []);

  const clearOutput = useCallback(() => {
    setOutputLines([]);
    setPollingTaskId(null);
    setIsExecuting(false);
    lastPolledStatus.current = '';
  }, []);

  useEffect(() => {
    if (!polledTask || !pollingTaskId) return;

    if (lastPolledStatus.current === polledTask.status) return;
    lastPolledStatus.current = polledTask.status;

    if (polledTask.status === 'completed') {
      addOutput({
        type: 'success',
        text: t('mml.console.taskCompleted', { success: String(polledTask.successCount), failed: String(polledTask.failedCount) }),
        timestamp: new Date().toLocaleTimeString(),
      });

      if (polledTask.results && polledTask.results.length > 0) {
        addOutput({ type: 'info', text: '─'.repeat(50), timestamp: new Date().toLocaleTimeString() });
        for (const r of polledTask.results) {
          addOutput({
            type: r.result.success ? 'stdout' : 'stderr',
            text: `[${r.deviceSn}] ${r.result.success ? t('mml.console.execSuccess') : t('mml.console.execFailed')}${r.result.rawOutput ? '\n' + r.result.rawOutput : ''}`,
            timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
          });
        }
      }

      addOutput({ type: 'info', text: '', timestamp: new Date().toLocaleTimeString() });
      setPollingTaskId(null);
      setIsExecuting(false);
    } else if (polledTask.status === 'failed') {
      addOutput({
        type: 'stderr',
        text: t('mml.console.taskFailed', { error: polledTask.result || 'Unknown' }),
        timestamp: new Date().toLocaleTimeString(),
      });

      if (polledTask.results && polledTask.results.length > 0) {
        for (const r of polledTask.results) {
          if (r.result.rawOutput) {
            addOutput({
              type: 'stderr',
              text: `[${r.deviceSn}] ${r.result.rawOutput}`,
              timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
            });
          }
        }
      }

      setPollingTaskId(null);
      setIsExecuting(false);
    } else if (polledTask.status === 'cancelled') {
      addOutput({
        type: 'stderr',
        text: t('mml.console.taskCancelled'),
        timestamp: new Date().toLocaleTimeString(),
      });
      setPollingTaskId(null);
      setIsExecuting(false);
    } else if (polledTask.status === 'running') {
      addOutput({
        type: 'info',
        text: t('mml.console.taskRunning', { done: String(polledTask.successCount + polledTask.failedCount), total: String(polledTask.totalDevices) }),
        timestamp: new Date().toLocaleTimeString(),
      });
    }
  }, [polledTask, pollingTaskId, addOutput]);

  const executeCommand = useCallback(async ({
    activeTab,
    command,
    commandLineText,
    devices,
    isManualEdit,
    operationType,
    paramPaths,
    parameters,
    selectedFields,
    selectedParams,
    taskName,
  }: ExecuteCommandParams) => {
    if (devices.length === 0) {
      addOutput({
        type: 'stderr',
        text: t('mml.console.errorNoDevice'),
        timestamp: new Date().toLocaleTimeString(),
      });
      return;
    }

    const trimmedCommandLineText = commandLineText.trim();
    const filteredPaths = paramPaths.filter((p) => p.trim());

    // paramPath 模式 + 无 command：直接看 paramPaths 是否非空，跳过 commandLineText
    // 校验（commandLineText 是 control 面板专用的命令行字符串，paramPath 模式不渲染）。
    const isRawParamPathMode = activeTab === 'paramPath' && !command && !isManualEdit;
    if (isRawParamPathMode) {
      if (filteredPaths.length === 0) {
        addOutput({
          type: 'stderr',
          text: t('mml.console.errorNoParamPath'),
          timestamp: new Date().toLocaleTimeString(),
        });
        return;
      }
    } else {
      if (!trimmedCommandLineText) {
        addOutput({
          type: 'stderr',
          text: t('mml.console.errorNoCommand'),
          timestamp: new Date().toLocaleTimeString(),
        });
        return;
      }
      if (!command && !isManualEdit) {
        addOutput({
          type: 'stderr',
          text: t('mml.console.errorSelectCommand'),
          timestamp: new Date().toLocaleTimeString(),
        });
        return;
      }
    }

    const payload = buildExecutePayload({
      activeTab,
      command,
      commandLineText: trimmedCommandLineText,
      devices,
      isManualEdit,
      operationType,
      paramPaths,
      parameters,
      selectedFields,
      selectedParams,
      taskName: taskName || command?.commandCode || trimmedCommandLineText,
    });

    setIsExecuting(true);

    // 显示在终端的命令字符串：paramPath 裸路径模式下用 paths 拼接，否则沿用 commandLineText
    const displayCommand = isRawParamPathMode
      ? filteredPaths.join(', ')
      : trimmedCommandLineText;
    addOutput({
      type: 'info',
      text: t('mml.console.executingCommand', { command: displayCommand }),
      timestamp: new Date().toLocaleTimeString(),
    });
    addOutput({
      type: 'info',
      text: t('mml.console.targetDevices', { devices: devices.map((d) => d.sn).join(', ') }),
      timestamp: new Date().toLocaleTimeString(),
    });
    addOutput({
      type: 'info',
      text: '─'.repeat(50),
      timestamp: new Date().toLocaleTimeString(),
    });

    try {
      const task = await executeMutation.mutateAsync({ payload });

      addOutput({
        type: 'success',
        text: t('mml.console.commandSubmitted', { id: task.id }),
        timestamp: new Date().toLocaleTimeString(),
      });

      if (TERMINAL_STATES.has(task.status) && task.results && task.results.length > 0) {
        outputTaskResults(t, task, addOutput);
        addOutput({ type: 'info', text: '', timestamp: new Date().toLocaleTimeString() });
        setIsExecuting(false);
      } else {
        addOutput({
          type: 'info',
          text: t('mml.console.waitingForResults', { count: String(task.totalDevices || devices.length) }),
          timestamp: new Date().toLocaleTimeString(),
        });
        lastPolledStatus.current = '';
        setPollingTaskId(task.id);
      }
    } catch (error) {
      addOutput({
        type: 'stderr',
        text: t('mml.console.commandSubmitFailed', { error: error instanceof Error ? error.message : 'Unknown' }),
        timestamp: new Date().toLocaleTimeString(),
      });
      setIsExecuting(false);
    }
  }, [executeMutation, addOutput]);

  const downloadOutput = useCallback(() => {
    const content = outputLines
      .map((line) => {
        const prefix = line.timestamp ? `[${line.timestamp}] ` : '';
        return `${prefix}${line.text}`;
      })
      .join('\n');

    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `mml-output-${new Date().toISOString().slice(0, 10)}.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [outputLines]);

  return {
    outputLines,
    isExecuting,
    addOutput,
    clearOutput,
    executeCommand,
    downloadOutput,
  };
}

function buildExecutePayload({
  activeTab,
  command,
  commandLineText,
  devices,
  isManualEdit,
  operationType,
  paramPaths,
  parameters,
  selectedFields,
  selectedParams,
  taskName,
}: ExecuteCommandParams): ExecutePayload {
  const deviceSns = devices.map((device) => device.sn);
  const manualCommands = commandLineText
    .split(';')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => ({ command_code: item }));

  if (isManualEdit || !command) {
    return {
      task_name: taskName || commandLineText,
      device_sns: deviceSns,
      commands: manualCommands,
    };
  }

  if (activeTab === 'control') {
    // 把控制面板上选中的参数码 + 填写的值合并成 commands[0].parameters。
    //
    // - LST/DSP/RMV：仅勾选语义，value 占位 ""。
    // - MOD/ADD：用户未填的参数过滤掉（to-do-list 当轮 #1：避免覆盖未修改的字段）。
    //   "未填" 判定：undefined / null / 空字符串。0 / false 是合法值，必须保留。
    const editOps = new Set<string>(['MOD', 'ADD']);
    const isEdit = editOps.has(command.operationType ?? '');

    const selectedCodes =
      selectedParams.length > 0 ? selectedParams : Object.keys(parameters);

    const mergedParameters: Record<string, string | number | boolean> = {};
    for (const code of selectedCodes) {
      const v = parameters[code];
      if (isEdit) {
        if (v === undefined || v === null || v === '') continue;
      }
      mergedParameters[code] = v ?? '';
    }

    const commandEntry: Record<string, unknown> = {
      command_code: command.commandCode,
      parameters: mergedParameters,
    };
    if (selectedFields.length > 0) {
      commandEntry.selected_fields = selectedFields;
    }

    return {
      task_name: taskName || command.commandCode,
      device_sns: deviceSns,
      commands: [commandEntry],
    };
  }

  // paramPath 模式分两种：
  //   1) 选了命令 → 带 command_code，service 层会按命令的 RPCMethod 走（含 MOD/ADD 等）
  //   2) 没选命令 → 后端走"裸路径"分支（service.go ExecuteCommand），仅支持 LST/DSP
  // 两种都把非空 paths 传上去，后端按上下文决定。
  const filteredPaths = paramPaths.filter((path) => path.trim());
  const opType = operationType || 'LST';

  if (!command) {
    const previewPath = filteredPaths[0] || '';
    const fallbackName = previewPath
      ? `RAW ${opType} ${previewPath.length > 40 ? previewPath.slice(0, 40) + '...' : previewPath}`
      : `RAW ${opType}`;
    return {
      task_name: taskName || fallbackName,
      param_paths: filteredPaths,
      operation_type: opType,
      device_sns: deviceSns,
    };
  }

  return {
    task_name: taskName || command.commandCode,
    command_code: command.commandCode,
    param_paths: filteredPaths,
    operation_type: opType,
    device_sns: deviceSns,
  };
}

function outputTaskResults(t: ReturnType<typeof useT>, task: MMLTask, addOutput: (line: TerminalLine) => void) {
  for (const r of task.results) {
    addOutput({
      type: 'info',
      text: t('mml.console.deviceHeader', { sn: r.deviceSn }),
      timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
    });
    addOutput({
      type: r.result.success ? 'stdout' : 'stderr',
      text: r.result.rawOutput || (r.result.success ? t('mml.console.execSuccess') : t('mml.console.execFailed')),
      timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
    });
    if (r.result.executionTime) {
      addOutput({
        type: 'info',
        text: t('mml.console.executionTimeLabel', { time: String(r.result.executionTime) }),
        timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
      });
    }
  }
  addOutput({
    type: 'success',
    text: t('mml.console.executionCompleted', { success: String(task.successCount), failed: String(task.failedCount) }),
    timestamp: new Date().toLocaleTimeString(),
  });
}
