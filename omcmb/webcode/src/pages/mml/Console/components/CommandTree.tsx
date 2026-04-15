import { useState, useCallback, useMemo, useEffect } from 'react';
import { Input, Select, Tree, Typography, Empty, Spin } from 'antd';
import { SearchOutlined, FolderOutlined, CodeOutlined } from '@ant-design/icons';
import type { TreeProps } from 'antd';
import type { MMLCommand } from '@/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import type { CommandTreeNode } from '../hooks/useCommandSelection';

interface CommandTreeProps {
  selectedCommand: MMLCommand | null;
  treeData: CommandTreeNode[];
  categoryOptions: { label: string; value: string }[];
  searchText: string;
  categoryFilter: string;
  isLoading?: boolean;
  onSearchChange: (text: string) => void;
  onFilterChange: (category: string) => void;
  onSelectCommand: (command: MMLCommand | null) => void | Promise<void>;
}

export default function CommandTree({
  selectedCommand,
  treeData,
  categoryOptions,
  searchText,
  categoryFilter,
  isLoading = false,
  onSearchChange,
  onFilterChange,
  onSelectCommand,
}: CommandTreeProps) {
  const t = useT();
  const token = useThemeToken();

  const categoryNodeKeys = useMemo(
    () => treeData.map((node) => String(node.key)),
    [treeData]
  );

  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(categoryNodeKeys);

  useEffect(() => {
    setExpandedKeys(categoryNodeKeys);
  }, [categoryNodeKeys]);

  const renderedTreeData = useMemo<TreeProps['treeData']>(() => {
    return treeData.map((categoryNode) => ({
      key: categoryNode.key,
      selectable: false,
      title: (
        <span style={{ fontWeight: 500 }}>
          <FolderOutlined style={{ marginRight: 6, color: token.colorPrimary }} />
          {categoryNode.label}
          <span style={{ marginLeft: 8, fontSize: 12, color: '#8c8c8c', fontWeight: 'normal' }}>
            ({categoryNode.count ?? categoryNode.children?.length ?? 0})
          </span>
        </span>
      ),
      children: categoryNode.children?.map((commandNode) => ({
        key: commandNode.key,
        isLeaf: true,
        title: (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              width: '100%',
              paddingRight: 8,
              gap: 8,
            }}
          >
            <span style={{ display: 'flex', alignItems: 'center', minWidth: 0 }}>
              <CodeOutlined style={{ marginRight: 6, color: '#52c41a' }} />
              <span style={{ fontWeight: 500, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {commandNode.label}
              </span>
            </span>
            <span
              style={{
                fontSize: 11,
                color: '#8c8c8c',
                fontFamily: 'monospace',
                flexShrink: 0,
              }}
            >
              {commandNode.commandCode}
            </span>
          </div>
        ),
      })),
    }));
  }, [token.colorPrimary, treeData]);

  const commandMap = useMemo(() => {
    const entries = treeData.flatMap((categoryNode) =>
      (categoryNode.children ?? [])
        .filter((node): node is CommandTreeNode & { command: MMLCommand } => Boolean(node.command))
        .map((node) => [String(node.key), node.command] as const)
    );

    return new Map<string, MMLCommand>(entries);
  }, [treeData]);

  const handleSelect: TreeProps['onSelect'] = useCallback(
    (selectedKeys) => {
      if (selectedKeys.length === 0) {
        return;
      }

      const command = commandMap.get(String(selectedKeys[0]));
      if (command) {
        void onSelectCommand(command);
      }
    },
    [commandMap, onSelectCommand]
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
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

      <div style={{ padding: '10px 12px', background: token.colorBgContainer }}>
        <Input.Search
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
          options={categoryOptions}
          value={categoryFilter || undefined}
          onChange={(val) => onFilterChange(val ?? '')}
        />
      </div>

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
        {isLoading ? (
          <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 48 }}>
            <Spin size="small" />
          </div>
        ) : renderedTreeData && renderedTreeData.length > 0 ? (
          <Tree
            showLine={{ showLeafIcon: false }}
            expandedKeys={expandedKeys}
            onExpand={setExpandedKeys}
            selectedKeys={selectedCommand ? [selectedCommand.id] : []}
            onSelect={handleSelect}
            treeData={renderedTreeData}
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
