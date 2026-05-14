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
              {activeStatement ? (
                <ActiveSubView statement={activeStatement} />
              ) : (
                <Empty
                  description={t('mml.console.stepBar.step2')}
                  style={{ padding: 12 }}
                />
              )}
              <TerminalPanel lines={[]} />
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
