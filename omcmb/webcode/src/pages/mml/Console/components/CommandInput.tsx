import { useState, useCallback, useEffect, useMemo } from 'react';
import { Button, Descriptions, Input, Modal, Space, Tabs, Typography } from 'antd';
import { PlayCircleOutlined, ReloadOutlined, SaveOutlined, WarningOutlined } from '@ant-design/icons';
import type { ConsoleDevice } from '../types';
import type { MMLCommand } from '@core/types/mml';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import { useDangerousCheck } from '@core/hooks/api/useMML';
import ParamFormRenderer, { type ParamFormChangePayload } from './ParamFormRenderer';
import ParamPathPanel, { type ParamPathChangePayload } from './ParamPathPanel';
import { resolveOperationType } from '../utils/resolveOperationType';

interface CommandInputProps {
  activeTab: 'control' | 'paramPath';
  canExecute: boolean;
  commandLineText: string;
  currentCommandLabel: string;
  executeButtonText: string;
  selectedDevices: ConsoleDevice[];
  selectedCommand: MMLCommand | null;
  onActiveTabChange: (tab: 'control' | 'paramPath') => void;
  onCommandLineChange: (value: string) => void;
  onParamChange: (values: Record<string, unknown>) => void;
  onExecute: () => void;
  onSaveScript?: () => void;
  onReset?: () => void;
  loading?: boolean;
}

export default function CommandInput({
  activeTab,
  canExecute,
  commandLineText,
  currentCommandLabel,
  executeButtonText,
  selectedDevices,
  selectedCommand,
  onActiveTabChange,
  onCommandLineChange,
  onParamChange,
  onExecute,
  onSaveScript,
  onReset,
  loading,
}: CommandInputProps) {
  const token = useThemeToken();
  const t = useT();
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [pendingDangerInfo, setPendingDangerInfo] = useState<{ name: string; desc: string } | null>(null);
  const [controlPayload, setControlPayload] = useState<ParamFormChangePayload>({});
  const [paramPathPayload, setParamPathPayload] = useState<ParamPathChangePayload>({
    operationType: 'LST',
    paramPaths: [],
    paramValues: [],
  });

  const { data: dangerousResult } = useDangerousCheck(selectedCommand?.commandCode ?? '');

  const currentOperationType = useMemo(() => resolveOperationType(selectedCommand), [selectedCommand]);
  const isParamPathTab = activeTab === 'paramPath';

  useEffect(() => {
    onActiveTabChange('control');
    setControlPayload({});
    setParamPathPayload({
      operationType: currentOperationType,
      paramPaths: [],
      paramValues: [],
    });
  }, [currentOperationType, onActiveTabChange, selectedCommand?.id]);

  useEffect(() => {
    onParamChange(activeTab === 'paramPath' ? paramPathPayload as Record<string, unknown> : controlPayload as Record<string, unknown>);
  }, [activeTab, controlPayload, onParamChange, paramPathPayload]);

  const doExecute = useCallback(() => {
    onExecute();
    setConfirmModalOpen(false);
    setPendingDangerInfo(null);
  }, [onExecute]);

  const handleExecute = useCallback(() => {
    // paramPath 模式 + 无命令 → 裸路径直接执行，不需要 selectedCommand。
    // 危险命令二次确认依赖 dangerousCheck（按 commandCode 查），无命令场景跳过。
    const isRawParamPathMode = activeTab === 'paramPath' && !selectedCommand;
    if (!canExecute) {
      return;
    }
    if (!isRawParamPathMode && !selectedCommand) {
      return;
    }

    if (selectedCommand && dangerousResult?.dangerous && dangerousResult.info) {
      setPendingDangerInfo({ name: dangerousResult.info.Name, desc: dangerousResult.info.Desc });
      setConfirmModalOpen(true);
      return;
    }

    doExecute();
  }, [activeTab, canExecute, dangerousResult, doExecute, selectedCommand]);

  const handleConfirmExecute = useCallback(() => {
    doExecute();
  }, [doExecute]);

  const handleCancelExecute = useCallback(() => {
    setConfirmModalOpen(false);
    setPendingDangerInfo(null);
  }, []);

  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      handleExecute();
    }
  }, [handleExecute]);

  const handleControlChange = useCallback((payload: ParamFormChangePayload) => {
    setControlPayload(payload);
    if (activeTab === 'control') {
      onParamChange(payload as Record<string, unknown>);
    }
  }, [activeTab, onParamChange]);

  const handleParamPathChange = useCallback((payload: ParamPathChangePayload) => {
    setParamPathPayload(payload);
    if (activeTab === 'paramPath') {
      onParamChange(payload as Record<string, unknown>);
    }
  }, [activeTab, onParamChange]);

  const tabItems = [
    {
      key: 'control',
      label: t('mml.console.controlPanel'),
      children: (
        <div
          className="no-scrollbar"
          style={{
            height: '100%',
            overflow: 'auto',
            padding: 12,
          }}
        >
          <ParamFormRenderer
            command={selectedCommand}
            value={(controlPayload.parameters as Record<string, string | number | boolean> | undefined) ?? undefined}
            onChange={handleControlChange}
          />
        </div>
      ),
    },
    {
      key: 'paramPath',
      label: t('mml.console.parameterPathCommand'),
      children: (
        <div
          className="no-scrollbar"
          style={{
            height: '100%',
            overflow: 'auto',
            padding: 12,
          }}
        >
          <ParamPathPanel command={selectedCommand} onChange={handleParamPathChange} />
        </div>
      ),
    },
  ];

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        background: token.colorBgContainer,
        borderRadius: 8,
        border: `1px solid ${token.colorBorderSecondary}`,
        overflow: 'hidden',
        boxShadow: '0 1px 4px rgba(0, 0, 0, 0.04)',
      }}
    >
      <div
        style={{
          padding: '10px 14px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          background: `linear-gradient(180deg, ${token.colorBgLayout} 0%, ${token.colorBgContainer} 100%)`,
        }}
      >
        <Descriptions column={1} size="small">
          <Descriptions.Item
            label={<span style={{ fontSize: 11, color: token.colorTextSecondary }}>{t('mml.console.currentCommand')}</span>}
          >
            {selectedCommand ? (
              <Typography.Text
                code
                style={{
                  fontSize: 12,
                  background: token.colorPrimaryBg,
                  border: `1px solid ${token.colorPrimaryBorder}`,
                  borderRadius: 4,
                  padding: '1px 6px',
                }}
              >
                {selectedCommand.commandCode}
              </Typography.Text>
            ) : (
              <span style={{ color: '#bfbfbf', fontSize: 11 }}>{t('mml.console.notSelected')}</span>
            )}
          </Descriptions.Item>
        </Descriptions>
      </div>

      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <Tabs
          className="mml-console-tabs"
          activeKey={activeTab}
          onChange={(tab) => onActiveTabChange(tab as 'control' | 'paramPath')}
          size="small"
          style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          tabBarStyle={{ padding: '0 12px', marginBottom: 0 }}
          items={tabItems}
        />
      </div>

      {!isParamPathTab && (
        <div
          style={{
            padding: '10px 14px',
            borderTop: `1px solid ${token.colorBorderSecondary}`,
            display: 'flex',
            gap: 10,
            background: token.colorBgLayout,
          }}
        >
          <Input
            size="small"
            value={commandLineText}
            onChange={(event) => onCommandLineChange(event.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('mml.console.commandInputPlaceholder')}
            prefix={
              <span
                style={{
                  color: token.colorPrimary,
                  fontFamily: 'monospace',
                  fontWeight: 'bold',
                }}
              >
                &gt;
              </span>
            }
            style={{
              flex: 1,
              borderRadius: 4,
              fontFamily: "'SFMono-Regular', Consolas, monospace",
            }}
          />
          <Button
            size="small"
            type="primary"
            icon={<PlayCircleOutlined />}
            onClick={handleExecute}
            loading={loading}
            disabled={!canExecute}
            style={{ borderRadius: 4, fontWeight: 500 }}
          >
            {executeButtonText}
          </Button>
        </div>
      )}

      <div
        style={{
          padding: '10px 14px',
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          justifyContent: isParamPathTab ? 'flex-end' : 'space-between',
          gap: 8,
          background: token.colorBgLayout,
        }}
      >
        {!isParamPathTab && (
          <Space size={8}>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              <span style={{ color: token.colorPrimary }}>{t('mml.console.deviceUnit', { count: selectedDevices.length })}</span>
              <span style={{ margin: '0 4px', color: token.colorBorder }}>·</span>
              {currentCommandLabel || <span style={{ color: '#bfbfbf' }}>{t('mml.console.noCommandSelected')}</span>}
              <span style={{ margin: '0 4px', color: token.colorBorder }}>·</span>
              {currentOperationType}
            </Typography.Text>
          </Space>
        )}
        <Space size={8}>
          {isParamPathTab && (
            <Button
              size="small"
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={handleExecute}
              loading={loading}
              disabled={!canExecute}
              style={{ borderRadius: 4 }}
            >
              {executeButtonText}
            </Button>
          )}
          {onReset && (
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={onReset}
              style={{ borderRadius: 4 }}
            >
              {t('common.reset')}
            </Button>
          )}
          {onSaveScript && !isParamPathTab && (
            // 未选择命令时按钮置灰：此处"命令"泛指左侧命令树选中项；由父层通过
            // 把 selectedCommand 作为可选依据。无选中命令时没有内容可保存，
            // 必须禁用以避免 message.warning('请先选择命令') 这类空操作。
            <Button
              size="small"
              icon={<SaveOutlined />}
              onClick={onSaveScript}
              disabled={!selectedCommand}
              style={{ borderRadius: 4 }}
            >
              {t('mml.console.saveScript')}
            </Button>
          )}
        </Space>
      </div>

      <Modal
        open={confirmModalOpen}
        onCancel={handleCancelExecute}
        onOk={handleConfirmExecute}
        okText={t('mml.console.confirmExecute')}
        cancelText={t('common.cancel')}
        okButtonProps={{
          danger: true,
          loading,
        }}
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <WarningOutlined style={{ color: '#faad14', fontSize: 20 }} />
            <span>{t('mml.console.confirmDangerousTitle')}</span>
          </div>
        }
        width={420}
        centered
      >
        <div style={{ padding: '16px 0' }}>
          <div
            style={{
              padding: 16,
              background: token.colorWarningBg,
              borderRadius: 8,
              marginBottom: 16,
            }}
          >
            <Typography.Text strong style={{ fontSize: 14, color: token.colorWarningText }}>
              {pendingDangerInfo?.name}
            </Typography.Text>
          </div>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            {pendingDangerInfo?.desc}
          </Typography.Text>
          <div style={{ marginTop: 16 }}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              <span style={{ color: token.colorPrimary, fontWeight: 500 }}>{t('mml.console.targetDeviceCount', { count: selectedDevices.length })}</span>
            </Typography.Text>
          </div>
          <div
            style={{
              marginTop: 16,
              padding: 12,
              background: token.colorBgLayout,
              borderRadius: 6,
              border: `1px solid ${token.colorBorderSecondary}`,
            }}
          >
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('mml.console.confirmContinue')}
            </Typography.Text>
          </div>
        </div>
      </Modal>
    </div>
  );
}
