import { useState, useCallback, useMemo } from 'react';
import { Input, Select, Tree, Typography, Empty } from 'antd';
import { SearchOutlined, FolderOutlined, CodeOutlined } from '@ant-design/icons';
import type { TreeDataNode, TreeProps } from 'antd';
import type { MMLCommand } from '@/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

interface CommandTreeProps {
  selectedCommand: MMLCommand | null;
  commands: MMLCommand[];
  categories: string[];
  commandsByCategory: Map<string, MMLCommand[]>;
  searchText: string;
  categoryFilter: string;
  onSearchChange: (text: string) => void;
  onFilterChange: (category: string) => void;
  onSelectCommand: (command: MMLCommand | null) => void;
}

export default function CommandTree({
  selectedCommand,
  commands,
  categories,
  commandsByCategory,
  searchText,
  categoryFilter,
  onSearchChange,
  onFilterChange,
  onSelectCommand,
}: CommandTreeProps) {
  const t = useT();
  const token = useThemeToken();

  // 展开的树节点
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(() => {
    return categories.map((cat) => `category-${cat}`);
  });

  // 生成树形数据
  const treeData = useMemo((): TreeDataNode[] => {
    const nodes: TreeDataNode[] = [];
    commandsByCategory.forEach((cmds, category) => {
      nodes.push({
        key: `category-${category}`,
        title: (
          <span style={{ fontWeight: 500 }}>
            <FolderOutlined style={{ marginRight: 6, color: token.colorPrimary }} />
            {category}
            <span style={{ marginLeft: 8, fontSize: 12, color: '#8c8c8c', fontWeight: 'normal' }}>
              ({cmds.length})
            </span>
          </span>
        ),
        selectable: false,
        children: cmds.map((cmd) => ({
          key: cmd.id,
          title: (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              width: '100%',
              paddingRight: 8,
            }}>
              <span>
                <CodeOutlined style={{ marginRight: 6, color: '#52c41a' }} />
                <span style={{ fontWeight: 500 }}>{cmd.commandName}</span>
              </span>
              <span style={{
                fontSize: 11,
                color: '#8c8c8c',
                fontFamily: 'monospace',
              }}>
                {cmd.commandCode}
              </span>
            </div>
          ),
          isLeaf: true,
        })),
      });
    });
    return nodes;
  }, [commandsByCategory, token.colorPrimary]);

  // 树节点选择
  const handleSelect: TreeProps['onSelect'] = useCallback(
    (selectedKeys: React.Key[]) => {
      if (selectedKeys.length > 0) {
        const key = String(selectedKeys[0]);
        const cmd = commands.find((c) => c.id === key);
        if (cmd) {
          onSelectCommand(cmd);
        }
      }
    },
    [commands, onSelectCommand]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* 头部 */}
      <div
        style={{
          padding: '10px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: token.colorBgLayout,
        }}
      >
        <Typography.Text strong style={{ fontSize: 13 }}>
          {t('nav.mml.commands')}
        </Typography.Text>
        {selectedCommand && (
          <span
            style={{
              fontSize: 10,
              padding: '2px 8px',
              background: token.colorPrimaryBg,
              border: `1px solid ${token.colorPrimaryBorder}`,
              borderRadius: 10,
              fontFamily: 'monospace',
              color: token.colorPrimary,
            }}
          >
            {selectedCommand.commandCode}
          </span>
        )}
      </div>

      {/* 搜索和筛选 */}
      <div style={{ padding: '10px 12px', background: token.colorBgContainer }}>
        <Input
          size="small"
          style={{ marginBottom: 8, borderRadius: 4 }}
          placeholder={t('common.search')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => onSearchChange(e.target.value)}
          allowClear
        />
        <Select
          size="small"
          style={{ width: '100%', borderRadius: 4 }}
          placeholder="命令类型"
          allowClear
          options={categories.map((c) => ({ label: c, value: c }))}
          value={categoryFilter || undefined}
          onChange={(val) => onFilterChange(val ?? '')}
        />
      </div>

      {/* 命令树 */}
      <div
        className="no-scrollbar"
        style={{
          flex: 1,
          minHeight: 80,
          overflow: 'auto',
          padding: '8px 4px',
          background: token.colorBgContainer,
        }}
      >
        {treeData.length > 0 ? (
          <Tree
            showLine={{ showLeafIcon: false }}
            expandedKeys={expandedKeys}
            onExpand={setExpandedKeys}
            selectedKeys={selectedCommand ? [selectedCommand.id] : []}
            onSelect={handleSelect}
            treeData={treeData}
            style={{ background: 'transparent', fontSize: 11 }}
          />
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无匹配的命令"
            style={{ marginTop: 40 }}
          />
        )}
      </div>
    </div>
  );
}
