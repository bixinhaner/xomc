import { useMemo, useState } from 'react';
import { Empty, Input, Modal, Space, Tag, Tree, Typography } from 'antd';
import type { DataNode } from 'antd/es/tree';
import type { CommandItem } from '../types';
import { MOCK_COMMANDS } from '../mock';
import { opColor, opLabel } from '../constants';

const { Text, Paragraph } = Typography;

interface CommandSelectModalProps {
  open: boolean;
  value: CommandItem | null;
  onCancel: () => void;
  onConfirm: (command: CommandItem) => void;
}

/**
 * 命令选择弹框（设计 §3.3.2）—— 左侧两层命令树（分组 → 命令）+ 搜索，右侧选中命令预览。
 * 当前为 mock：命令来自 MOCK_COMMANDS。
 */
export default function CommandSelectModal({
  open,
  value,
  onCancel,
  onConfirm,
}: CommandSelectModalProps) {
  const [keyword, setKeyword] = useState('');
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [wasOpen, setWasOpen] = useState(false);

  // 打开时回填（渲染阶段调整 state，避开 set-state-in-effect）。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setKeyword('');
      setSelectedId(value?.id);
    }
  }

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return MOCK_COMMANDS;
    return MOCK_COMMANDS.filter(
      (c) =>
        c.commandCode.toLowerCase().includes(kw) ||
        c.commandName.toLowerCase().includes(kw) ||
        c.description.toLowerCase().includes(kw) ||
        c.paramPaths.some((p) => p.path.toLowerCase().includes(kw)),
    );
  }, [keyword]);

  const treeData: DataNode[] = useMemo(() => {
    const byGroup = new Map<string, CommandItem[]>();
    filtered.forEach((c) => {
      const arr = byGroup.get(c.groupName) ?? [];
      arr.push(c);
      byGroup.set(c.groupName, arr);
    });
    return Array.from(byGroup.entries()).map(([group, cmds]) => ({
      key: `group:${group}`,
      title: group,
      selectable: false,
      children: cmds.map((c) => ({
        key: c.id,
        title: (
          <Space size={6}>
            <Tag color={opColor(c.operationType)} style={{ marginInlineEnd: 0 }}>
              {c.operationType}
            </Tag>
            <span>{c.commandName}</span>
          </Space>
        ),
      })),
    }));
  }, [filtered]);

  const expandedKeys = useMemo(
    () => treeData.map((n) => n.key as string),
    [treeData],
  );

  const selectedCommand = useMemo(
    () => MOCK_COMMANDS.find((c) => c.id === selectedId) ?? null,
    [selectedId],
  );

  return (
    <Modal
      title="选择命令"
      open={open}
      width={860}
      onCancel={onCancel}
      onOk={() => selectedCommand && onConfirm(selectedCommand)}
      okText="确定选择"
      cancelText="取消"
      okButtonProps={{ disabled: !selectedCommand }}
      destroyOnHidden
    >
      <Input.Search
        allowClear
        placeholder="搜索：命令码 / 名称 / 路径 / 描述"
        style={{ marginBottom: 12 }}
        value={keyword}
        onChange={(e) => setKeyword(e.target.value)}
      />
      <div style={{ display: 'flex', gap: 12, height: 380 }}>
        <div style={{ width: '42%', overflow: 'auto', borderRight: '1px solid #f0f0f0', paddingRight: 8 }}>
          {treeData.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="无匹配命令" />
          ) : (
            <Tree
              blockNode
              treeData={treeData}
              expandedKeys={expandedKeys}
              selectedKeys={selectedId ? [selectedId] : []}
              onSelect={(keys) => {
                const k = keys[0] as string | undefined;
                if (k && !k.startsWith('group:')) setSelectedId(k);
              }}
            />
          )}
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          {selectedCommand ? (
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              <Space size={8} wrap>
                <Tag color={opColor(selectedCommand.operationType)} style={{ marginInlineEnd: 0 }}>
                  {selectedCommand.operationType} {opLabel(selectedCommand.operationType)}
                </Tag>
                <Text strong style={{ fontSize: 15 }}>{selectedCommand.commandName}</Text>
              </Space>
              <Text type="secondary" code>{selectedCommand.commandCode}</Text>
              <Paragraph style={{ marginBottom: 4 }}>{selectedCommand.description}</Paragraph>
              <Text strong>参数 PATH（{selectedCommand.paramPaths.length} 项）：</Text>
              <Space direction="vertical" size={4} style={{ width: '100%' }}>
                {selectedCommand.paramPaths.map((p) => (
                  <div
                    key={p.path}
                    style={{ display: 'flex', alignItems: 'baseline', gap: 8, fontSize: 12, minWidth: 0 }}
                  >
                    {/* PATH 名称固定宽展示,PATH 过长截断 + hover Tooltip 显示完整(§3.10.2) */}
                    <Text style={{ flex: '0 0 96px' }} ellipsis={{ tooltip: p.label }}>
                      {p.label}
                    </Text>
                    <Text
                      code
                      style={{ flex: 1, minWidth: 0, fontSize: 12 }}
                      ellipsis={{ tooltip: p.path }}
                    >
                      {p.path}
                    </Text>
                  </div>
                ))}
              </Space>
            </Space>
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请选择左侧命令查看详情" />
          )}
        </div>
      </div>
    </Modal>
  );
}
