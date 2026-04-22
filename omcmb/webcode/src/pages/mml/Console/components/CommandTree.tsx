import { useState, useCallback, useMemo } from 'react';
import { Input, Tree, Typography, Empty, Spin, Button } from 'antd';
import { SearchOutlined, FolderOutlined, CodeOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons';
import type { TreeProps } from 'antd';
import type { MMLCommand, MMLParamRef } from '@core/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import type { CommandTreeNode } from '../hooks/useCommandSelection';

interface CommandTreeProps {
  commands: MMLCommand[];
  selectedCommand: MMLCommand | null;
  treeData: CommandTreeNode[];
  searchText: string;
  isLoading?: boolean;
  onSearchChange: (text: string) => void;
  onSelectCommand: (command: MMLCommand | null, param?: MMLParamRef | null) => void | Promise<void>;
  onAddPublicTemplate?: () => void;
  onAddPrivateTemplate?: () => void;
}

export default function CommandTree({
  commands,
  selectedCommand,
  treeData,
  searchText,
  isLoading = false,
  onSearchChange,
  onSelectCommand,
  onAddPublicTemplate,
  onAddPrivateTemplate,
}: CommandTreeProps) {
  const t = useT();
  const token = useThemeToken();

  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  const renderAddButton = (onClick?: () => void) => {
    if (!onClick) return null;
    return (
      <Button
        type="text"
        size="small"
        icon={<PlusOutlined />}
        style={{ marginLeft: 4, padding: '0 4px', fontSize: 11 }}
        onClick={(e) => {
          e.stopPropagation();
          onClick();
        }}
      />
    );
  };

  const renderCommandLeaf = (node: CommandTreeNode) => ({
    key: node.key,
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
          <CodeOutlined style={{ marginRight: 6, color: node.template ? '#722ed1' : '#52c41a' }} />
          <span style={{ fontWeight: 500, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {node.label}
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
          {node.commandCode}
        </span>
      </div>
    ),
  });

  const renderedTreeData = useMemo<TreeProps['treeData']>(() => {
    return treeData.map((topNode) => {
      // Custom template root node
      if (topNode.nodeType === 'custom_root') {
        return {
          key: topNode.key,
          selectable: false,
          title: (
            <span style={{ fontWeight: 500 }}>
              <FolderOutlined style={{ marginRight: 6, color: token.colorPrimary }} />
              {topNode.label}
            </span>
          ),
          children: (topNode.children ?? []).map((subNode) => {
            const isPublic = subNode.nodeType === 'custom_public';
            const isPrivate = subNode.nodeType === 'custom_private';

            return {
              key: subNode.key,
              selectable: false,
              title: (
                <span style={{ fontWeight: 500, display: 'inline-flex', alignItems: 'center' }}>
                  <FolderOutlined style={{ marginRight: 6, color: isPublic ? '#52c41a' : '#fa8c16' }} />
                  {subNode.label}
                  <span style={{ marginLeft: 8, fontSize: 12, color: '#8c8c8c', fontWeight: 'normal' }}>
                    ({subNode.count ?? subNode.children?.length ?? 0})
                  </span>
                  {renderAddButton(isPublic ? onAddPublicTemplate : isPrivate ? onAddPrivateTemplate : undefined)}
                </span>
              ),
              children: (subNode.children ?? []).map((leafNode) => {
                // User directory node
                if (leafNode.nodeType === 'user_dir') {
                  return {
                    key: leafNode.key,
                    selectable: false,
                    title: (
                      <span style={{ fontWeight: 400 }}>
                        <UserOutlined style={{ marginRight: 6, color: '#8c8c8c' }} />
                        {leafNode.label}
                        <span style={{ marginLeft: 8, fontSize: 12, color: '#8c8c8c' }}>
                          ({leafNode.count ?? leafNode.children?.length ?? 0})
                        </span>
                      </span>
                    ),
                    children: (leafNode.children ?? []).map(renderCommandLeaf),
                  };
                }
                // Template leaf node
                return renderCommandLeaf(leafNode);
              }),
            };
          }),
        };
      }

      // Built-in category nodes
      return {
        key: topNode.key,
        selectable: false,
        title: (
          <span style={{ fontWeight: 500 }}>
            <FolderOutlined style={{ marginRight: 6, color: token.colorPrimary }} />
            {topNode.label}
            <span style={{ marginLeft: 8, fontSize: 12, color: '#8c8c8c', fontWeight: 'normal' }}>
              ({topNode.count ?? topNode.children?.length ?? 0})
            </span>
          </span>
        ),
        children: (topNode.children ?? []).map(renderCommandLeaf),
      };
    });
  }, [token.colorPrimary, treeData, onAddPublicTemplate, onAddPrivateTemplate]);

  const commandMap = useMemo(() => {
    function flatten(nodes: CommandTreeNode[]): CommandTreeNode[] {
      return nodes.flatMap((n) => [n, ...flatten(n.children ?? [])]);
    }
    const allNodes = flatten(treeData);

    // Map for all selectable nodes: key → { command }
    type CommandEntry = { command: MMLCommand };
    const entries: Array<[string, CommandEntry]> = [];

    for (const node of allNodes) {
      if (node.nodeType === 'command' && node.command) {
        entries.push([String(node.key), { command: node.command }]);
      }
    }

    // Build a lookup from commandCode to command definition for metadata inheritance
    const commandByCode = new Map<string, MMLCommand>();
    for (const cmd of commands) {
      commandByCode.set(cmd.commandCode, cmd);
    }

    // Also include template nodes — convert template to a synthetic MMLCommand
    const templateEntries = allNodes
      .filter((node): node is CommandTreeNode & { template: NonNullable<CommandTreeNode['template']> } => Boolean(node.template))
      .map((node) => {
        const tmpl = node.template;
        // Look up the matching command definition to inherit parameter metadata
        const matchedCmd = commandByCode.get(tmpl.commandCode);

        const params = Object.entries(tmpl.parameters).map(([name, defaultValue]) => {
          // Inherit metadata from the matched command's param definition
          const matchedParam = matchedCmd?.params?.find((p) => p.name === name);
          if (matchedParam) {
            return {
              ...matchedParam,
              defaultValue: defaultValue ?? matchedParam.defaultValue,
            };
          }
          return {
            name,
            type: 'string' as const,
            required: false,
            description: '',
            defaultValue,
          };
        });

        const syntheticCommand: MMLCommand = {
          id: `tmpl-${tmpl.id}`,
          commandName: tmpl.commandName,
          commandCode: tmpl.commandCode,
          category: tmpl.categoryGroup || t('mml.console.customTemplates'),
          description: tmpl.description,
          params,
          productTypes: tmpl.productTypes,
          operationType: tmpl.operationType,
          paramPaths: matchedCmd?.paramPaths,
          supportedOperations: matchedCmd?.supportedOperations,
          helpDoc: matchedCmd?.helpDoc,
          notes: matchedCmd?.notes,
        };
        return [String(node.key), { command: syntheticCommand } as CommandEntry] as const;
      });

    return new Map<string, CommandEntry>([...entries, ...templateEntries]);
  }, [commands, treeData]);

  const handleSelect: TreeProps['onSelect'] = useCallback(
    (selectedKeys) => {
      if (selectedKeys.length === 0) {
        return;
      }

      const entry = commandMap.get(String(selectedKeys[0]));
      if (entry) {
        void onSelectCommand(entry.command);
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
          style={{ borderRadius: 4 }}
          placeholder={t('common.search')}
          prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
          value={searchText}
          onChange={(e) => onSearchChange(e.target.value)}
          allowClear
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
            description={t('mml.console.noMatchingCommands')}
            style={{ marginTop: 40 }}
          />
        )}
      </div>
    </div>
  );
}
