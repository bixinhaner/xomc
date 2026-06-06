import { useMemo, useState } from 'react';
import { Button, Empty, Input, Modal, Space, Spin, Tag, Tooltip, Tree, Typography } from 'antd';
import { RightOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import { useGroupTree, useCommandSubFields } from '@core/hooks/api/useMmlConsole';
import { useI18nText } from '@/hooks/useI18nText';
import type { CommandItem } from '../types';
import { opColor } from '../constants';
import { flattenGroupTree, mapCommandItem, subFieldsToParamPaths } from '../adapters';

const { Text } = Typography;

interface CommandSelectModalProps {
  open: boolean;
  value: CommandItem | null;
  onCancel: () => void;
  onConfirm: (command: CommandItem) => void;
  /** 「指定参数」快捷入口：跳过命令选择，直接进入「配置参数」弹框的「指定参数」标签（裸路径专家模式）。 */
  onGotoRawParams: () => void;
}

/**
 * 命令选择弹框（设计 §3.3.2 + §3.12）—— 左侧两层命令树（分组 → 命令）+ 搜索，
 * 右侧选中命令预览（操作类型 / 命令码 / 参数 PATH）。
 * 数据来源：`useGroupTree` 拉层级命令树拍平为二级；选中命令后 `useCommandSubFields`
 * 拉该命令的参数路径（决定结果表格列 + 可写项）。
 */
export default function CommandSelectModal({
  open,
  value,
  onCancel,
  onConfirm,
  onGotoRawParams,
}: CommandSelectModalProps) {
  const { locale } = useI18nText();
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

  const { data: treeNodes, isLoading: treeLoading } = useGroupTree(undefined, locale);

  // 拍平为「分组 → 命令」二级，并建 id → 条目索引（取 groupName / GroupTreeCommand）。
  const entries = useMemo(() => flattenGroupTree(treeNodes ?? []), [treeNodes]);
  const entryById = useMemo(() => {
    const m = new Map<string, (typeof entries)[number]>();
    entries.forEach((e) => m.set(e.command.id, e));
    return m;
  }, [entries]);

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return entries;
    return entries.filter(
      (e) =>
        e.command.commandCode.toLowerCase().includes(kw) ||
        e.command.displayName.toLowerCase().includes(kw) ||
        e.groupName.toLowerCase().includes(kw),
    );
  }, [entries, keyword]);

  const treeData: DataNode[] = useMemo(() => {
    const byGroup = new Map<string, typeof entries>();
    filtered.forEach((e) => {
      const arr = byGroup.get(e.groupName) ?? [];
      arr.push(e);
      byGroup.set(e.groupName, arr);
    });
    return Array.from(byGroup.entries()).map(([group, items]) => ({
      key: `group:${group}`,
      title: group,
      selectable: false,
      children: items.map((e) => ({
        key: e.command.id,
        title: (
          <Space size={6}>
            <Tag color={opColor(e.command.operationType)} style={{ marginInlineEnd: 0 }}>
              {e.command.operationType}
            </Tag>
            <span>{e.command.displayName}</span>
          </Space>
        ),
      })),
    }));
  }, [filtered]);

  const expandedKeys = useMemo(() => treeData.map((n) => n.key as string), [treeData]);

  const selectedEntry = selectedId ? entryById.get(selectedId) : undefined;

  // 选中命令的参数路径（决定结果列 / 可写项）。命令变更自动重取。
  const { data: subFields, isFetching: subFieldsLoading } = useCommandSubFields(selectedId, locale);
  const paramPaths = useMemo(() => subFieldsToParamPaths(subFields ?? []), [subFields]);

  const handleOk = (): void => {
    if (!selectedEntry || !subFields) return;
    onConfirm(mapCommandItem(selectedEntry.groupName, selectedEntry.command, subFields));
  };

  return (
    <Modal
      title="选择命令"
      open={open}
      width={860}
      onCancel={onCancel}
      onOk={handleOk}
      okText="确定选择"
      cancelText="取消"
      okButtonProps={{ disabled: !selectedEntry || subFieldsLoading || !subFields }}
      destroyOnHidden
    >
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12 }}>
        <Input.Search
          allowClear
          placeholder="搜索：命令码 / 名称 / 分组"
          style={{ flex: 1, maxWidth: 420 }}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
        <Tooltip title="跳过命令选择，直接执行「指定参数」">
          <Button type="link" onClick={onGotoRawParams}>
            指定参数
            <RightOutlined style={{ fontSize: 11 }} />
          </Button>
        </Tooltip>
      </div>
      <div style={{ display: 'flex', gap: 12, height: 380 }}>
        <div style={{ width: '42%', overflow: 'auto', borderRight: '1px solid #f0f0f0', paddingRight: 8 }}>
          {treeLoading ? (
            <div style={{ textAlign: 'center', marginTop: 120 }}>
              <Spin />
            </div>
          ) : treeData.length === 0 ? (
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
          {selectedEntry ? (
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              {/* 命令详情头：操作类型缩写(Tag) + 命令名称 */}
              <Space size={8} wrap>
                <Tag color={opColor(selectedEntry.command.operationType)} style={{ marginInlineEnd: 0 }}>
                  {selectedEntry.command.operationType}
                </Tag>
                <Text strong style={{ fontSize: 15 }}>
                  {selectedEntry.command.displayName}
                </Text>
              </Space>
              {subFieldsLoading ? (
                <Spin size="small" />
              ) : (
                <>
                  <Text strong>参数 PATH（{paramPaths.length} 项）：</Text>
                  {paramPaths.length === 0 ? (
                    <Text type="secondary">该命令无可展示的参数路径</Text>
                  ) : (
                    <Space direction="vertical" size={4} style={{ width: '100%' }}>
                      {paramPaths.map((p) => (
                        <div
                          key={p.path}
                          style={{ display: 'flex', alignItems: 'baseline', gap: 8, fontSize: 12, minWidth: 0 }}
                        >
                          {/* PATH 名称固定宽展示，过长截断 + hover Tooltip 显示完整（§3.10.2） */}
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
                  )}
                </>
              )}
            </Space>
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请选择左侧命令查看详情" />
          )}
        </div>
      </div>
    </Modal>
  );
}
