import { useCallback, useMemo, useState } from 'react';
import { Empty, Input, Modal, Tree, Typography } from 'antd';
import { FolderOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import { useDeviceGroups } from '@core/hooks/api/useDevices';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface MoveToGroupModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (groupId: string) => void;
  confirmLoading?: boolean;
  selectedCount?: number;
}

type GroupItem = { id: string; name: string; parentId: string | null; deviceCount: number; description: string };

function buildTreeNodes(groups: GroupItem[], searchText: string): DataNode[] {
  const lower = searchText.toLowerCase();

  // 搜索时展示匹配节点及其祖先链
  const matchedIds = new Set<string>();
  if (lower) {
    for (const g of groups) {
      if (g.name.toLowerCase().includes(lower)) {
        // 标记自身及所有祖先
        let current: GroupItem | undefined = g;
        while (current) {
          matchedIds.add(current.id);
          current = current.parentId ? groups.find((p) => p.id === current!.parentId) : undefined;
        }
      }
    }
  }

  function buildChildren(parentId: string | null): DataNode[] {
    return groups
      .filter((g) => g.parentId === parentId)
      .filter((g) => !lower || matchedIds.has(g.id))
      .map((g) => {
        const children = buildChildren(g.id);
        const isDirectMatch = lower && g.name.toLowerCase().includes(lower);
        return {
          key: g.id,
          title: (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <FolderOutlined style={{ color: '#fa8c16', fontSize: 14 }} />
              <span style={{ fontWeight: isDirectMatch ? 500 : 400 }}>{g.name}</span>
              <Text type="secondary" style={{ fontSize: 11 }}>({g.deviceCount})</Text>
            </span>
          ),
          children: children.length > 0 ? children : undefined,
        };
      });
  }

  return buildChildren(null);
}

export default function MoveToGroupModal({
  open,
  onClose,
  onConfirm,
  confirmLoading,
  selectedCount,
}: MoveToGroupModalProps) {
  const t = useT();
  const { data: groupsData, isLoading } = useDeviceGroups();
  const groups: GroupItem[] = (groupsData && 'groups' in groupsData ? (groupsData as { groups: GroupItem[] }).groups : (groupsData as unknown as GroupItem[] | undefined)) ?? [];

  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [searchText, setSearchText] = useState('');
  const [showTip, setShowTip] = useState(false);

  const treeData = useMemo(() => buildTreeNodes(groups, searchText), [groups, searchText]);

  const selectedGroupName = useMemo(
    () => groups.find((g) => g.id === selectedGroupId)?.name ?? '',
    [groups, selectedGroupId]
  );

  const handleOk = useCallback(() => {
    if (!selectedGroupId) {
      setShowTip(true);
      return;
    }
    onConfirm(selectedGroupId);
  }, [selectedGroupId, onConfirm]);

  const handleCancel = useCallback(() => {
    setSelectedGroupId('');
    setSearchText('');
    setShowTip(false);
    onClose();
  }, [onClose]);

  return (
    <Modal
      title={t('common.moveToGroup')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={confirmLoading}
      width={480}
      destroyOnHidden
    >
      {selectedCount != null && selectedCount > 0 && (
        <div style={{ marginBottom: 12, color: 'rgba(0,0,0,0.45)', fontSize: 13 }}>
          {t('device.moveToGroupTip', { count: selectedCount })}
        </div>
      )}

      <Input
        prefix={<SearchOutlined style={{ color: 'rgba(0,0,0,0.25)' }} />}
        placeholder={t('device.searchGroup')}
        allowClear
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        style={{ marginBottom: 12 }}
      />

      <div
        style={{
          border: '1px solid #d9d9d9',
          borderRadius: 6,
          padding: '8px 4px',
          minHeight: 200,
          maxHeight: 360,
          overflow: 'auto',
          background: '#fafafa',
        }}
      >
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 40, color: 'rgba(0,0,0,0.25)' }}>
            {t('common.loading')}...
          </div>
        ) : treeData.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.noData')} />
        ) : (
          <Tree
            treeData={treeData}
            defaultExpandAll
            selectedKeys={selectedGroupId ? [selectedGroupId] : []}
            onSelect={(keys) => {
              const key = keys[0] as string | undefined;
              if (key) {
                setSelectedGroupId(key);
                setShowTip(false);
              }
            }}
            blockNode
            style={{ background: 'transparent' }}
          />
        )}
      </div>

      {selectedGroupId && (
        <div style={{ marginTop: 8, fontSize: 13, color: 'rgba(0,0,0,0.65)' }}>
          {t('device.selectedGroup')}: <Text strong>{selectedGroupName}</Text>
        </div>
      )}

      {showTip && (
        <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>
          {t('sync.selectGroupTip')}
        </div>
      )}
    </Modal>
  );
}
