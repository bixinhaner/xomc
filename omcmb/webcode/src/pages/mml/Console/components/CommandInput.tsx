import { useState, useCallback, useEffect } from 'react';
import { Button, Descriptions, Form, Input, InputNumber, Modal, Select, Space, Typography, Tabs } from 'antd';
import { PlayCircleOutlined, SaveOutlined, ReloadOutlined, PlusOutlined, MinusCircleOutlined, ExclamationCircleOutlined, WarningOutlined } from '@ant-design/icons';
import type { ConsoleDevice } from '../types';
import type { MMLCommand } from '@/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

// 操作类型选项
const OPERATION_TYPE_OPTIONS = [
  { label: 'LST - 查询', value: 'LST' },
  { label: 'MOD - 修改', value: 'MOD' },
  { label: 'ADD - 增加', value: 'ADD' },
  { label: 'RMV - 删除', value: 'RMV' },
];

// 需要确认的危险命令列表
const DANGEROUS_COMMANDS = [
  { pattern: /RST/i, name: '重启', description: '此操作将重启设备，设备会暂时断开连接' },
  { pattern: /FACTORYRESET/i, name: '恢复默认配置', description: '此操作将恢复设备出厂设置，所有配置将被清除' },
  { pattern: /CELLDEACTIVATE/i, name: '小区去激活', description: '此操作将去激活小区，可能影响网络服务' },
  { pattern: /RFCTXOFF/i, name: '关闭小区射频', description: '此操作将关闭小区射频发射，会影响无线信号' },
  { pattern: /COLDREBOOT/i, name: '冷重启', description: '此操作将执行设备冷重启，设备会完全断电重启' },
];

// 检查命令是否为危险命令
function checkDangerousCommand(commandCode: string): { isDangerous: boolean; command?: typeof DANGEROUS_COMMANDS[0] } {
  for (const cmd of DANGEROUS_COMMANDS) {
    if (cmd.pattern.test(commandCode)) {
      return { isDangerous: true, command: cmd };
    }
  }
  return { isDangerous: false };
}

interface ParamPath {
  id: string;
  path: string;
}

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
  const [activeTab, setActiveTab] = useState('control');

  // 参数路径配置状态
  const [operationType, setOperationType] = useState<string>('LST');
  const [paramPaths, setParamPaths] = useState<ParamPath[]>([{ id: '1', path: '' }]);

  // 确认弹窗状态
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [pendingCommand, setPendingCommand] = useState<typeof DANGEROUS_COMMANDS[0] | null>(null);

  // 当选中命令变化时，自动填入命令代码
  useEffect(() => {
    if (selectedCommand) {
      setConsoleInput(selectedCommand.commandCode);
    }
  }, [selectedCommand]);

  // 真正执行命令
  const doExecute = useCallback(() => {
    onExecute();
    setConsoleInput('');
    setConfirmModalOpen(false);
    setPendingCommand(null);
  }, [onExecute]);

  // 执行命令（带危险命令检查）
  const handleExecute = useCallback(() => {
    if (selectedDevices.length === 0) {
      return;
    }
    if (!selectedCommand) {
      return;
    }

    // 检查是否为危险命令
    const { isDangerous, command } = checkDangerousCommand(selectedCommand.commandCode);
    if (isDangerous && command) {
      setPendingCommand(command);
      setConfirmModalOpen(true);
      return;
    }

    // 非危险命令直接执行
    doExecute();
  }, [selectedDevices, selectedCommand, doExecute]);

  // 确认执行危险命令
  const handleConfirmExecute = useCallback(() => {
    doExecute();
  }, [doExecute]);

  // 取消执行
  const handleCancelExecute = useCallback(() => {
    setConfirmModalOpen(false);
    setPendingCommand(null);
  }, []);

  // 键盘快捷键
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleExecute();
    }
  }, [handleExecute]);

  // 添加参数路径
  const handleAddParamPath = useCallback(() => {
    setParamPaths((prev) => [
      ...prev,
      { id: String(Date.now()), path: '' },
    ]);
  }, []);

  // 删除参数路径
  const handleRemoveParamPath = useCallback((id: string) => {
    setParamPaths((prev) => {
      if (prev.length <= 1) return prev;
      return prev.filter((p) => p.id !== id);
    });
  }, []);

  // 更新参数路径
  const handleUpdateParamPath = useCallback((id: string, path: string) => {
    setParamPaths((prev) =>
      prev.map((p) => (p.id === id ? { ...p, path } : p))
    );
  }, []);

  // 是否是参数路径指定 Tab
  const isParamPathTab = activeTab === 'paramPath';

  // Tab 项配置
  const tabItems = [
    {
      key: 'control',
      label: '控制面板',
      children: (
        <div
          className="no-scrollbar"
          style={{
            height: '100%',
            overflow: 'auto',
            padding: 12,
          }}
        >
          {selectedCommand ? (
            selectedCommand.params.length > 0 ? (
              <Form layout="vertical" size="small">
                {selectedCommand.params.map((param) => (
                  <Form.Item
                    key={param.name}
                    label={
                      <span style={{ fontSize: 12 }}>
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
                fontSize: 12,
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
              fontSize: 12,
            }}>
              请先从左侧选择命令
            </div>
          )}
        </div>
      ),
    },
    {
      key: 'paramPath',
      label: '参数路径指定',
      children: (
        <div
          className="no-scrollbar"
          style={{
            height: '100%',
            overflow: 'auto',
            padding: 12,
          }}
        >
          {/* 操作类型 */}
          <Form layout="vertical" size="small">
            <Form.Item label={<span style={{ fontSize: 12 }}>操作类型</span>}>
              <Select
                options={OPERATION_TYPE_OPTIONS}
                value={operationType}
                onChange={setOperationType}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Form>

          {/* 参数路径列表 */}
          <div style={{ marginTop: 8 }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 12,
            }}>
              <Typography.Text strong style={{ fontSize: 12 }}>参数路径</Typography.Text>
              <Button
                size="small"
                type="dashed"
                icon={<PlusOutlined />}
                onClick={handleAddParamPath}
              >
                添加路径
              </Button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {paramPaths.map((item, index) => (
                <div key={item.id} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{
                    fontSize: 12,
                    color: '#8c8c8c',
                    width: 24,
                    flexShrink: 0,
                  }}>
                    {index + 1}.
                  </span>
                  <Input
                    size="small"
                    placeholder="例如: Device.Services.FAPService.1.CellConfig.1"
                    value={item.path}
                    onChange={(e) => handleUpdateParamPath(item.id, e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <Button
                    size="small"
                    type="text"
                    danger
                    icon={<MinusCircleOutlined />}
                    onClick={() => handleRemoveParamPath(item.id)}
                    disabled={paramPaths.length <= 1}
                  />
                </div>
              ))}
            </div>
          </div>

          {/* 帮助提示 */}
          <div style={{
            marginTop: 16,
            padding: 10,
            background: token.colorBgTextDisabled,
            borderRadius: 4,
            fontSize: 11,
            color: '#8c8c8c',
          }}>
            <div style={{ marginBottom: 4, fontWeight: 500 }}>提示</div>
            <div>• 参数路径支持 TR-069 参数树路径格式</div>
          </div>
        </div>
      ),
    },
  ];

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: token.colorBgContainer,
        borderRadius: 8,
        border: `1px solid ${token.colorBorderSecondary}`,
        overflow: 'hidden',
        boxShadow: '0 1px 4px rgba(0, 0, 0, 0.04)',
      }}
    >
      {/* 头部信息 */}
      <div
        style={{
          padding: '10px 14px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          background: `linear-gradient(180deg, ${token.colorBgLayout} 0%, ${token.colorBgContainer} 100%)`,
        }}
      >
        <Descriptions column={2} size="small">
          <Descriptions.Item
            label={
              <span style={{ fontSize: 11, color: token.colorTextSecondary }}>
                当前命令
              </span>
            }
          >
            {selectedCommand ? (
              <Typography.Text
                code
                style={{
                  fontSize: 12,
                  background: token.colorPrimaryBg,
                  border: `1px solid ${token.colorPrimaryBorder}`,
                  borderRadius: 4,
                  padding: '1px 6px',
                }}
              >
                {selectedCommand.commandCode}
              </Typography.Text>
            ) : (
              <span style={{ color: '#bfbfbf', fontSize: 11 }}>未选择</span>
            )}
          </Descriptions.Item>
          <Descriptions.Item
            label={
              <span style={{ fontSize: 11, color: token.colorTextSecondary }}>
                目标设备
              </span>
            }
          >
            <span
              style={{
                fontSize: 11,
                color: selectedDevices.length > 0 ? token.colorPrimary : '#bfbfbf',
                fontWeight: selectedDevices.length > 0 ? 500 : 400,
              }}
            >
              {selectedDevices.length > 0 ? `${selectedDevices.length} 台` : '未选择'}
            </span>
          </Descriptions.Item>
        </Descriptions>
      </div>

      {/* Tab 内容区 */}
      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <Tabs
          className="mml-console-tabs"
          activeKey={activeTab}
          onChange={setActiveTab}
          size="small"
          style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          tabBarStyle={{ padding: '0 12px', marginBottom: 0 }}
          items={tabItems}
        />
      </div>

      {/* 命令输入栏 - 只在控制面板 Tab 显示 */}
      {!isParamPathTab && (
        <div
          style={{
            padding: '10px 14px',
            borderTop: `1px solid ${token.colorBorderSecondary}`,
            display: 'flex',
            gap: 10,
            background: token.colorBgLayout,
          }}
        >
          <Input
            size="small"
            value={consoleInput}
            onChange={(e) => setConsoleInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入命令，多个命令用分号隔开"
            prefix={
              <span
                style={{
                  color: token.colorPrimary,
                  fontFamily: 'monospace',
                  fontWeight: 'bold',
                }}
              >
                &gt;
              </span>
            }
            style={{
              flex: 1,
              borderRadius: 4,
              fontFamily: "'SFMono-Regular', Consolas, monospace",
            }}
          />
          <Button
            size="small"
            type="primary"
            icon={<PlayCircleOutlined />}
            onClick={handleExecute}
            loading={loading}
            disabled={selectedDevices.length === 0 || !selectedCommand}
            style={{ borderRadius: 4, fontWeight: 500 }}
          >
            执行
          </Button>
        </div>
      )}

      {/* 底部操作栏 */}
      <div
        style={{
          padding: '10px 14px',
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          justifyContent: isParamPathTab ? 'flex-end' : 'space-between',
          gap: 8,
          background: token.colorBgLayout,
        }}
      >
        {!isParamPathTab && (
          <Space size={8}>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              <span style={{ color: token.colorPrimary }}>{selectedDevices.length}</span> 设备
              <span style={{ margin: '0 4px', color: token.colorBorder }}>·</span>
              {selectedCommand?.commandCode || (
                <span style={{ color: '#bfbfbf' }}>未选命令</span>
              )}
            </Typography.Text>
          </Space>
        )}
        <Space size={8}>
          {isParamPathTab && (
            <Button
              size="small"
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleExecute}
              loading={loading}
              disabled={selectedDevices.length === 0}
              style={{ borderRadius: 4 }}
            >
              执行
            </Button>
          )}
          {onReset && (
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={onReset}
              style={{ borderRadius: 4 }}
            >
              重置
            </Button>
          )}
          {onSaveScript && !isParamPathTab && (
            <Button
              size="small"
              icon={<SaveOutlined />}
              onClick={onSaveScript}
              style={{ borderRadius: 4 }}
            >
              保存脚本
            </Button>
          )}
        </Space>
      </div>

      {/* 危险命令确认弹窗 */}
      <Modal
        open={confirmModalOpen}
        onCancel={handleCancelExecute}
        onOk={handleConfirmExecute}
        okText="确认执行"
        cancelText="取消"
        okButtonProps={{
          danger: true,
          loading: loading,
        }}
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <WarningOutlined style={{ color: '#faad14', fontSize: 20 }} />
            <span>确认执行危险操作</span>
          </div>
        }
        width={420}
        centered
      >
        <div style={{ padding: '16px 0' }}>
          <div
            style={{
              padding: 16,
              background: token.colorWarningBg,
              borderRadius: 8,
              marginBottom: 16,
            }}
          >
            <Typography.Text strong style={{ fontSize: 14, color: token.colorWarningText }}>
              {pendingCommand?.name}
            </Typography.Text>
          </div>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {pendingCommand?.description}
          </Typography.Text>
          <div style={{ marginTop: 16 }}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              目标设备：<span style={{ color: token.colorPrimary, fontWeight: 500 }}>{selectedDevices.length} 台</span>
            </Typography.Text>
          </div>
          <div
            style={{
              marginTop: 16,
              padding: 12,
              background: token.colorBgLayout,
              borderRadius: 6,
              border: `1px solid ${token.colorBorderSecondary}`,
            }}
          >
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              请确认是否继续执行此操作
            </Typography.Text>
          </div>
        </div>
      </Modal>
    </div>
  );
}
