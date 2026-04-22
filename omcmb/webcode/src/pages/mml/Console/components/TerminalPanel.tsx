import { forwardRef, useImperativeHandle, useRef, useEffect } from 'react';
import { Button, Space, Tooltip, Typography } from 'antd';
import { ClearOutlined, CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { message } from 'antd';
import type { TerminalLine } from '../types';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

const LINE_COLORS: Record<string, string> = {
  stdout: '#52C41A',
  stderr: '#F5222D',
  info: '#91D5FF',
  success: '#B7EB8F',
};

export interface TerminalPanelHandle {
  appendLine: (line: TerminalLine) => void;
  clear: () => void;
  scrollToBottom: () => void;
}

interface TerminalPanelProps {
  lines: TerminalLine[];
  onClear?: () => void;
  onDownload?: () => void;
}

const TerminalPanel = forwardRef<TerminalPanelHandle, TerminalPanelProps>(
  ({ lines, onClear, onDownload }, ref) => {
    const t = useT();
    const _token = useThemeToken();
    const containerRef = useRef<HTMLDivElement>(null);

    // 暴露方法
    useImperativeHandle(ref, () => ({
      appendLine: () => {
        // 由外部控制 lines，这里不需要实现
      },
      clear: () => {
        onClear?.();
      },
      scrollToBottom: () => {
        if (containerRef.current) {
          containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
      },
    }));

    // 自动滚动到底部
    useEffect(() => {
      if (containerRef.current) {
        containerRef.current.scrollTop = containerRef.current.scrollHeight;
      }
    }, [lines]);

    // 复制输出
    const handleCopy = async () => {
      const text = lines.map((l) => l.text).join('\n');
      try {
        await navigator.clipboard.writeText(text);
        void message.success(t('common.copiedToClipboard'));
      } catch {
        void message.error(t('common.copyFailed'));
      }
    };

    return (
      <div
        style={{
          background: '#0d1117',
          borderRadius: 8,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          boxShadow: 'inset 0 1px 0 rgba(255, 255, 255, 0.05)',
        }}
      >
        {/* 工具栏 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '8px 14px',
            background: 'linear-gradient(180deg, #21262d 0%, #161b22 100%)',
            borderBottom: '1px solid #30363d',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div
              style={{
                width: 12,
                height: 12,
                borderRadius: '50%',
                background: '#238636',
                boxShadow: '0 0 8px rgba(35, 134, 54, 0.5)',
              }}
            />
            <Typography.Text
              strong
              style={{ fontSize: 12, color: '#c9d1d9', letterSpacing: '0.5px' }}
            >
              {t('mml.console.terminalOutput')}
            </Typography.Text>
          </div>
          <Space size={4}>
            <Tooltip title={t('mml.console.copyOutput')}>
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={handleCopy}
                style={{
                  background: '#21262d',
                  border: '1px solid #30363d',
                  color: '#8b949e',
                  borderRadius: 4,
                }}
                className="terminal-btn"
              />
            </Tooltip>
            <Tooltip title={t('common.clear')}>
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={onClear}
                style={{
                  background: '#21262d',
                  border: '1px solid #30363d',
                  color: '#8b949e',
                  borderRadius: 4,
                }}
              />
            </Tooltip>
            {onDownload && (
              <Tooltip title={t('common.download')}>
                <Button
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={onDownload}
                  style={{
                    background: '#21262d',
                    border: '1px solid #30363d',
                    color: '#8b949e',
                    borderRadius: 4,
                  }}
                />
              </Tooltip>
            )}
          </Space>
        </div>

        {/* 内容区 */}
        <div
          ref={containerRef}
          className="no-scrollbar"
          style={{
            flex: 1,
            overflow: 'auto',
            padding: '12px 16px',
            fontFamily: "'JetBrains Mono', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace",
            fontSize: 12,
            lineHeight: 1.7,
            color: '#7ee787',
            background: 'linear-gradient(180deg, #0d1117 0%, #161b22 100%)',
          }}
        >
          {lines.length === 0 ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#484f58' }}>
              <span style={{ color: '#7ee787' }}>&gt;</span>
              <span>{t('mml.console.waitingForOutput')}</span>
            </div>
          ) : (
            lines.map((line, i) => (
              <div
                key={i}
                style={{
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                  padding: '2px 0',
                }}
              >
                {line.timestamp && (
                  <span
                    style={{
                      color: '#484f58',
                      marginRight: 10,
                      fontSize: 10,
                      fontFamily: 'monospace',
                    }}
                  >
                    [{line.timestamp}]
                  </span>
                )}
                <span
                  style={{
                    color: LINE_COLORS[line.type ?? 'stdout'] ?? '#7ee787',
                  }}
                >
                  {line.text}
                </span>
              </div>
            ))
          )}
        </div>
      </div>
    );
  }
);

TerminalPanel.displayName = 'TerminalPanel';

export default TerminalPanel;
