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
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* 终端输出：用户决策 2026-05-18 + 二次确认 — 必须放在整个 RightPanel 顶端，
          Tabs 外。在 Tabs 内部时看起来"插在 tab 栏和 Control Panel 中间"；外置后
          它跨 Control / ParamPath 两个 tab 永远可见，更接近真终端的固定区域感。 */}
      <div style={{ height: 240, flexShrink: 0 }}>
        <TerminalPanel lines={[]} />
      </div>
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
                {/* 参数列表（LST/MOD/ADD/RMV 因 op 不同换皮）：选中长命令
                    如 LST DEVICE_INFO 可能有 200+ sub_fields，包一层
                    max-height + overflow:auto 防撑长页面。 */}
                <div
                  style={{
                    maxHeight: 'calc(100vh - 540px)',
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
    </div>
  );
}
