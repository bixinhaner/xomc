import { useState } from 'react';
import { Input, Table, Spin, Empty, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useParamsList } from '@core/hooks/api/useMmlAdmin';
import type { ParamAdmin } from '@core/types/mmlAdmin';
import { useT } from '@/hooks/useT';
import ParamReferencesDrawer from './ParamReferencesDrawer';

const PAGE_SIZE = 20;

export default function ParamsTab() {
  const t = useT();
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [refsDrawerOpen, setRefsDrawerOpen] = useState(false);
  const [activeParam, setActiveParam] = useState<ParamAdmin | undefined>();

  const { data, isLoading } = useParamsList({ search, page, pageSize: PAGE_SIZE });

  const columns: ColumnsType<ParamAdmin> = [
    { title: 'param_code', dataIndex: 'paramCode', key: 'paramCode', width: 200 },
    { title: 'TR-069 Path', dataIndex: 'tr069Path', key: 'tr069Path', ellipsis: true },
    { title: 'access_type', dataIndex: 'accessType', key: 'accessType', width: 120 },
    {
      title: 'metadata',
      key: 'meta',
      width: 240,
      render: (_, r) => (
        <span>
          {r.isObject && <Tag>object</Tag>}
          {r.supportsAdd && <Tag color="green">add</Tag>}
          {r.supportsDelete && <Tag color="red">delete</Tag>}
          {r.changeApplies === 'OnReboot' && <Tag color="warning">OnReboot</Tag>}
          {r.catalogProtected && <Tag color="default">protected</Tag>}
        </span>
      ),
    },
    {
      title: t('mml.admin.catalog.params.references'),
      key: 'refs',
      width: 120,
      render: (_, r) => (
        <a
          onClick={(e) => {
            e.stopPropagation();
            setActiveParam(r);
            setRefsDrawerOpen(true);
          }}
        >
          {t('mml.admin.catalog.params.references')}
        </a>
      ),
    },
  ];

  if (isLoading) return <Spin />;
  if (!data || data.items.length === 0) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Input.Search
          placeholder={t('mml.admin.catalog.params.search')}
          allowClear
          onSearch={(v) => {
            setSearch(v);
            setPage(1);
          }}
        />
        <Empty />
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Input.Search
        placeholder={t('mml.admin.catalog.params.search')}
        allowClear
        onSearch={(v) => {
          setSearch(v);
          setPage(1);
        }}
      />
      <Table<ParamAdmin>
        rowKey="id"
        columns={columns}
        dataSource={data.items}
        pagination={{
          current: data.page,
          pageSize: data.pageSize,
          total: data.total,
          onChange: setPage,
        }}
      />
      <ParamReferencesDrawer
        open={refsDrawerOpen}
        param={activeParam}
        onClose={() => setRefsDrawerOpen(false)}
      />
    </div>
  );
}
