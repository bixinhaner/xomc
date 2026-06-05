import { Empty } from 'antd';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import CommandMetaSection from './CommandMetaSection';
import PathListSection from './PathListSection';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §4：右栏详情面板。
// 三个空态：
//   1. 未选中（command=undefined && groupSummary=undefined）→ 引导文案
//   2. 选中分组（groupSummary 提供）→ 提示选择具体命令
//   3. 选中命令 → CommandMetaSection + PathListSection

export interface RightDetailPanelProps {
  command: GroupTreeCommand | undefined;
  parentGroupId: string | undefined;
  groupOptions: GroupTreeNode[];
  /** 选中的是分组时展示分组摘要（命令数等） */
  selectedGroup: GroupTreeNode | undefined;
  editing: boolean;
  onEditingChange: (editing: boolean) => void;
  onDirtyChange: (dirty: boolean) => void;
}

export default function RightDetailPanel({
  command,
  parentGroupId,
  groupOptions,
  selectedGroup,
  editing,
  onEditingChange,
  onDirtyChange,
}: RightDetailPanelProps): React.ReactElement {
  const t = useT();

  if (!command) {
    if (selectedGroup) {
      return (
        <Empty
          description={
            <>
              <div>
                {t('mml.admin.catalog.empty.groupSelected', {
                  name: selectedGroup.displayName,
                  count: selectedGroup.commands?.length ?? 0,
                })}
              </div>
              <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                {t('mml.admin.catalog.empty.groupHint')}
              </div>
            </>
          }
        />
      );
    }
    return (
      <Empty
        description={
          <>
            <div>{t('mml.admin.catalog.empty.selectCommand')}</div>
            <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
              {t('mml.admin.catalog.empty.selectCommandHint')}
            </div>
          </>
        }
      />
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <CommandMetaSection
        command={command}
        parentGroupId={parentGroupId ?? ''}
        groupOptions={groupOptions}
        editing={editing}
        onEditingChange={onEditingChange}
        onDirtyChange={onDirtyChange}
      />
      <PathListSection command={command} />
    </div>
  );
}
