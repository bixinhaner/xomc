import { useState, useMemo } from 'react';
import { Button, Input, Tree, Typography, Space, Modal, Form, InputNumber, message } from 'antd';
import { EditOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useConfigParams } from '@core/hooks/api/useConfig';
import { useT } from '@/hooks/useT';

interface ParamClassRow extends Record<string, unknown> {
  id: string;
  paramCode: string;
  paramName: string;
  paramType: string;
  currentValue: string | number;
  defaultValue: string | number;
  unit: string;
  description: string;
}

const CLASS_TREE: DataNode[] = [
  {
    title: '全部分类',
    key: 'all',
    children: [
      {
        title: '射频参数',
        key: 'rf',
        children: [
          { title: '功率参数', key: 'rf-power' },
          { title: '天线参数', key: 'rf-antenna' },
          { title: '频点参数', key: 'rf-freq' },
        ],
      },
      {
        title: '无线资源管理',
        key: 'rrm',
        children: [
          { title: '切换参数', key: 'rrm-ho' },
          { title: '调度参数', key: 'rrm-sched' },
        ],
      },
      {
        title: '小区参数',
        key: 'cell',
        children: [
          { title: '基础配置', key: 'cell-basic' },
          { title: '邻区配置', key: 'cell-neighbor' },
        ],
      },
      {
        title: '传输参数',
        key: 'transport',
      },
    ],
  },
];

const mockData: ParamClassRow[] = [
  { id: '1', paramCode: 'TX_POWER', paramName: '发射功率', paramType: 'number', currentValue: 46, defaultValue: 43, unit: 'dBm', description: '基站发射功率' },
  { id: '2', paramCode: 'RS_POWER', paramName: '参考信号功率', paramType: 'number', currentValue: -3, defaultValue: 0, unit: 'dB', description: '参考信号功率偏置' },
  { id: '3', paramCode: 'PA_VALUE', paramName: 'PA值', paramType: 'enum', currentValue: 0, defaultValue: 0, unit: '', description: 'PDSCH功率调整值' },
  { id: '4', paramCode: 'PB_VALUE', paramName: 'PB值', paramType: 'enum', currentValue: 1, defaultValue: 1, unit: '', description: 'PDSCH功率调整值' },
];

export default function BatchParamClass() {
  const t = useT();
  const [selectedClass, setSelectedClass] = useState<string>('');
  const [searchValue, setSearchValue] = useState('');
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [batchEditVisible, setBatchEditVisible] = useState(false);
  const [batchForm] = Form.useForm();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useConfigParams({
    category: selectedClass || undefined,
    page,
    pageSize,
  });

  const tableSource = (data?.items ?? mockData) as unknown as ParamClassRow[];

  const columns: DataTableColumn<ParamClassRow>[] = useMemo(() => [
    { key: 'paramCode', title: t('config.paramCode'), dataIndex: 'paramCode', width: 160, mono: true, copyable: true },
    { key: 'paramName', title: t('config.paramName'), dataIndex: 'paramName', width: 160 },
    { key: 'paramType', title: t('config.paramType'), dataIndex: 'paramType', width: 100 },
    { key: 'currentValue', title: t('perf.value'), dataIndex: 'currentValue', width: 120 },
    { key: 'defaultValue', title: t('config.defaultValue'), dataIndex: 'defaultValue', width: 120 },
    { key: 'unit', title: t('perf.unit'), dataIndex: 'unit', width: 80 },
    { key: 'description', title: t('table.description'), dataIndex: 'description', width: 200, ellipsis: true },
  ], [t]);

  const handleBatchEdit = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    setBatchEditVisible(true);
  };

  const handleBatchSave = () => {
    batchForm.validateFields().then((vals: Record<string, unknown>) => {
      void message.success(t('common.save'));
      console.log('batch edit values:', vals);
      setBatchEditVisible(false);
      batchForm.resetFields();
    }).catch(() => undefined);
  };

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
          treeData={CLASS_TREE}
          onSelect={(keys) => {
            const key = keys[0] as string;
            setSelectedClass(key && key !== 'all' ? key : '');
          }}
          defaultExpandAll
          showLine
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography.Text strong style={{ fontSize: 14 }}>
          {selectedClass ? `${t('nav.config.batchClass')} — ${selectedClass}` : t('nav.config.batchClass')}
        </Typography.Text>
        <Space>
          <Button
            type="primary"
            icon={<EditOutlined />}
            onClick={handleBatchEdit}
            disabled={selectedKeys.length === 0}
          >
            {t('common.edit')} {selectedKeys.length > 0 ? `(${selectedKeys.length})` : ''}
          </Button>
        </Space>
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <DataTable<ParamClassRow>
          tableId="batch-param-class"
          columns={columns}
          dataSource={tableSource}
          loading={isLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          total={data?.total ?? tableSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          scroll={{ x: 1000 }}
        />
      </div>

      <Modal
        title={`${t('common.edit')} (${selectedKeys.length})`}
        open={batchEditVisible}
        onOk={handleBatchSave}
        onCancel={() => { setBatchEditVisible(false); batchForm.resetFields(); }}
        okText={t('common.confirm')}
      >
        <Form form={batchForm} layout="vertical">
          <Form.Item label={t('perf.value')} name="targetValue" rules={[{ required: true, message: t('common.placeholder') }]}>
            <InputNumber style={{ width: '100%' }} placeholder={t('common.placeholder')} />
          </Form.Item>
          <Form.Item label={t('table.description')} name="remark">
            <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </TreeListPageLayout>
  );
}
