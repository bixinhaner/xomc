import React, { useCallback, useState } from 'react';
import { Button, message, Tooltip } from 'antd';
import { CopyOutlined, MinusSquareOutlined, PlusSquareOutlined } from '@ant-design/icons';
import { useThemeToken, useIsDark } from '@/hooks/useThemeToken';

export interface JSONViewerProps {
  data: unknown;
  defaultExpanded?: boolean;
  maxHeight?: number | string;
  style?: React.CSSProperties;
}

interface JSONNodeProps {
  data: unknown;
  depth: number;
  keyName?: string;
  isLast?: boolean;
  defaultExpanded: boolean;
  colors: typeof LIGHT_COLORS;
}

const LIGHT_COLORS = {
  key: 'var(--color-primary-600)',
  string: '#52C41A',
  number: '#FA8C16',
  boolean: '#722ED1',
  null: '#8C8C8C',
  bracket: '#595959',
  comma: '#595959',
  toggle: '#bfbfbf',
};

const DARK_COLORS = {
  key: '#5CACFF',
  string: '#73D13D',
  number: '#FFA940',
  boolean: '#B37FEB',
  null: '#8C8C8C',
  bracket: '#B0B0B0',
  comma: '#B0B0B0',
  toggle: '#707070',
};

const INDENT = 18;

const JSONNode: React.FC<JSONNodeProps> = ({
  data,
  depth,
  keyName,
  isLast = true,
  defaultExpanded,
  colors,
}) => {
  const [collapsed, setCollapsed] = useState(!defaultExpanded && depth > 0);

  const isObject = typeof data === 'object' && data !== null && !Array.isArray(data);
  const isArray = Array.isArray(data);
  const isExpandable = isObject || isArray;
  const entries = isObject
    ? Object.entries(data as Record<string, unknown>)
    : isArray
      ? (data as unknown[]).map((v, i) => [String(i), v] as [string, unknown])
      : [];

  const renderValue = (val: unknown, _keyStr?: string, isLastItem = true, _itemDepth = depth): React.ReactNode => {
    if (val === null) return <span style={{ color: colors.null, fontFamily: 'monospace', fontSize: 13 }}>null{isLastItem ? '' : ','}</span>;
    if (typeof val === 'boolean')
      return (
        <span style={{ color: colors.boolean, fontFamily: 'monospace', fontSize: 13 }}>
          {String(val)}{isLastItem ? '' : ','}
        </span>
      );
    if (typeof val === 'number')
      return (
        <span style={{ color: colors.number, fontFamily: 'monospace', fontSize: 13 }}>
          {val}{isLastItem ? '' : ','}
        </span>
      );
    if (typeof val === 'string')
      return (
        <span style={{ color: colors.string, fontFamily: 'monospace', fontSize: 13 }}>
          "{val}"{isLastItem ? '' : ','}
        </span>
      );
    return null;
  };

  const isPrimitive = !isExpandable;

  const keyLabel = keyName !== undefined ? (
    <span>
      <span style={{ color: colors.key, fontFamily: 'monospace', fontSize: 13 }}>"{keyName}"</span>
      <span style={{ color: colors.bracket, fontFamily: 'monospace', fontSize: 13 }}>: </span>
    </span>
  ) : null;

  if (isPrimitive) {
    return (
      <div style={{ paddingLeft: depth * INDENT }}>
        {keyLabel}
        {renderValue(data, keyName, isLast, depth)}
      </div>
    );
  }

  const openBracket = isArray ? '[' : '{';
  const closeBracket = isArray ? ']' : '}';
  const count = entries.length;

  return (
    <div>
      <div
        style={{
          paddingLeft: depth * INDENT,
          display: 'flex',
          alignItems: 'center',
          gap: 2,
          cursor: 'pointer',
          userSelect: 'none',
        }}
        onClick={() => setCollapsed(!collapsed)}
      >
        <span style={{ marginRight: 2, color: colors.toggle, display: 'inline-flex', alignItems: 'center' }}>
          {collapsed
            ? <PlusSquareOutlined style={{ fontSize: 11 }} />
            : <MinusSquareOutlined style={{ fontSize: 11 }} />}
        </span>
        {keyLabel}
        <span style={{ color: colors.bracket, fontFamily: 'monospace', fontSize: 13 }}>
          {openBracket}
          {collapsed && (
            <span style={{ color: colors.null, fontSize: 12 }}>
              {count} {isArray ? 'items' : 'keys'}
            </span>
          )}
          {collapsed && closeBracket}
          {collapsed && !isLast && ','}
        </span>
      </div>

      {!collapsed && (
        <>
          {entries.map(([k, v], i) => {
            const isLastChild = i === entries.length - 1;
            if (typeof v === 'object' && v !== null) {
              return (
                <JSONNode
                  key={k}
                  data={v}
                  depth={depth + 1}
                  keyName={isArray ? undefined : k}
                  isLast={isLastChild}
                  defaultExpanded={defaultExpanded}
                  colors={colors}
                />
              );
            }
            return (
              <div key={k} style={{ paddingLeft: (depth + 1) * INDENT }}>
                {!isArray && (
                  <span>
                    <span style={{ color: colors.key, fontFamily: 'monospace', fontSize: 13 }}>
                      "{k}"
                    </span>
                    <span style={{ color: colors.bracket, fontFamily: 'monospace', fontSize: 13 }}>
                      :{' '}
                    </span>
                  </span>
                )}
                {renderValue(v, k, isLastChild, depth + 1)}
              </div>
            );
          })}
          <div style={{ paddingLeft: depth * INDENT }}>
            <span style={{ color: colors.bracket, fontFamily: 'monospace', fontSize: 13 }}>
              {closeBracket}
              {!isLast && ','}
            </span>
          </div>
        </>
      )}
    </div>
  );
};

const JSONViewer: React.FC<JSONViewerProps> = ({
  data,
  defaultExpanded = true,
  maxHeight = 400,
  style,
}) => {
  const token = useThemeToken();
  const isDark = useIsDark();
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS;

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(data, null, 2));
      void message.success('已复制到剪贴板');
    } catch {
      void message.error('复制失败');
    }
  }, [data]);

  return (
    <div
      style={{
        border: `1px solid ${token.colorBorderSecondary}`,
        borderRadius: 6,
        background: token.colorBgLayout,
        position: 'relative',
        ...style,
      }}
    >
      {/* Toolbar */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
          padding: '4px 8px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          background: token.colorBgContainer,
          borderRadius: '6px 6px 0 0',
        }}
      >
        <Tooltip title="复制 JSON">
          <Button
            size="small"
            icon={<CopyOutlined />}
            onClick={() => void handleCopy()}
          >
            复制
          </Button>
        </Tooltip>
      </div>

      {/* JSON content */}
      <div
        style={{
          padding: '12px 16px',
          overflowY: 'auto',
          maxHeight,
          overflowX: 'auto',
        }}
      >
        <JSONNode data={data} depth={0} defaultExpanded={defaultExpanded} colors={colors} />
      </div>
    </div>
  );
};

export default JSONViewer;
