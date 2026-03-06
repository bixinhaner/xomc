import React, { useEffect, useImperativeHandle, useRef, useState } from 'react';
import { Button, Space, Tooltip } from 'antd';
import { ClearOutlined, CopyOutlined } from '@ant-design/icons';
import { message } from 'antd';

export interface TerminalLine {
  text: string;
  type?: 'stdout' | 'stderr' | 'info' | 'success';
  timestamp?: string;
}

export interface TerminalOutputHandle {
  appendLine: (line: TerminalLine) => void;
  clear: () => void;
}

export interface TerminalOutputProps {
  lines?: TerminalLine[];
  height?: number | string;
  autoScroll?: boolean;
  showTimestamp?: boolean;
  style?: React.CSSProperties;
}

const LINE_COLORS: Record<string, string> = {
  stdout: '#52C41A',
  stderr: '#F5222D',
  info: '#91D5FF',
  success: '#B7EB8F',
};

const TerminalOutput = React.forwardRef<TerminalOutputHandle, TerminalOutputProps>(
  (
    {
      lines: initialLines = [],
      height = 300,
      autoScroll = true,
      showTimestamp = false,
      style,
    },
    ref
  ) => {
    const [lines, setLines] = useState<TerminalLine[]>(initialLines);
    const containerRef = useRef<HTMLDivElement>(null);

    useImperativeHandle(ref, () => ({
      appendLine: (line: TerminalLine) => {
        setLines((prev) => [...prev, line]);
      },
      clear: () => {
        setLines([]);
      },
    }));

    useEffect(() => {
      setLines(initialLines);
    }, [initialLines]);

    useEffect(() => {
      if (autoScroll && containerRef.current) {
        containerRef.current.scrollTop = containerRef.current.scrollHeight;
      }
    }, [lines, autoScroll]);

    const handleCopy = async () => {
      const text = lines.map((l) => l.text).join('\n');
      try {
        await navigator.clipboard.writeText(text);
        void message.success('已复制到剪贴板');
      } catch {
        void message.error('复制失败');
      }
    };

    const handleClear = () => {
      setLines([]);
    };

    return (
      <div
        style={{
          background: '#1A1A1A',
          borderRadius: 6,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          ...style,
        }}
      >
        {/* Toolbar */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            padding: '4px 8px',
            background: '#2A2A2A',
            borderBottom: '1px solid #333',
            gap: 4,
          }}
        >
          <Space size={4}>
            <Tooltip title="复制输出">
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={() => void handleCopy()}
                style={{ background: '#3A3A3A', border: 'none', color: '#ccc' }}
              />
            </Tooltip>
            <Tooltip title="清空">
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={handleClear}
                style={{ background: '#3A3A3A', border: 'none', color: '#ccc' }}
              />
            </Tooltip>
          </Space>
        </div>

        {/* Content */}
        <div
          ref={containerRef}
          style={{
            height,
            overflowY: 'auto',
            padding: '10px 14px',
            fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace",
            fontSize: 13,
            lineHeight: 1.6,
            color: '#52C41A',
          }}
        >
          {lines.length === 0 ? (
            <span style={{ color: '#555' }}>{'> 等待输出...'}</span>
          ) : (
            lines.map((line, i) => (
              <div key={i} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                {showTimestamp && line.timestamp && (
                  <span style={{ color: '#666', marginRight: 8, fontSize: 11 }}>
                    [{line.timestamp}]
                  </span>
                )}
                <span
                  style={{
                    color: LINE_COLORS[line.type ?? 'stdout'] ?? '#52C41A',
                  }}
                >
                  {line.text}
                </span>
              </div>
            ))
          )}
          {/* Auto-scroll anchor */}
          <div />
        </div>
      </div>
    );
  }
);

TerminalOutput.displayName = 'TerminalOutput';

export default TerminalOutput;
