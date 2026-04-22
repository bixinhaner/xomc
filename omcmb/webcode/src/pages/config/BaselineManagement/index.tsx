import { useState, useMemo } from 'react';
import { Button, Input, Space, Tag, Tree, Typography, message } from 'antd';
import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useBaselineConfigs, useBaselineConfigById } from '@core/hooks/api/useConfig';
import { useT } from '@/hooks/useT';

interface BaselineParamRow extends Record<string, unknown> {
  id: string;
  paramCode: string;
  paramName: string;
  paramType: string;
  baselineValue: string;
  currentValue: string;
  isDiff: boolean;
}

const mockBaselineParams: BaselineParamRow[] = [
  { id: '1', paramCode: 'TX_POWER', paramName: '发射功率', paramType: 'number', baselineValue: '43', currentValue: '46', isDiff: true },
  { id: '2', paramCode: 'RS_POWER', paramName: '参考信号功率', paramType: 'number', baselineValue: '0', currentValue: '0', isDiff: false },
  { id: '3', paramCode: 'CELL_BW', paramName: '小区带宽', paramType: 'enum', baselineValue: '20MHz', currentValue: '20MHz', isDiff: false },
  { id: '4', paramCode: 'PCI', paramName: '物理小区标识', paramType: 'number', baselineValue: '10', currentValue: '15', isDiff: true },
  { id: '5', paramCode: 'TAC', paramName: '跟踪区码', paramType: 'number', baselineValue: '1001', currentValue: '1001', isDiff: false },
  { id: '6', paramCode: 'HO_THRESHOLD', paramName: '切换门限', paramType: 'number', baselineValue: '-110', currentValue: '-108', isDiff: true },
];

export default function BaselineManagement() {
  const t = useT();
  const [selectedBaselineId, setSelectedBaselineId] = useState<string>('');
  const [searchValue, setSearchValue] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data: baselinesData, isLoading: baselineLoading } = useBaselineConfigs({ page: 1, pageSize: 100 });
  const { data: baselineDetail } = useBaselineConfigById(selectedBaselineId);

  const mockBaselines = [
    { id: 'bl-001', baselineName: '4G-eNB标准基线v1.0', deviceType: 'eNB', version: '1.0', status: 'active' },
    { id: 'bl-002', baselineName: '4G-eNB标准基线v2.0', deviceType: 'eNB', version: '2.0', status: 'draft' },
    { id: 'bl-003', baselineName: '5G-gNB基础基线v1.0', deviceType: 'gNB', version: '1.0', status: 'active' },
  ];

  const baselines = (baselinesData?.items ?? mockBaselines) as typeof mockBaselines;

  const buildBaselineTree = (items: typeof mockBaselines): DataNode[] => {
    const grouped: Record<string, typeof items> = {};
    for (const b of items) {
      const key = b.deviceType;
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(b);
    }

    return Object.entries(grouped).map(([deviceType, group]) => ({
      title: deviceType,
      key: `group-${deviceType}`,
      children: group.map((b) => ({
        title: (
          <span>
            {b.baselineName}
            <Tag color={b.status === 'active' ? 'success' : b.status === 'draft' ? 'warning' : 'default'} style={{ marginLeft: 6, fontSize: 10 }}>
              {b.status === 'active' ? t('status.enabled') : b.status === 'draft' ? t('status.pending') : t('status.disabled')}
            </Tag>
          </span>
        ),
        key: b.id,
      })),
    }));
  };

  const treeData = buildBaselineTree(baselines);

  const tableSource = (baselineDetail?.params ?? mockBaselineParams) as unknown as BaselineParamRow[];

  const diffCount = tableSource.filter((p) => p.isDiff).length;

  const columns: DataTableColumn<BaselineParamRow>[] = useMemo(() => [
    { key: 'paramCode', title: t('config.paramCode'), dataIndex: 'paramCode', width: 160, mono: true, copyable: true },
    { key: 'paramName', title: t('config.paramName'), dataIndex: 'paramName', width: 160 },
    { key: 'paramType', title: t('config.paramType'), dataIndex: 'paramType', width: 100 },
    { key: 'baselineValue', title: t('config.baseline'), dataIndex: 'baselineValue', width: 120 },
    {
      key: 'currentValue',
      title: t('perf.value'),
      dataIndex: 'currentValue',
      width: 120,
      render: (val, record) => (
        <span style={{ color: record.isDiff ? '#faad14' : 'inherit', fontWeight: record.isDiff ? 600 : 400 }}>
          {val as string}
        </span>
      ),
    },
    {
      key: 'isDiff',
      title: t('config.compare'),
      dataIndex: 'isDiff',
      width: 80,
      render: (val) =>
        val ? <Tag color="warning">{t('status.failed')}</Tag> : <Tag color="success">{t('status.success')}</Tag>,
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: (_, record) =>
        record.isDiff ? (
          <Button
            type="link"
            size="small"
            onClick={() => void message.info(`${t('common.reset')}: ${record.paramCode as string}`)}
          >
            {t('common.reset')}
          </Button>
        ) : null,
    },
  ], [t]);

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('config.baseline')}</Typography.Text>
        <Button size="small" type="primary" icon={<PlusOutlined />}>{t('common.add')}</Button>
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
        {baselineLoading ? (
          <div style={{ padding: 16, color: '#8c8c8c', fontSize: 12 }}>{t('common.loading')}</div>
        ) : (
          <Tree
            treeData={treeData}
            onSelect={(keys) => {
              const key = keys[0] as string;
              if (key && !key.startsWith('group-')) {
                setSelectedBaselineId(key);
              }
            }}
            defaultExpandAll
            showLine
          />
        )}
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Space>
          <Typography.Text strong style={{ fontSize: 14 }}>
            {selectedBaselineId ? t('config.compare') : t('common.pleaseSelect')}
          </Typography.Text>
          {selectedBaselineId && diffCount > 0 && (
            <Tag color="warning">{diffCount} {t('status.failed')}</Tag>
          )}
        </Space>
        {selectedBaselineId && (
          <Space>
            <Button onClick={() => void message.info(t('common.exportInProgress'))}>{t('common.export')}</Button>
            <Button type="primary" onClick={() => void message.info(t('config.apply'))}>{t('config.apply')}</Button>
          </Space>
        )}
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <DataTable<BaselineParamRow>
          tableId="baseline-management"
          columns={columns}
          dataSource={tableSource}
          loading={false}
          rowKey="id"
          total={tableSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          scroll={{ x: 900 }}
        />
      </div>
    </TreeListPageLayout>
  );
}
