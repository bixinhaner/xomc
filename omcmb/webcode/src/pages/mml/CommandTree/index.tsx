import { useState, useMemo } from 'react';
import { Button, Input, Modal, Space, Tag, Tree, Typography, Descriptions } from 'antd';
import { SearchOutlined, PlayCircleOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useMMLCommands } from '@/hooks/api/useMML';
import type { MMLCommand } from '@/types/mml';
import { useT } from '@/hooks/useT';

interface CommandRow extends Record<string, unknown> {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  paramCount: number;
  productTypes: string[];
}

const CATEGORY_KEYS = [
  { titleKey: 'mml.category.cellMgmt', key: '1', label: '小区管理' },
  { titleKey: 'mml.category.neighborMgmt', key: '2', label: '邻区管理' },
  { titleKey: 'mml.category.bsMgmt', key: '3', label: '基站管理' },
  { titleKey: 'mml.category.alarmQuery', key: '4', label: '告警查询' },
  { titleKey: 'mml.category.perfCollect', key: '5', label: '性能采集' },
  { titleKey: 'mml.category.transMgmt', key: '6', label: '传输管理' },
  { titleKey: 'mml.category.versionMgmt', key: '7', label: '版本管理' },
];

const mockCommands: MMLCommand[] = [
  { id: '1', commandName: '查询小区信息', commandCode: 'LST CELL', category: '1', description: '查询小区配置信息，支持按小区ID过滤查询', params: [{ name: 'CELLID', type: 'number', required: false, description: '小区ID' }], productTypes: ['eNB', 'gNB'] },
  { id: '2', commandName: '激活小区', commandCode: 'ACT CELL', category: '1', description: '激活指定小区，使其进入服务状态', params: [{ name: 'CELLID', type: 'number', required: true, description: '小区ID' }], productTypes: ['eNB'] },
  { id: '3', commandName: '去激活小区', commandCode: 'DEA CELL', category: '1', description: '去激活指定小区，退出服务状态', params: [{ name: 'CELLID', type: 'number', required: true, description: '小区ID' }], productTypes: ['eNB'] },
  { id: '4', commandName: '查询邻区', commandCode: 'LST NCELL', category: '2', description: '查询邻区配置关系列表', params: [{ name: 'LOCALCELLID', type: 'number', required: false, description: '本地小区ID' }], productTypes: ['eNB', 'gNB'] },
  { id: '5', commandName: '添加邻区', commandCode: 'ADD NCELL', category: '2', description: '添加邻区关系', params: [{ name: 'LOCALCELLID', type: 'number', required: true, description: '本地小区ID' }, { name: 'CELLID', type: 'number', required: true, description: '邻小区ID' }], productTypes: ['eNB'] },
  { id: '6', commandName: '查询基站状态', commandCode: 'LST BTSSTATE', category: '3', description: '查询基站运行状态信息', params: [], productTypes: ['eNB', 'gNB'] },
  { id: '7', commandName: '查询活动告警', commandCode: 'LST ALMAF', category: '4', description: '查询当前活动告警列表', params: [{ name: 'ALMFAULTID', type: 'number', required: false, description: '告警ID' }, { name: 'SEVERITY', type: 'enum', required: false, description: '告警级别', options: [{ label: '严重', value: 1 }, { label: '主要', value: 2 }] }], productTypes: ['eNB', 'gNB'] },
  { id: '8', commandName: '复位基站', commandCode: 'RST BTS', category: '3', description: '对基站进行软复位操作', params: [{ name: 'RSTTYPE', type: 'enum', required: true, description: '复位类型', options: [{ label: '软复位', value: 0 }, { label: '硬复位', value: 1 }] }], productTypes: ['eNB', 'gNB'] },
];

export default function CommandTree() {
  const t = useT();
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [searchValue, setSearchValue] = useState('');
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedCommand, setSelectedCommand] = useState<MMLCommand | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // category value → 中文 label 映射
  const catLabelMap = useMemo(
    () => new Map(CATEGORY_KEYS.map((c) => [c.key, c.label])),
    []
  );
  const getCatLabel = (val: string) => catLabelMap.get(val) || val;

  const commandCategories = useMemo(() => [
    {
      title: t('mml.category.all'),
      key: 'all',
      children: CATEGORY_KEYS.map((c) => ({ title: t(c.titleKey), key: c.key })),
    },
  ], [t]);

  const { data, isLoading, refetch } = useMMLCommands({
    keyword: searchValue,
    category: selectedCategory || undefined,
    page,
    pageSize,
  });

  const allCommands = data?.items ?? mockCommands;

  const filteredCommands = allCommands.filter((c) => {
    if (selectedCategory && selectedCategory !== 'all') {
      return c.category === selectedCategory;
    }
    return true;
  });

  const tableSource: CommandRow[] = filteredCommands.map((c) => ({
    id: c.id,
    commandName: c.commandName,
    commandCode: c.commandCode,
    category: c.category,
    description: c.description,
    paramCount: c.params.length,
    productTypes: c.productTypes,
  }));

  const columns: DataTableColumn<CommandRow>[] = useMemo(() => [
    { key: 'commandName', title: t('table.name'), dataIndex: 'commandName', width: 180 },
    { key: 'commandCode', title: t('config.paramCode'), dataIndex: 'commandCode', width: 150, mono: true, copyable: true },
    { key: 'category', title: t('perf.category'), dataIndex: 'category', width: 110, render: (val: string) => getCatLabel(val) },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 280, ellipsis: true },
    { key: 'paramCount', title: t('table.total'), dataIndex: 'paramCount', width: 90 },
    {
      key: 'productTypes',
      title: t('device.productType'),
      dataIndex: 'productTypes',
      width: 130,
      render: (val) =>
        (val as string[]).map((tp) => (
          <Tag key={tp} color="blue" style={{ marginRight: 2 }}>{tp}</Tag>
        )),
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => {
              const cmd = allCommands.find((c) => c.id === record.id);
              if (cmd) { setSelectedCommand(cmd); setDetailVisible(true); }
            }}
          >
            {t('common.detail')}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<PlayCircleOutlined />}
            onClick={() => {
              const cmd = allCommands.find((c) => c.id === record.id);
              if (cmd) setSelectedCommand(cmd);
            }}
          >
            {t('common.execute')}
          </Button>
        </Space>
      ),
    },
  ], [t, allCommands]);

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('perf.category')}</Typography.Text>
      </div>
      <div style={{ padding: '0 12px 8px' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
        <Tree
          treeData={commandCategories as DataNode[]}
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedCategory(key && key !== 'all' ? key : '');
          }}
          defaultExpandAll
          showLine
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0' }}>
        <Typography.Text strong style={{ fontSize: 14 }}>
          {selectedCategory ? getCatLabel(selectedCategory) : t('nav.mml.commands')}
          <Tag style={{ marginLeft: 8 }} color="blue">{tableSource.length}</Tag>
        </Typography.Text>
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <DataTable<CommandRow>
          tableId="command-tree"
          columns={columns}
          dataSource={tableSource}
          loading={isLoading}
          rowKey="id"
          total={data?.total ?? tableSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          scroll={{ x: 1100 }}
        />
      </div>

      <Modal
        title={`${t('common.detail')}: ${selectedCommand?.commandCode ?? ''}`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={600}
      >
        {selectedCommand && (
          <div>
            <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label={t('table.name')}>{selectedCommand.commandName}</Descriptions.Item>
              <Descriptions.Item label={t('config.paramCode')}>
                <Typography.Text code>{selectedCommand.commandCode}</Typography.Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('perf.category')}>{getCatLabel(selectedCommand.category)}</Descriptions.Item>
              <Descriptions.Item label={t('device.productType')}>
                {selectedCommand.productTypes.map((tp) => <Tag key={tp} color="blue">{tp}</Tag>)}
              </Descriptions.Item>
              <Descriptions.Item label={t('table.description')}>{selectedCommand.description}</Descriptions.Item>
            </Descriptions>

            {selectedCommand.params.length > 0 && (
              <>
                <Typography.Text strong style={{ fontSize: 13 }}>{t('config.paramName')}</Typography.Text>
                <table style={{ width: '100%', marginTop: 8, borderCollapse: 'collapse', fontSize: 12 }}>
                  <thead>
                    <tr style={{ background: '#fafafa' }}>
                      {[t('table.name'), t('table.type'), t('table.status'), t('table.description')].map((h) => (
                        <th key={h} style={{ border: '1px solid #f0f0f0', padding: '6px 8px', textAlign: 'left' }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {selectedCommand.params.map((p) => (
                      <tr key={p.name}>
                        <td style={{ border: '1px solid #f0f0f0', padding: '5px 8px', fontFamily: 'monospace' }}>{p.name}</td>
                        <td style={{ border: '1px solid #f0f0f0', padding: '5px 8px' }}>{p.type}</td>
                        <td style={{ border: '1px solid #f0f0f0', padding: '5px 8px' }}>
                          {p.required ? <Tag color="red">{t('common.confirm')}</Tag> : <Tag>{t('common.more')}</Tag>}
                        </td>
                        <td style={{ border: '1px solid #f0f0f0', padding: '5px 8px' }}>{p.description}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </>
            )}
            {selectedCommand.params.length === 0 && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>{t('common.noData')}</Typography.Text>
            )}
          </div>
        )}
      </Modal>
    </TreeListPageLayout>
  );
}
