import { useState } from 'react';
import { Button, Descriptions, Form, Input, InputNumber, List, Select, Space, Typography, message } from 'antd';
import { PlayCircleOutlined, SaveOutlined, SearchOutlined } from '@ant-design/icons';
import CommandConsoleLayout from '@/components/Layout/CommandConsoleLayout';
import TerminalOutput from '@/components/TerminalOutput';
import type { TerminalLine } from '@/components/TerminalOutput';
import { useMMLCommands, useExecuteMMLCommand } from '@/hooks/api/useMML';
import type { MMLCommand } from '@/types/mml';
import { useT } from '@/hooks/useT';

const COMMAND_CATEGORY_KEYS = [
  { key: 'BSC配置', labelKey: 'mml.category.bscConfig' },
  { key: '小区管理', labelKey: 'mml.category.cellMgmt' },
  { key: '邻区管理', labelKey: 'mml.category.neighborMgmt' },
  { key: '告警查询', labelKey: 'mml.category.alarmQuery' },
  { key: '性能采集', labelKey: 'mml.category.perfCollect' },
  { key: '传输管理', labelKey: 'mml.category.transMgmt' },
  { key: '版本管理', labelKey: 'mml.category.versionMgmt' },
];

const mockCommands: MMLCommand[] = [
  {
    id: '1',
    commandName: '查询小区信息',
    commandCode: 'LST CELL',
    category: '小区管理',
    description: '查询小区配置信息，支持按小区ID过滤',
    params: [
      { name: 'CELLID', type: 'number', required: false, description: '小区ID，不填则查询全部', minValue: 0, maxValue: 65535 },
    ],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '2',
    commandName: '设置小区激活状态',
    commandCode: 'SET CELL',
    category: '小区管理',
    description: '激活或去激活指定小区',
    params: [
      { name: 'CELLID', type: 'number', required: true, description: '小区ID', minValue: 0, maxValue: 65535 },
      { name: 'ACTTYPE', type: 'enum', required: true, description: '操作类型', options: [{ label: '激活', value: 0 }, { label: '去激活', value: 1 }] },
    ],
    productTypes: ['eNB'],
  },
  {
    id: '3',
    commandName: '查询邻区关系',
    commandCode: 'LST NCELL',
    category: '邻区管理',
    description: '查询邻区配置关系',
    params: [
      { name: 'LOCALCELLID', type: 'number', required: false, description: '本地小区ID' },
    ],
    productTypes: ['eNB', 'gNB'],
  },
  {
    id: '4',
    commandName: '查询基站状态',
    commandCode: 'LST BTSSTATE',
    category: 'BSC配置',
    description: '查询基站运行状态信息',
    params: [],
    productTypes: ['eNB', 'gNB'],
  },
];

export default function CommandMode() {
  const t = useT();
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [searchText, setSearchText] = useState('');
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [outputLines, setOutputLines] = useState<TerminalLine[]>([
    { type: 'info', text: `--- ${t('nav.config.commandMode')} ---`, timestamp: '2026-03-02 10:00:00' },
    { type: 'info', text: t('common.pleaseSelect'), timestamp: '2026-03-02 10:00:01' },
  ]);
  const [paramForm] = Form.useForm();

  const { data: commandsData } = useMMLCommands({ category: selectedCategory, keyword: searchText, page: 1, pageSize: 100 });
  const executeCommand = useExecuteMMLCommand();

  const commands = commandsData?.items ?? mockCommands;

  const filteredCommands = commands.filter((cmd) => {
    const matchCat = !selectedCategory || cmd.category === selectedCategory;
    const matchSearch = !searchText || cmd.commandName.includes(searchText) || cmd.commandCode.includes(searchText.toUpperCase());
    return matchCat && matchSearch;
  });

  const appendOutput = (lines: TerminalLine[]) => {
    setOutputLines((prev) => [...prev, ...lines]);
  };

  const handleExecute = () => {
    if (!selectedCommand) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    paramForm.validateFields().then((vals: Record<string, unknown>) => {
      const paramStr = Object.entries(vals)
        .filter(([, v]) => v !== undefined && v !== '')
        .map(([k, v]) => `${k}=${String(v)}`)
        .join(', ');
      const cmdLine = `${selectedCommand.commandCode}${paramStr ? `: ${paramStr}` : ''}`;

      appendOutput([
        { type: 'info', text: `> ${cmdLine}`, timestamp: new Date().toLocaleString() },
        { type: 'info', text: t('common.loading'), timestamp: new Date().toLocaleString() },
      ]);

      executeCommand.mutate(
        { commandCode: selectedCommand.commandCode, deviceSns: ['ENB00001'], params: vals as Record<string, string | number | boolean> },
        {
          onSuccess: (result) => {
            appendOutput([
              { type: result.success ? 'success' : 'stderr', text: result.rawOutput, timestamp: new Date().toLocaleString() },
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

  const handleClearOutput = () => {
    setOutputLines([{ type: 'info', text: `--- ${t('common.refresh')} ---`, timestamp: new Date().toLocaleString() }]);
  };

  const leftPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('nav.mml.commands')}</Typography.Text>
      </div>
      <div style={{ padding: '8px 12px' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ padding: '0 12px 8px' }}>
        <Select
          size="small"
          style={{ width: '100%' }}
          placeholder={t('common.pleaseSelect')}
          allowClear
          options={COMMAND_CATEGORY_KEYS.map((c) => ({ label: t(c.labelKey), value: c.key }))}
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
                padding: '6px 12px',
              }}
              onClick={() => { setSelectedCommand(cmd); paramForm.resetFields(); }}
            >
              <div>
                <div style={{ fontSize: 12, fontWeight: 500 }}>{cmd.commandName}</div>
                <div style={{ fontSize: 11, color: '#8c8c8c', fontFamily: 'monospace' }}>{cmd.commandCode}</div>
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
        <Typography.Text strong style={{ fontSize: 13 }}>{t('common.detail')}</Typography.Text>
      </div>
      {selectedCommand ? (
        <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
          <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label={t('table.name')}>{selectedCommand.commandName}</Descriptions.Item>
            <Descriptions.Item label={t('config.paramCode')}>
              <Typography.Text code>{selectedCommand.commandCode}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('perf.category')}>{selectedCommand.category}</Descriptions.Item>
            <Descriptions.Item label={t('table.description')}>{selectedCommand.description}</Descriptions.Item>
          </Descriptions>

          {selectedCommand.params.length > 0 && (
            <>
              <Typography.Text strong style={{ fontSize: 12 }}>{t('config.paramName')}</Typography.Text>
              <Form form={paramForm} layout="vertical" size="small" style={{ marginTop: 8 }}>
                {selectedCommand.params.map((param) => (
                  <Form.Item
                    key={param.name}
                    label={
                      <span>
                        {param.name}
                        {param.required && <span style={{ color: '#ff4d4f', marginLeft: 4 }}>*</span>}
                      </span>
                    }
                    name={param.name}
                    tooltip={param.description}
                    rules={param.required ? [{ required: true, message: `${t('common.placeholder')} ${param.name}` }] : []}
                  >
                    {param.type === 'enum' ? (
                      <Select
                        options={param.options?.map((o) => ({ label: String(o.label), value: o.value }))}
                        placeholder={`${t('common.pleaseSelect')} ${param.name}`}
                        allowClear={!param.required}
                      />
                    ) : param.type === 'number' ? (
                      <InputNumber
                        style={{ width: '100%' }}
                        min={param.minValue}
                        max={param.maxValue}
                        placeholder={param.description}
                      />
                    ) : (
                      <Input placeholder={param.description} />
                    )}
                  </Form.Item>
                ))}
              </Form>
            </>
          )}

          {selectedCommand.params.length === 0 && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.noData')}</Typography.Text>
          )}
        </div>
      ) : (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Typography.Text type="secondary">{t('common.pleaseSelect')}</Typography.Text>
        </div>
      )}
      <div style={{ padding: 12, borderTop: '1px solid #f0f0f0', display: 'flex', gap: 8 }}>
        <Button
          type="primary"
          icon={<PlayCircleOutlined />}
          onClick={handleExecute}
          loading={executeCommand.isPending}
          disabled={!selectedCommand}
          style={{ flex: 1 }}
        >
          {t('common.execute')}
        </Button>
        <Button icon={<SaveOutlined />} disabled={!selectedCommand}>{t('common.save')}</Button>
      </div>
    </div>
  );

  const rightPanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 16px 8px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('table.result')}</Typography.Text>
        <Button size="small" onClick={handleClearOutput}>{t('common.reset')}</Button>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
        <TerminalOutput
          lines={outputLines}
          height="100%"
          showTimestamp
          autoScroll
        />
      </div>
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
