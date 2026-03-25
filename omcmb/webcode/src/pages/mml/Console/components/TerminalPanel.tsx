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
    const token = useThemeToken();
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
        void message.success('已复制到剪贴板');
      } catch {
        void message.error('复制失败');
      }
    };

    return (
      <div
        style={{
          background: '#1A1A1A',
          borderRadius: 6,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        }}
      >
        {/* 工具栏 */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '6px 12px',
            background: '#2A2A2A',
            borderBottom: '1px solid #333',
          }}
        >
          <Typography.Text strong style={{ fontSize: 12, color: '#fff' }}>
            {t('nav.mml.console')}
          </Typography.Text>
          <Space size={4}>
            <Tooltip title="复制输出">
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={handleCopy}
                style={{ background: '#3A3A3A', border: 'none', color: '#ccc' }}
              />
            </Tooltip>
            <Tooltip title="清空">
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={onClear}
                style={{ background: '#3A3A3A', border: 'none', color: '#ccc' }}
              />
            </Tooltip>
            {onDownload && (
              <Tooltip title="下载">
                <Button
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={onDownload}
                  style={{ background: '#3A3A3A', border: 'none', color: '#ccc' }}
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
            padding: '10px 14px',
            fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace",
            fontSize: 12,
            lineHeight: 1.6,
            color: '#52C41A',
          }}
        >
          {lines.length === 0 ? (
            <span style={{ color: '#555' }}>{'> 等待输出...'}</span>
          ) : (
            lines.map((line, i) => (
              <div key={i} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                {line.timestamp && (
                  <span style={{ color: '#666', marginRight: 8, fontSize: 10 }}>
                    [{line.timestamp}]
                  </span>
                )}
                <span style={{ color: LINE_COLORS[line.type ?? 'stdout'] ?? '#52C41A' }}>
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
