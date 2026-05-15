import { useMemo } from 'react';
import { Tree, Empty, Spin, Alert } from 'antd';
import { FolderOutlined, CodeOutlined } from '@ant-design/icons';
import type { TreeDataNode } from 'antd';
import { useGroupTree } from '@core/hooks/api/useMmlConsole';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

function buildAdminTreeData(nodes: GroupTreeNode[]): TreeDataNode[] {
  const sorted = [...nodes].sort((a, b) => a.displayOrder - b.displayOrder);
  return sorted.map((g) => ({
    key: `group:${g.id}`,
    title: (
      <span>
        <FolderOutlined style={{ marginRight: 4 }} />
        {g.displayName}
        <span style={{ marginLeft: 8, fontSize: 12, color: '#999' }}>{g.path}</span>
      </span>
    ),
    selectable: true,
    children: [
      ...buildAdminTreeData(g.children ?? []),
      ...[...(g.commands ?? [])]
        .sort((a, b) => a.displayName.localeCompare(b.displayName))
        .map<TreeDataNode>((c) => ({
          key: `cmd:${c.id}`,
          title: (
            <span>
              <CodeOutlined style={{ marginRight: 4 }} />
              {c.displayName}
            </span>
          ),
          isLeaf: true,
        })),
    ],
  }));
}

export default function GroupsTab() {
  const t = useT();
  const { data: tree = [], isLoading } = useGroupTree(undefined, 'zh-CN');

  const treeData = useMemo(() => buildAdminTreeData(tree), [tree]);

  if (isLoading) return <Spin />;
  if (tree.length === 0) return <Empty />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Alert
        type="info"
        showIcon
        message={t('mml.admin.catalog.groups.dragHint')}
        description={t('mml.admin.catalog.xmlImport.pending')}
      />
      <Tree treeData={treeData} showLine defaultExpandAll={false} />
    </div>
  );
}
