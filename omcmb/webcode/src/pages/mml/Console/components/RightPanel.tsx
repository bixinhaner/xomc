import { useState, useMemo } from 'react';
import { Tabs, Empty } from 'antd';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type { Statement } from '@core/types/mmlConsole';
import type { MMLTask } from '@core/types/mml';
import MmlEditor from './MmlEditor';
import SubFieldChecklist from './SubFieldChecklist';
import SubFieldInputList from './SubFieldInputList';
import InstancePicker from './InstancePicker';
import TerminalPanel from './TerminalPanel';
import ParamPathExpert from './ParamPathExpert';
import { useT } from '@/hooks/useT';

type RightTab = 'control' | 'paramPath';

export interface RightPanelProps {
  onExecuted?: (task: MMLTask) => void;
}

function ActiveSubView({ statement }: { statement: Statement }) {
  switch (statement.operationType) {
    case 'LST':
      return <SubFieldChecklist statement={statement} />;
    case 'MOD':
    case 'ADD':
      return <SubFieldInputList statement={statement} />;
    case 'RMV':
      return <InstancePicker statement={statement} />;
    default:
      return null;
  }
}

export default function RightPanel({ onExecuted }: RightPanelProps) {
  const t = useT();
  const [tab, setTab] = useState<RightTab>('control');
  const activeStatementUid = useMmlConsoleStore((s) => s.activeStatementUid);
  const statements = useMmlConsoleStore((s) => s.statements);

  const activeStatement = useMemo(
    () => statements.find((s) => s.uid === activeStatementUid) ?? null,
    [statements, activeStatementUid],
  );

  return (
    <Tabs
      activeKey={tab}
      onChange={(k) => setTab(k as RightTab)}
      items={[
        {
          key: 'control',
          label: t('mml.console.tab.control'),
          children: (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <MmlEditor onExecuted={onExecuted} />
              {/* 终端输出位置：MmlEditor 下方、参数列表上方（恢复 2026-05-14
                  之前的布局；da6c1c68 重构曾把它挪到末尾）。 */}
              <TerminalPanel lines={[]} />
              {/* 参数列表（LST/MOD/ADD/RMV 因 op 不同换皮）：选中长命令
                  如 LST DEVICE_INFO 可能有 200+ sub_fields，包一层
                  max-height + overflow:auto 防撑长页面。 */}
              <div
                style={{
                  maxHeight: 'calc(100vh - 360px)',
                  overflowY: 'auto',
                  paddingRight: 4,
                }}
              >
                {activeStatement ? (
                  <ActiveSubView statement={activeStatement} />
                ) : (
                  <Empty
                    description={t('mml.console.stepBar.step2')}
                    style={{ padding: 12 }}
                  />
                )}
              </div>
            </div>
          ),
        },
        {
          key: 'paramPath',
          label: t('mml.console.tab.paramPath'),
          children: <ParamPathExpert command={null} />,
        },
      ]}
    />
  );
}
