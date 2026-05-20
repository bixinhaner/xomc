/**
 * @deprecated v2.4 D37（见 adjustment-plan §15）：本组件下线。
 *   原职责：① TextArea 显示拼接的 MML 命令文本 ② "DO" 执行按钮直接走 useExecuteStatements 旧通道
 *   下线原因：结构化通道（ConsoleActionBar + statementToStructured）已替代；用户不需要看到
 *           拼接的 MML 文本（后端 API 直接收 structured path[]），TextArea 是冗余 UI。
 *   去向：RightPanel.tsx 已移除引用；本文件保留 1 个 Sprint 后物理删除（防意外回滚需求）。
 */
import { useCallback, useMemo, type ChangeEvent } from 'react';
import { Input, Button, Modal, Spin, Alert, message, Space } from 'antd';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useExecuteStatements, useParseMML } from '@core/hooks/api/useMmlConsole';
import type { ParseResponse, Statement } from '@core/types/mmlConsole';
import type { MMLTask } from '@core/types/mml';
import { useT } from '@/hooks/useT';

const { TextArea } = Input;

export interface MmlEditorProps {
  onExecuted?: (task: MMLTask) => void;
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

export default function MmlEditor({ onExecuted }: MmlEditorProps) {
  const t = useT();
  const mmlText = useMmlConsoleStore((s) => s.mmlText);
  const parsePending = useMmlConsoleStore((s) => s.parsePending);
  const parseErrors = useMmlConsoleStore((s) => s.parseErrors);
  const statements = useMmlConsoleStore((s) => s.statements);
  const selectedDeviceSns = useMmlConsoleStore((s) => s.selectedDeviceSns);
  const setMmlTextDebounced = useMmlConsoleStore((s) => s.setMmlTextDebounced);

  const parseMutation = useParseMML();
  const executeMutation = useExecuteStatements();

  const parseFn = useCallback(
    (text: string): Promise<ParseResponse> => parseMutation.mutateAsync({ mmlString: text }),
    [parseMutation],
  );

  const handleTextChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setMmlTextDebounced(e.target.value, parseFn);
  };

  const runExecute = useCallback(async () => {
    try {
      const task = await executeMutation.mutateAsync({
        statements,
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

  const handleDoClick = () => {
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
    runExecute();
  };

  const errorMessages = useMemo(
    () =>
      parseErrors.map((pe) =>
        t('mml.console.parseError.syntax', {
          index: pe.statementIndex + 1,
          reason: pe.reason,
        }),
      ),
    [parseErrors, t],
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
        <TextArea
          value={mmlText}
          onChange={handleTextChange}
          rows={6}
          style={{ flex: 1, fontFamily: 'monospace' }}
          spellCheck={false}
        />
        <Space direction="vertical">
          <Button type="primary" onClick={handleDoClick} loading={executeMutation.isPending}>
            {t('mml.console.editor.execute')}
          </Button>
          {parsePending && (
            <Space size={4}>
              <Spin size="small" />
              <span style={{ fontSize: 12, color: '#888' }}>
                {t('mml.console.editor.parsing')}
              </span>
            </Space>
          )}
        </Space>
      </div>
      {errorMessages.length > 0 && (
        <Alert
          type="error"
          showIcon
          message={
            <ul style={{ margin: 0, paddingLeft: 18 }}>
              {errorMessages.map((m, i) => (
                <li key={i}>{m}</li>
              ))}
            </ul>
          }
        />
      )}
    </div>
  );
}
