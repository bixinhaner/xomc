import { useState, useCallback } from 'react';
import type { TerminalLine } from '../types';
import type { ConsoleDevice } from '../types';
import type { MMLCommand } from '@/types/mml';
import { useExecuteMMLCommand } from '@/hooks/api/useMML';

export function useCommandExecution() {
  const [outputLines, setOutputLines] = useState<TerminalLine[]>([]);
  const [isExecuting, setIsExecuting] = useState(false);
  const executeMutation = useExecuteMMLCommand();

  // 添加输出行
  const addOutput = useCallback((line: TerminalLine) => {
    setOutputLines((prev) => [...prev, line]);
  }, []);

  // 清空输出
  const clearOutput = useCallback(() => {
    setOutputLines([]);
  }, []);

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
      // 调用 API 创建任务（所有设备一次请求）
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
      addOutput({
        type: 'info',
        text: `共 ${task.totalDevices} 台设备待执行`,
        timestamp: new Date().toLocaleTimeString(),
      });
      addOutput({
        type: 'info',
        text: '',
        timestamp: new Date().toLocaleTimeString(),
      });

      // 逐设备显示信息
      for (const device of devices) {
        addOutput({
          type: 'info',
          text: `--- 设备: ${device.sn} ---`,
          timestamp: new Date().toLocaleTimeString(),
        });
        addOutput({
          type: 'stdout',
          text: `  设备名称: ${device.name}`,
          timestamp: new Date().toLocaleTimeString(),
        });
        addOutput({
          type: 'stdout',
          text: `  设备类型: ${device.type}`,
          timestamp: new Date().toLocaleTimeString(),
        });
        addOutput({
          type: 'stdout',
          text: `  产品型号: ${device.productType}`,
          timestamp: new Date().toLocaleTimeString(),
        });
        addOutput({
          type: 'stdout',
          text: `  运行状态: ${device.status === 'online' ? '在线' : device.status === 'alarm' ? '告警' : '离线'}`,
          timestamp: new Date().toLocaleTimeString(),
        });
        addOutput({
          type: 'info',
          text: '',
          timestamp: new Date().toLocaleTimeString(),
        });
      }

      // 执行完成
      addOutput({
        type: 'success',
        text: `任务已提交，共 ${devices.length} 台设备`,
        timestamp: new Date().toLocaleTimeString(),
      });
    } catch (error) {
      addOutput({
        type: 'stderr',
        text: `执行失败: ${error instanceof Error ? error.message : '未知错误'}`,
        timestamp: new Date().toLocaleTimeString(),
      });
    } finally {
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
