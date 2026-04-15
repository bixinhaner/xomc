import { useState, useCallback, useRef, useEffect } from 'react';
import type { TerminalLine } from '../types';
import type { ConsoleDevice } from '../types';
import type { MMLCommand, MMLTask } from '@/types/mml';
import { useExecuteMMLCommand, useMMLTaskPolling } from '@/hooks/api/useMML';

const TERMINAL_STATES = new Set(['completed', 'failed', 'cancelled']);

export function useCommandExecution() {
  const [outputLines, setOutputLines] = useState<TerminalLine[]>([]);
  const [isExecuting, setIsExecuting] = useState(false);
  const [pollingTaskId, setPollingTaskId] = useState<string | null>(null);
  const lastPolledStatus = useRef<string>('');

  const executeMutation = useExecuteMMLCommand();

  // 轮询任务状态
  const { data: polledTask } = useMMLTaskPolling(pollingTaskId, !!pollingTaskId);

  // 添加输出行
  const addOutput = useCallback((line: TerminalLine) => {
    setOutputLines((prev) => [...prev, line]);
  }, []);

  // 清空输出
  const clearOutput = useCallback(() => {
    setOutputLines([]);
    setPollingTaskId(null);
    lastPolledStatus.current = '';
  }, []);

  // 当轮询到任务完成时，输出结果
  useEffect(() => {
    if (!polledTask || !pollingTaskId) return;

    // 避免重复输出
    if (lastPolledStatus.current === polledTask.status) return;
    lastPolledStatus.current = polledTask.status;

    if (polledTask.status === 'completed') {
      addOutput({
        type: 'success',
        text: `任务完成 — 成功: ${polledTask.successCount}, 失败: ${polledTask.failedCount}`,
        timestamp: new Date().toLocaleTimeString(),
      });

      // 输出每个设备的结果
      if (polledTask.results && polledTask.results.length > 0) {
        addOutput({ type: 'info', text: '─'.repeat(50), timestamp: new Date().toLocaleTimeString() });
        for (const r of polledTask.results) {
          addOutput({
            type: r.result.success ? 'stdout' : 'stderr',
            text: `[${r.deviceSn}] ${r.result.success ? '成功' : '失败'}${r.result.rawOutput ? '\n' + r.result.rawOutput : ''}`,
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
        text: `任务失败 — ${polledTask.result || '未知错误'}`,
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
        text: '任务已取消',
        timestamp: new Date().toLocaleTimeString(),
      });
      setPollingTaskId(null);
      setIsExecuting(false);
    } else if (polledTask.status === 'running') {
      addOutput({
        type: 'info',
        text: `任务执行中... (已完成 ${polledTask.successCount + polledTask.failedCount}/${polledTask.totalDevices})`,
        timestamp: new Date().toLocaleTimeString(),
      });
    }
  }, [polledTask, pollingTaskId, addOutput]);

  // 执行命令
  const executeCommand = useCallback(async (
    devices: ConsoleDevice[],
    command: MMLCommand | null,
    params: Record<string, string | number | boolean>
  ) => {
    if (devices.length === 0) {
      addOutput({
        type: 'stderr',
        text: '错误：请先选择设备',
        timestamp: new Date().toLocaleTimeString(),
      });
      return;
    }

    if (!command) {
      addOutput({
        type: 'stderr',
        text: '错误：请先选择命令',
        timestamp: new Date().toLocaleTimeString(),
      });
      return;
    }

    setIsExecuting(true);

    // 添加命令开始标记
    addOutput({
      type: 'info',
      text: `执行命令: ${command.commandCode}`,
      timestamp: new Date().toLocaleTimeString(),
    });
    addOutput({
      type: 'info',
      text: `目标设备: ${devices.map((d) => d.sn).join(', ')}`,
      timestamp: new Date().toLocaleTimeString(),
    });
    addOutput({
      type: 'info',
      text: '─'.repeat(50),
      timestamp: new Date().toLocaleTimeString(),
    });

    try {
      // 调用 API 创建任务
      const task = await executeMutation.mutateAsync({
        commandCode: command.commandCode,
        deviceSns: devices.map((d) => d.sn),
        params,
      });

      addOutput({
        type: 'success',
        text: `任务已创建 (ID: ${task.id})`,
        timestamp: new Date().toLocaleTimeString(),
      });

      // 如果任务立即完成（mock 模式或同步执行），直接输出结果
      if (TERMINAL_STATES.has(task.status) && task.results && task.results.length > 0) {
        outputTaskResults(task, addOutput);
        addOutput({ type: 'info', text: '', timestamp: new Date().toLocaleTimeString() });
        setIsExecuting(false);
      } else {
        // 异步任务：启动轮询
        addOutput({
          type: 'info',
          text: `共 ${task.totalDevices || devices.length} 台设备待执行，等待结果...`,
          timestamp: new Date().toLocaleTimeString(),
        });
        lastPolledStatus.current = '';
        setPollingTaskId(task.id);
      }
    } catch (error) {
      addOutput({
        type: 'stderr',
        text: `执行失败: ${error instanceof Error ? error.message : '未知错误'}`,
        timestamp: new Date().toLocaleTimeString(),
      });
      setIsExecuting(false);
    }
  }, [executeMutation, addOutput]);

  // 下载输出
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

// 输出任务结果的辅助函数
function outputTaskResults(task: MMLTask, addOutput: (line: TerminalLine) => void) {
  for (const r of task.results) {
    addOutput({
      type: 'info',
      text: `--- 设备: ${r.deviceSn} ---`,
      timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
    });
    addOutput({
      type: r.result.success ? 'stdout' : 'stderr',
      text: r.result.rawOutput || (r.result.success ? '执行成功' : '执行失败'),
      timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
    });
    if (r.result.executionTime) {
      addOutput({
        type: 'info',
        text: `  执行耗时: ${r.result.executionTime}ms`,
        timestamp: r.result.timestamp ?? new Date().toLocaleTimeString(),
      });
    }
  }
  addOutput({
    type: 'success',
    text: `执行完成 — 成功: ${task.successCount}, 失败: ${task.failedCount}`,
    timestamp: new Date().toLocaleTimeString(),
  });
}
