import { useMemo, useState } from 'react';
import { Input, Table, Tag, Spin, Empty, Space, Modal, Tooltip, message } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useGroupTree } from '@core/hooks/api/useMmlConsole';
import { useDeleteCommand } from '@core/hooks/api/useMmlAdmin';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import CommandDetailDrawer from './CommandDetailDrawer';

interface CommandRow extends GroupTreeCommand {
  groupPath: string;
}

function flattenCommands(nodes: GroupTreeNode[]): CommandRow[] {
  const out: CommandRow[] = [];
  const walk = (arr: GroupTreeNode[]) => {
    arr.forEach((g) => {
      (g.commands ?? []).forEach((c) => out.push({ ...c, groupPath: g.path }));
      if (g.children?.length) walk(g.children);
    });
  };
  walk(nodes);
  return out;
}

export default function CommandsTab() {
  const t = useT();
  const { data: tree = [], isLoading, refetch } = useGroupTree(undefined, 'zh-CN');
  const deleteMut = useDeleteCommand();

  const [search, setSearch] = useState('');
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [activeCommand, setActiveCommand] = useState<GroupTreeCommand | undefined>();

  const rows = useMemo(() => {
    const all = flattenCommands(tree);
    if (!search.trim()) return all;
    const lower = search.toLowerCase();
    return all.filter(
      (r) =>
        r.commandCode.toLowerCase().includes(lower) ||
        r.logicalCode.toLowerCase().includes(lower) ||
        r.displayName.toLowerCase().includes(lower),
    );
  }, [tree, search]);

  const handleDelete = (cmd: CommandRow) => {
    if (cmd.catalogProtected) {
      message.warning(t('mml.admin.catalog.common.lockedTooltip'));
      return;
    }
    Modal.confirm({
      title: `${t('mml.admin.catalog.common.delete')} ${cmd.commandCode}?`,
      okButtonProps: { danger: true },
      okText: t('mml.admin.catalog.common.confirm'),
      cancelText: t('mml.admin.catalog.common.cancel'),
      onOk: async () => {
        try {
          await deleteMut.mutateAsync(cmd.id);
          message.success(t('mml.admin.catalog.common.deleteSuccess'));
          void refetch();
        } catch (e) {
          if (e instanceof Error) message.error(e.message);
        }
      },
    });
  };

  const columns: ColumnsType<CommandRow> = [
    {
      title: t('mml.admin.catalog.commands.code'),
      dataIndex: 'commandCode',
      key: 'commandCode',
      width: 220,
    },
    {
      title: t('mml.admin.catalog.commands.logicalCode'),
      dataIndex: 'logicalCode',
      key: 'logicalCode',
      width: 180,
    },
    {
      title: t('mml.admin.catalog.commands.op'),
      dataIndex: 'operationType',
      key: 'operationType',
      width: 80,
      render: (op: string) => <Tag>{op}</Tag>,
    },
    {
      title: t('mml.admin.catalog.commands.displayName'),
      dataIndex: 'displayName',
      key: 'displayName',
      ellipsis: true,
      render: (name: string, cmd) => (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          {name}
          {cmd.catalogProtected && (
            <Tooltip title={t('mml.admin.catalog.common.lockedTooltip')}>
              <LockOutlined style={{ color: '#999' }} />
            </Tooltip>
          )}
        </span>
      ),
    },
    {
      title: t('mml.admin.catalog.commands.groupPath'),
      dataIndex: 'groupPath',
      key: 'groupPath',
      ellipsis: true,
      width: 240,
    },
    {
      title: 'Op',
      key: 'op',
      width: 140,
      render: (_, cmd) => (
        <Space>
          <a
            onClick={() => {
              setActiveCommand(cmd);
              setDrawerOpen(true);
            }}
          >
            {t('mml.admin.catalog.common.edit')}
          </a>
          <a
            onClick={() => handleDelete(cmd)}
            style={{ color: cmd.catalogProtected ? '#bfbfbf' : undefined, cursor: cmd.catalogProtected ? 'not-allowed' : 'pointer' }}
          >
            {t('mml.admin.catalog.common.delete')}
          </a>
        </Space>
      ),
    },
  ];

  if (isLoading) return <Spin />;
  if (tree.length === 0) return <Empty />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Input.Search
        placeholder={t('mml.admin.catalog.params.search')}
        allowClear
        onSearch={setSearch}
        onChange={(e) => !e.target.value && setSearch('')}
      />
      <Table<CommandRow>
        rowKey="id"
        columns={columns}
        dataSource={rows}
        size="small"
        pagination={{ pageSize: 20, showSizeChanger: false }}
      />
      <CommandDetailDrawer
        open={drawerOpen}
        command={activeCommand}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
}
