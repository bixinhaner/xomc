import { useState, useMemo, useCallback } from 'react';
import { Tabs, Empty, Modal, message } from 'antd';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useExecuteStatementsStructured } from '@core/hooks/api/useMmlConsole';
import { statementToStructured } from '@core/types/mmlConsole';
import type { Statement } from '@core/types/mmlConsole';
import type { MMLTask } from '@core/types/mml';
import MmlEditor from './MmlEditor';
import SubFieldChecklist from './SubFieldChecklist';
import SubFieldInputList from './SubFieldInputList';
import InstancePicker from './InstancePicker';
import TerminalPanel from './TerminalPanel';
import ParamPathExpert from './ParamPathExpert';
import { ConsoleActionBar } from './ConsoleActionBar';
import {
  InstanceArityInput,
  deriveArityFromSubFields,
} from './InstanceArityInput';
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

function hasOnRebootHits(statements: Statement[]): boolean {
  return statements.some((stmt) =>
    stmt.subFields.some(
      (sf) =>
        sf.changeApplies === 'OnReboot' &&
        (stmt.selectedSubFieldIds.includes(sf.id) || stmt.values[sf.mmlCode] !== undefined),
    ),
  );
}

export default function RightPanel({ onExecuted }: RightPanelProps) {
  const t = useT();
  const [tab, setTab] = useState<RightTab>('control');
  const activeStatementUid = useMmlConsoleStore((s) => s.activeStatementUid);
  const statements = useMmlConsoleStore((s) => s.statements);
  const selectedDeviceSns = useMmlConsoleStore((s) => s.selectedDeviceSns);
  const updateStatement = useMmlConsoleStore((s) => s.updateStatement);
  const setInstanceSelectors = useMmlConsoleStore((s) => s.setInstanceSelectors);
  // R-9.2：ConsoleActionBar 走结构化通道；MmlEditor 仍用旧 useExecuteStatements。
  // 切换边界：activeStatement 由 CommandTree 加载并附带 subFields → 可做 structured 转换。
  const executeMutation = useExecuteStatementsStructured();

  const activeStatement = useMemo(
    () => statements.find((s) => s.uid === activeStatementUid) ?? null,
    [statements, activeStatementUid],
  );

  // R-9.1 ConsoleActionBar 摘要文案：按 op_type 显示已勾选/已填项数
  const summaryText = useMemo(() => {
    if (!activeStatement) return undefined;
    switch (activeStatement.operationType) {
      case 'LST':
        return t('mml.console.actionBar.summaryLst', {
          selected: activeStatement.selectedSubFieldIds.length,
          total: activeStatement.subFields.length,
        });
      case 'MOD':
      case 'ADD':
        return t('mml.console.actionBar.summaryMod', {
          filled: Object.keys(activeStatement.values).length,
        });
      case 'RMV':
        return t('mml.console.actionBar.summaryRmv', {
          index: activeStatement.rmvInstanceIndex ?? '-',
        });
      default:
        return undefined;
    }
  }, [activeStatement, t]);

  const runExecute = useCallback(async () => {
    try {
      // 把 UI 编辑态 Statement[] 翻译为 StructuredStatement[]，路径以 standardPath
      // 形式直传后端；后端按 device.product_class 翻译为 privatePath。
      const structured = statements.map(statementToStructured);
      const task = await executeMutation.mutateAsync({
        statements: structured,
        deviceSns: selectedDeviceSns,
        executeType: 'immediate',
      });
      message.success(t('mml.console.execute.success', { taskId: task.id }));
      onExecuted?.(task);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(t('mml.console.execute.failed', { message: msg }));
    }
  }, [executeMutation, onExecuted, selectedDeviceSns, statements, t]);

  const handleExecute = useCallback(() => {
    if (selectedDeviceSns.length === 0) {
      message.warning(t('mml.console.execute.noDevices'));
      return;
    }
    if (statements.length === 0) {
      message.warning(t('mml.console.execute.noStatements'));
      return;
    }
    if (hasOnRebootHits(statements)) {
      Modal.confirm({
        title: t('mml.console.editor.onRebootConfirm'),
        onOk: runExecute,
      });
      return;
    }
    void runExecute();
  }, [runExecute, selectedDeviceSns.length, statements, t]);

  // R-4：从当前 activeStatement 的 subFields 推断 instance arity；
  // > 0 时在 ActiveSubView 之前渲染 InstanceArityInput，4 个 op 都适用。
  const instanceArity = useMemo(
    () => (activeStatement ? deriveArityFromSubFields(activeStatement.subFields) : 0),
    [activeStatement],
  );

  const handleSelectorChange = useCallback(
    (key: string, value: string) => {
      if (!activeStatement) return;
      const next: Record<string, string> = {
        ...(activeStatement.instanceSelectors ?? {}),
        [key]: value,
      };
      setInstanceSelectors(activeStatement.uid, next);
    },
    [activeStatement, setInstanceSelectors],
  );

  // LST 全选 / 全不选只对当前 activeStatement 的 subFields 操作
  const handleSelectAll = useCallback(() => {
    if (!activeStatement || activeStatement.operationType !== 'LST') return;
    updateStatement(activeStatement.uid, {
      selectedSubFieldIds: activeStatement.subFields.map((sf) => sf.id),
    });
  }, [activeStatement, updateStatement]);

  const handleClearAll = useCallback(() => {
    if (!activeStatement || activeStatement.operationType !== 'LST') return;
    updateStatement(activeStatement.uid, { selectedSubFieldIds: [] });
  }, [activeStatement, updateStatement]);

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
            label: t('mml.tabs.controlPanel'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <MmlEditor onExecuted={onExecuted} />
                {/* 参数列表（LST/MOD/ADD/RMV 因 op 不同换皮）：选中长命令
                    如 LST DEVICE_INFO 可能有 200+ sub_fields，包一层
                    max-height + overflow:auto 防撑长页面。 */}
                <div
                  style={{
                    maxHeight: 'calc(100vh - 600px)',
                    overflowY: 'auto',
                    paddingRight: 4,
                  }}
                >
                  {activeStatement ? (
                    <>
                      {/* R-4：命令含多层 {i} 占位符时显示 instance 索引输入；
                          arity=0 时组件自身返回 null，不留空白。 */}
                      <InstanceArityInput
                        arity={instanceArity}
                        values={activeStatement.instanceSelectors ?? {}}
                        onChange={handleSelectorChange}
                        operationType={activeStatement.operationType}
                      />
                      <ActiveSubView statement={activeStatement} />
                    </>
                  ) : (
                    <Empty
                      description={t('mml.console.stepBar.step2')}
                      style={{ padding: 12 }}
                    />
                  )}
                </div>
                {/* R-9.1 底部固定操作行：执行按钮显式带 op_type，避免随
                    LST/MOD/ADD/RMV 切换位置漂移。仅当有 activeStatement
                    时挂载（无命令选中时不渲染，避免空按钮误导）。 */}
                {activeStatement && (
                  <ConsoleActionBar
                    operationType={activeStatement.operationType}
                    summaryText={summaryText}
                    deviceCount={selectedDeviceSns.length}
                    disabled={selectedDeviceSns.length === 0 || statements.length === 0}
                    loading={executeMutation.isPending}
                    onSelectAll={
                      activeStatement.operationType === 'LST' ? handleSelectAll : undefined
                    }
                    onClearAll={
                      activeStatement.operationType === 'LST' ? handleClearAll : undefined
                    }
                    onExecute={handleExecute}
                  />
                )}
              </div>
            ),
          },
          {
            key: 'paramPath',
            label: t('mml.tabs.parameterPathCommand'),
            children: <ParamPathExpert command={null} />,
          },
        ]}
      />
    </div>
  );
}
