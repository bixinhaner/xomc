import { useState, useCallback, useEffect } from 'react';
import { Button, Descriptions, Form, Input, InputNumber, Select, Space, Typography } from 'antd';
import { PlayCircleOutlined, SaveOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ConsoleDevice } from '../types';
import type { MMLCommand } from '@/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

interface CommandInputProps {
  selectedDevices: ConsoleDevice[];
  selectedCommand: MMLCommand | null;
  paramValues: Record<string, string | number | boolean>;
  onParamChange: (values: Record<string, string | number | boolean>) => void;
  onExecute: () => void;
  onSaveScript?: () => void;
  onReset?: () => void;
  loading?: boolean;
}

export default function CommandInput({
  selectedDevices,
  selectedCommand,
  paramValues,
  onParamChange,
  onExecute,
  onSaveScript,
  onReset,
  loading,
}: CommandInputProps) {
  const t = useT();
  const token = useThemeToken();
  const [consoleInput, setConsoleInput] = useState('');

  // 当选中命令变化时，自动填入命令代码
  useEffect(() => {
    if (selectedCommand) {
      setConsoleInput(selectedCommand.commandCode);
    }
  }, [selectedCommand]);

  // 执行命令
  const handleExecute = useCallback(() => {
    if (selectedDevices.length === 0) {
      return;
    }
    if (!selectedCommand) {
      return;
    }
    onExecute();
    setConsoleInput('');
  }, [selectedDevices, selectedCommand, onExecute]);

  // 键盘快捷键
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleExecute();
    }
  }, [handleExecute]);

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100%',
      background: token.colorBgContainer,
      borderRadius: 6,
      border: `1px solid ${token.colorBorderSecondary}`,
      overflow: 'hidden',
    }}>
      {/* 头部信息 */}
      <div style={{
        padding: '8px 12px',
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
      }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="命令">
            {selectedCommand ? (
              <Typography.Text code style={{ fontSize: 11 }}>
                {selectedCommand.commandCode}
              </Typography.Text>
            ) : (
              <span style={{ color: '#8c8c8c' }}>未选择</span>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="设备">
            <span style={{ fontSize: 11 }}>
              {selectedDevices.length > 0 ? `${selectedDevices.length} 台` : '未选择'}
            </span>
          </Descriptions.Item>
        </Descriptions>
      </div>

      {/* 参数配置区 */}
      <div className="no-scrollbar" style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        {selectedCommand ? (
          selectedCommand.params.length > 0 ? (
            <Form layout="vertical" size="small">
              {selectedCommand.params.map((param) => (
                <Form.Item
                  key={param.name}
                  label={
                    <span style={{ fontSize: 11 }}>
                      {param.name}
                      {param.required && <span style={{ color: '#ff4d4f', marginLeft: 2 }}>*</span>}
                    </span>
                  }
                  tooltip={param.description}
                >
                  {param.type === 'enum' ? (
                    <Select
                      allowClear={!param.required}
                      options={param.options?.map((o) => ({
                        label: String(o.label),
                        value: o.value,
                      }))}
                      value={paramValues[param.name] as string | number | undefined}
                      onChange={(val) =>
                        onParamChange({ ...paramValues, [param.name]: val })
                      }
                    />
                  ) : param.type === 'number' ? (
                    <InputNumber
                      style={{ width: '100%' }}
                      min={param.minValue}
                      max={param.maxValue}
                      value={paramValues[param.name] as number | undefined}
                      onChange={(val) =>
                        onParamChange({ ...paramValues, [param.name]: val ?? '' })
                      }
                    />
                  ) : (
                    <Input
                      value={paramValues[param.name] as string | undefined}
                      onChange={(e) =>
                        onParamChange({ ...paramValues, [param.name]: e.target.value })
                      }
                    />
                  )}
                </Form.Item>
              ))}
            </Form>
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#8c8c8c',
              fontSize: 11,
            }}>
              该命令无需配置参数
            </div>
          )
        ) : (
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            height: '100%',
            color: '#8c8c8c',
            fontSize: 11,
          }}>
            请先从左侧选择命令
          </div>
        )}
      </div>

      {/* 命令输入栏 */}
      <div style={{
        padding: '8px 12px',
        borderTop: `1px solid ${token.colorBorderSecondary}`,
        display: 'flex',
        gap: 8,
      }}>
        <Input
          size="small"
          value={consoleInput}
          onChange={(e) => setConsoleInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="输入命令... (Ctrl+Enter 执行)"
          prefix={<span style={{ color: '#52c41a', fontFamily: 'monospace' }}>&gt;</span>}
          style={{ flex: 1 }}
        />
        <Button
          size="small"
          type="primary"
          icon={<PlayCircleOutlined />}
          onClick={handleExecute}
          loading={loading}
          disabled={selectedDevices.length === 0 || !selectedCommand}
        >
          执行
        </Button>
      </div>

      {/* 底部操作栏 */}
      <div style={{
        padding: '8px 12px',
        borderTop: `1px solid ${token.colorBorderSecondary}`,
        display: 'flex',
        justifyContent: 'space-between',
      }}>
        <Space size={8}>
          <Typography.Text type="secondary" style={{ fontSize: 10 }}>
            {selectedDevices.length} 设备 · {selectedCommand?.commandCode || '未选命令'}
          </Typography.Text>
        </Space>
        <Space size={4}>
          {onReset && (
            <Button size="small" icon={<ReloadOutlined />} onClick={onReset}>
              重置
            </Button>
          )}
          {onSaveScript && (
            <Button size="small" icon={<SaveOutlined />} onClick={onSaveScript}>
              保存脚本
            </Button>
          )}
        </Space>
      </div>
    </div>
  );
}
