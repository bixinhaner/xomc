import { useState } from 'react';
import { Button, Card, Descriptions, Form, Input, InputNumber, List, Select, Space, Tabs, Tag, Typography, message } from 'antd';
import { SearchOutlined, PlayCircleOutlined, SaveOutlined } from '@ant-design/icons';
import CommandConsoleLayout from '@/components/Layout/CommandConsoleLayout';
import TerminalOutput from '@/components/TerminalOutput';
import type { TerminalLine } from '@/components/TerminalOutput';
import { useMMLCommands, useExecuteMMLCommand } from '@/hooks/api/useMML';
import type { MMLCommand } from '@/types/mml';
import { useT } from '@/hooks/useT';

const DEVICE_LIST = [
  { sn: 'ENB00001', name: '北京朝阳基站01', type: 'eNB', status: 'online' },
  { sn: 'ENB00002', name: '北京海淀基站01', type: 'eNB', status: 'online' },
  { sn: 'ENB00003', name: '上海浦东基站01', type: 'eNB', status: 'alarm' },
  { sn: 'GNB00001', name: '北京5G基站01', type: 'gNB', status: 'online' },
  { sn: 'GNB00002', name: '北京5G基站02', type: 'gNB', status: 'offline' },
];

const MOCK_COMMANDS: MMLCommand[] = [
  {
    id: '1',
    commandName: '查询小区信息',
    commandCode: 'LST CELL',
    category: '小区管理',
    description: '查询小区配置信息',
    params: [{ name: 'CELLID', type: 'number', required: false, description: '小区ID，不填则查询全部', minValue: 0, maxValue: 65535 }],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '2',
    commandName: '激活小区',
    commandCode: 'ACT CELL',
    category: '小区管理',
    description: '激活指定小区',
    params: [{ name: 'CELLID', type: 'number', required: true, description: '小区ID', minValue: 0, maxValue: 65535 }],
    productTypes: ['eNB'],
  },
  {
    id: '3',
    commandName: '查询基站状态',
    commandCode: 'LST BTSSTATE',
    category: '基站管理',
    description: '查询基站运行状态信息',
    params: [],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '4',
    commandName: '查询邻区',
    commandCode: 'LST NCELL',
    category: '邻区管理',
    description: '查询邻区配置关系列表',
    params: [{ name: 'LOCALCELLID', type: 'number', required: false, description: '本地小区ID' }],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '5',
    commandName: '查询告警',
    commandCode: 'LST ALMAF',
    category: '告警查询',
    description: '查询当前活动告警',
    params: [
      {
        name: 'ALMFAULTID',
        type: 'number',
        required: false,
        description: '告警ID，不填则查全部',
      },
      {
        name: 'SEVERITY',
        type: 'enum',
        required: false,
        description: '告警级别',
        options: [
          { label: '严重', value: 1 },
          { label: '主要', value: 2 },
          { label: '次要', value: 3 },
          { label: '警告', value: 4 },
        ],
      },
    ],
    productTypes: ['eNB', 'gNB'],
  },
];

const STATUS_COLORS: Record<string, string> = {
  online: '#52c41a',
  offline: '#d9d9d9',
  alarm: '#fa8c16',
};

export default function MMLConsole() {
  const t = useT();
  const [deviceSearch, setDeviceSearch] = useState('');
  const [selectedDevice, setSelectedDevice] = useState<string | null>(null);
  const [commandSearch, setCommandSearch] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('');
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [outputLines, setOutputLines] = useState<TerminalLine[]>([
    { type: 'info', text: `--- ${t('nav.mml.console')} ---`, timestamp: new Date().toLocaleString() },
    { type: 'info', text: t('common.pleaseSelect'), timestamp: new Date().toLocaleString() },
  ]);
  const [paramForm] = Form.useForm();
  const [consoleInput, setConsoleInput] = useState('');

  const { data: cmdData } = useMMLCommands({ keyword: commandSearch, category: selectedCategory, page: 1, pageSize: 100 });
  const executeCommand = useExecuteMMLCommand();

  const commands = cmdData?.items ?? MOCK_COMMANDS;
  const categories = [...new Set(commands.map((c) => c.category))];

  const filteredDevices = DEVICE_LIST.filter((d) =>
    !deviceSearch || d.sn.includes(deviceSearch) || d.name.includes(deviceSearch),
  );

  const filteredCommands = commands.filter((c) => {
    const matchCat = !selectedCategory || c.category === selectedCategory;
    const matchSearch = !commandSearch || c.commandName.includes(commandSearch) || c.commandCode.includes(commandSearch.toUpperCase());
    return matchCat && matchSearch;
  });

  const appendOutput = (lines: TerminalLine[]) => {
    setOutputLines((prev) => [...prev, ...lines]);
  };

  const handleExecute = () => {
    if (!selectedDevice) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    if (!selectedCommand) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    paramForm.validateFields().then((vals: Record<string, unknown>) => {
      const paramStr = Object.entries(vals)
        .filter(([, v]) => v !== undefined && v !== '')
        .map(([k, v]) => `${k}=${String(v)}`)
        .join(', ');
      const cmdStr = `${selectedCommand.commandCode}${paramStr ? `: ${paramStr}` : ''}`;

      appendOutput([
        { type: 'info', text: `[${selectedDevice}] > ${cmdStr}`, timestamp: new Date().toLocaleString() },
      ]);

      executeCommand.mutate(
        {
          commandCode: selectedCommand.commandCode,
          deviceSns: [selectedDevice],
          params: vals as Record<string, string | number | boolean>,
        },
        {
          onSuccess: (result) => {
            appendOutput([
              { type: result.success ? 'stdout' : 'stderr', text: result.rawOutput, timestamp: new Date().toLocaleString() },
              { type: 'info', text: `${t('status.success')}, ${result.executionTime}ms`, timestamp: new Date().toLocaleString() },
            ]);
          },
          onError: () => {
            appendOutput([{ type: 'stderr', text: t('status.failed'), timestamp: new Date().toLocaleString() }]);
          },
        },
      );
    }).catch(() => undefined);
  };

  const handleConsoleInputExecute = () => {
    if (!consoleInput.trim()) return;
    if (!selectedDevice) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    appendOutput([
      { type: 'info', text: `[${selectedDevice}] > ${consoleInput}`, timestamp: new Date().toLocaleString() },
    ]);
    const parts = consoleInput.trim().split(':');
    const commandCode = parts[0].trim();
    executeCommand.mutate(
      { commandCode, deviceSns: [selectedDevice] },
      {
        onSuccess: (result) => {
          appendOutput([
            { type: result.success ? 'stdout' : 'stderr', text: result.rawOutput, timestamp: new Date().toLocaleString() },
          ]);
        },
        onError: () => {
          appendOutput([{ type: 'stderr', text: `${t('status.failed')}: ${commandCode}`, timestamp: new Date().toLocaleString() }]);
        },
      },
    );
    setConsoleInput('');
  };

  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Device Selector */}
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 12 }}>{t('device.name')}</Typography.Text>
        <Input
          size="small"
          style={{ marginTop: 6 }}
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={deviceSearch}
          onChange={(e) => setDeviceSearch(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ maxHeight: 200, overflow: 'auto', borderBottom: '1px solid #f0f0f0' }}>
        <List
          size="small"
          dataSource={filteredDevices}
          renderItem={(device) => (
            <List.Item
              style={{
                cursor: 'pointer',
                background: selectedDevice === device.sn ? '#e6f4ff' : 'transparent',
                padding: '4px 12px',
              }}
              onClick={() => setSelectedDevice(device.sn)}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, width: '100%' }}>
                <div
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: STATUS_COLORS[device.status] ?? '#d9d9d9',
                    flexShrink: 0,
                  }}
                />
                <div>
                  <div style={{ fontSize: 11, fontWeight: 500 }}>{device.sn}</div>
                  <div style={{ fontSize: 10, color: '#8c8c8c' }}>{device.name}</div>
                </div>
                <Tag style={{ marginLeft: 'auto', fontSize: 10 }}>{device.type}</Tag>
              </div>
            </List.Item>
          )}
        />
      </div>

      {/* Command Tree */}
      <div style={{ padding: '8px 12px 4px' }}>
        <Typography.Text strong style={{ fontSize: 12 }}>{t('nav.mml.commands')}</Typography.Text>
        <Input
          size="small"
          style={{ marginTop: 6, marginBottom: 4 }}
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={commandSearch}
          onChange={(e) => setCommandSearch(e.target.value)}
          allowClear
        />
        <Select
          size="small"
          style={{ width: '100%', marginBottom: 4 }}
          placeholder={t('common.pleaseSelect')}
          allowClear
          options={categories.map((c) => ({ label: c, value: c }))}
          value={selectedCategory || undefined}
          onChange={(val) => setSelectedCategory(val ?? '')}
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <List
          size="small"
          dataSource={filteredCommands}
          renderItem={(cmd) => (
            <List.Item
              style={{
                cursor: 'pointer',
                background: selectedCommand?.id === cmd.id ? '#e6f4ff' : 'transparent',
                padding: '5px 12px',
              }}
              onClick={() => { setSelectedCommand(cmd); paramForm.resetFields(); }}
            >
              <div>
                <div style={{ fontSize: 11, fontWeight: 500 }}>{cmd.commandName}</div>
                <div style={{ fontSize: 10, color: '#8c8c8c', fontFamily: 'monospace' }}>{cmd.commandCode}</div>
              </div>
            </List.Item>
          )}
        />
      </div>
    </div>
  );

  const centerPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 12 }}>{t('common.detail')}</Typography.Text>
      </div>
      {selectedCommand ? (
        <>
          <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
            <Descriptions column={1} size="small" style={{ marginBottom: 12 }}>
              <Descriptions.Item label={t('nav.mml.commands')}>
                <Typography.Text code style={{ fontSize: 11 }}>{selectedCommand.commandCode}</Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('table.description')}>
                <span style={{ fontSize: 11 }}>{selectedCommand.description}</span>
              </Descriptions.Item>
            </Descriptions>

            {selectedCommand.params.length > 0 ? (
              <Form form={paramForm} layout="vertical" size="small">
                {selectedCommand.params.map((param) => (
                  <Form.Item
                    key={param.name}
                    label={
                      <span style={{ fontSize: 11 }}>
                        {param.name}
                        {param.required && <span style={{ color: '#ff4d4f', marginLeft: 2 }}>*</span>}
                      </span>
                    }
                    name={param.name}
                    tooltip={param.description}
                    rules={param.required ? [{ required: true }] : []}
                  >
                    {param.type === 'enum' ? (
                      <Select size="small" options={param.options?.map((o) => ({ label: String(o.label), value: o.value }))} allowClear={!param.required} />
                    ) : param.type === 'number' ? (
                      <InputNumber size="small" style={{ width: '100%' }} min={param.minValue} max={param.maxValue} />
                    ) : (
                      <Input size="small" />
                    )}
                  </Form.Item>
                ))}
              </Form>
            ) : (
              <Typography.Text type="secondary" style={{ fontSize: 11 }}>{t('common.noData')}</Typography.Text>
            )}
          </div>
          <div style={{ padding: '8px 12px', borderTop: '1px solid #f0f0f0', display: 'flex', gap: 6 }}>
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={handleExecute}
              loading={executeCommand.isPending}
              style={{ flex: 1 }}
            >
              {t('common.execute')}
            </Button>
            <Button size="small" icon={<SaveOutlined />}>{t('common.save')}</Button>
          </div>
        </>
      ) : (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.pleaseSelect')}</Typography.Text>
        </div>
      )}
    </div>
  );

  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Tabs
        size="small"
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        items={[
          {
            key: 'console',
            label: t('nav.mml.console'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
                <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
                  <TerminalOutput
                    lines={outputLines}
                    height="100%"
                    showTimestamp
                    autoScroll
                    style={{ minHeight: 300 }}
                  />
                </div>
                <div style={{ display: 'flex', gap: 6, padding: '8px 4px 4px' }}>
                  <Input
                    size="small"
                    value={consoleInput}
                    onChange={(e) => setConsoleInput(e.target.value)}
                    onPressEnter={handleConsoleInputExecute}
                    placeholder={t('common.placeholder')}
                    prefix={<span style={{ color: '#52c41a', fontFamily: 'monospace' }}>&gt;</span>}
                  />
                  <Button size="small" type="primary" onClick={handleConsoleInputExecute} loading={executeCommand.isPending}>
                    {t('common.execute')}
                  </Button>
                </div>
              </div>
            ),
          },
          {
            key: 'script',
            label: t('nav.mml.script'),
            children: (
              <div style={{ padding: 12 }}>
                <div style={{ marginBottom: 8 }}>
                  <Select
                    size="small"
                    placeholder={t('common.pleaseSelect')}
                    style={{ width: 140, marginRight: 8 }}
                    options={[
                      { label: 'LST', value: 'LST' },
                      { label: 'SET', value: 'SET' },
                      { label: 'ADD', value: 'ADD' },
                      { label: 'RMV', value: 'RMV' },
                    ]}
                  />
                  <Input size="small" placeholder={t('common.placeholder')} style={{ width: 200 }} />
                </div>
                <Input.TextArea
                  rows={8}
                  placeholder={t('common.placeholder')}
                  style={{ fontFamily: 'monospace', fontSize: 12 }}
                />
                <div style={{ marginTop: 8 }}>
                  <Button size="small" type="primary" icon={<PlayCircleOutlined />}>{t('common.execute')}</Button>
                </div>
              </div>
            ),
          },
        ]}
        tabBarStyle={{ padding: '0 12px', marginBottom: 0 }}
        tabBarExtraContent={
          <Button size="small" onClick={() => setOutputLines([{ type: 'info', text: `--- ${t('common.reset')} ---`, timestamp: new Date().toLocaleString() }])}>
            {t('common.reset')}
          </Button>
        }
      />
    </div>
  );

  return (
    <CommandConsoleLayout
      left={leftPanel}
      center={centerPanel}
      right={rightPanel}
    />
  );
}
